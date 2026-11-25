// Pure decision logic for Merge Gatekeeper. No I/O happens here, so every
// function can be unit tested with plain data.

export const State = Object.freeze({
  SUCCESS: "success",
  ERROR: "error",
  FAILURE: "failure",
  PENDING: "pending",
});

// https://docs.github.com/en/rest/checks/runs
const CHECK_RUN_COMPLETED = "completed";
const SUCCESS_CONCLUSIONS = new Set(["success", "neutral"]);
const SKIPPED_CONCLUSION = "skipped";

/**
 * Parses the comma-separated `ignored` input. Blank entries are dropped and
 * each name is trimmed.
 * @param {string | undefined} input
 * @returns {string[]}
 */
export function parseIgnored(input) {
  if (!input) return [];
  return input
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

/**
 * Parses a duration input into whole seconds.
 *
 * A bare number is seconds, as in v1 (e.g. "600"). A duration with units is
 * also accepted: any combination of hours, minutes and seconds in that order
 * (e.g. "90s", "10m", "1h", "1m30s").
 *
 * @param {string} name input name, used in the error message
 * @param {string} value raw input value
 * @returns {number} seconds
 */
export function parseSeconds(name, value) {
  const trimmed = String(value ?? "").trim();
  if (/^\d+$/.test(trimmed)) return Number(trimmed);

  const m = /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/.exec(trimmed);
  if (!trimmed || !m) {
    throw new Error(
      `input "${name}" must be a number of seconds or a duration such as "90s", "10m" or "1h30m", got: "${value}"`,
    );
  }
  const [, h = "0", min = "0", s = "0"] = m;
  return Number(h) * 3600 + Number(min) * 60 + Number(s);
}

/**
 * Splits "owner/repo" into its parts. Anything after the first slash is
 * treated as the repository name.
 * @param {string | undefined} str
 * @returns {{ owner: string, repo: string }}
 */
export function ownerAndRepository(str) {
  if (!str) return { owner: "", repo: "" };
  const [owner, ...rest] = str.split("/");
  return { owner, repo: rest.join("/") };
}

/**
 * @typedef {object} Job
 * @property {string} name   job name, used to match `self` and `ignored`
 * @property {string} label  unique name for logs; differs from `name` only when
 *                           several workflows have a job with the same name
 * @property {string} state  one of State
 */

/**
 * @typedef {object} Workflow
 * @property {number} id
 * @property {string} name
 * @property {string} path  workflow file, e.g. ".github/workflows/build.yaml"
 */

/** Slug of the GitHub App that creates check runs for Actions jobs. */
export const ACTIONS_APP_SLUG = "github-actions";

/** @param {{ id?: unknown }} run */
const runId = (run) => (typeof run.id === "number" ? run.id : -Infinity);

/**
 * Check suites of Actions check runs that are not in `workflows` yet.
 * @param {Array<{ app?: { slug?: string } | null, check_suite?: { id?: number } | null }>} checkRuns
 * @param {Map<number, Workflow>} workflows
 * @returns {number[]}
 */
export function unknownActionsSuites(checkRuns, workflows) {
  const ids = new Set();
  for (const run of checkRuns) {
    const suite = run?.check_suite?.id;
    if (run?.app?.slug === ACTIONS_APP_SLUG && suite != null && !workflows.has(suite)) ids.add(suite);
  }
  return [...ids];
}

/**
 * Builds the list of jobs to evaluate from the raw API responses.
 *
 * Commit statuses:
 * - Deduplicated by context. GitHub returns the latest status first, so the
 *   first one seen wins.
 *
 * Check runs:
 * - With `workflows` (check suite id -> workflow, from the Actions API), an
 *   Actions job is identified by its workflow and name. Jobs with the same
 *   name in different workflows are kept apart (#60), while new runs of the
 *   same workflow (re-runs, or a new event on the same commit) replace older
 *   ones. An Actions check suite that is not in `workflows` yet is kept apart
 *   by its check suite id until it is. Check runs from other apps are
 *   identified by app and name.
 * - Without `workflows` (no `actions: read` permission), a job is identified
 *   by name only, as in v2.0.0.
 * - For each job, only the latest run (highest id) counts.
 * - A check run with the same name as a commit status context is skipped.
 * - A check run that is not completed is pending. A completed run is success
 *   for success/neutral, dropped for skipped, and error for anything else.
 *   A completed run with no conclusion yet is treated as pending.
 *
 * @param {Array<{context?: string, state?: string}>} statuses
 * @param {Array<{id?: number, name?: string, status?: string, conclusion?: string | null, check_suite?: { id?: number } | null, app?: { id?: number, slug?: string } | null}>} checkRuns
 * @param {Map<number, Workflow> | null} [workflows]
 * @returns {Job[]} in discovery order
 */
export function collectJobStates(statuses, checkRuns, workflows = null) {
  /** @type {Job[]} */
  const jobs = [];
  const statusContexts = new Set();

  for (const s of statuses) {
    if (!s || typeof s.context !== "string" || typeof s.state !== "string") {
      throw new Error(
        `github combined status response is invalid context: ${s?.context}, status: ${s?.state}`,
      );
    }
    if (statusContexts.has(s.context)) continue;
    statusContexts.add(s.context);
    jobs.push({ name: s.context, label: s.context, state: s.state });
  }

  /**
   * Where a check run belongs, and how to show it when its name is shared.
   * @returns {{ key: string, short: string, long: string }}
   */
  const ownerOf = (run) => {
    if (!workflows) return { key: "", short: "", long: "" };
    const suite = run.check_suite?.id;
    const workflow = suite != null ? workflows.get(suite) : undefined;
    if (workflow) return { key: `workflow:${workflow.id}`, short: workflow.name, long: workflow.path || workflow.name };
    if (run.app?.slug === ACTIONS_APP_SLUG) {
      return { key: `suite:${suite ?? ""}`, short: `check suite ${suite}`, long: `check suite ${suite}` };
    }
    const app = run.app?.slug ?? String(run.app?.id ?? "");
    return { key: `app:${run.app?.id ?? ""}`, short: app, long: app };
  };

  // Latest check run per (owner, name), in order of first appearance.
  /** @type {Map<string, { run: any, owner: ReturnType<typeof ownerOf> }>} */
  const latest = new Map();
  for (const run of checkRuns) {
    if (!run || typeof run.name !== "string" || typeof run.status !== "string") {
      throw new Error(
        `github checkRun response is invalid name: ${run?.name}, status: ${run?.status}`,
      );
    }
    if (statusContexts.has(run.name)) continue;
    const owner = ownerOf(run);
    const key = `${owner.key}\u0000${run.name}`;
    const current = latest.get(key);
    if (!current || runId(run) > runId(current.run)) latest.set(key, { run, owner });
  }

  // A name shared by several owners gets the owner in its label, so logs can
  // tell them apart: the workflow name, or the workflow file when two
  // workflows also share a name.
  const count = (values) => values.reduce((m, v) => m.set(v, (m.get(v) ?? 0) + 1), new Map());
  const entries = [...latest.values()];
  const perName = count(entries.map((e) => e.run.name));
  const shortLabel = (e) => (perName.get(e.run.name) > 1 ? `${e.run.name} (${e.owner.short})` : e.run.name);
  const perShortLabel = count(entries.map(shortLabel));

  for (const e of entries) {
    const { run } = e;
    let state;
    if (run.status !== CHECK_RUN_COMPLETED || run.conclusion == null) {
      state = State.PENDING;
    } else if (run.conclusion === SKIPPED_CONCLUSION) {
      continue;
    } else {
      state = SUCCESS_CONCLUSIONS.has(run.conclusion) ? State.SUCCESS : State.ERROR;
    }
    let label = shortLabel(e);
    if (perShortLabel.get(label) > 1) label = `${run.name} (${e.owner.long})`;
    jobs.push({ name: run.name, label, state });
  }

  return jobs;
}

/**
 * @typedef {object} Verdict
 * @property {string[]} totalJobs     jobs that were evaluated (excludes self and ignored)
 * @property {string[]} completeJobs  jobs in success state
 * @property {string[]} errJobs       jobs in error/failure state
 * @property {string[]} ignoredJobs   the configured ignored list
 * @property {boolean} failed         true when any evaluated job failed
 * @property {boolean} succeeded      true when every job (including self/ignored) is done
 */

/**
 * Turns the job list into a verdict. `self` and `ignored` match job names, so
 * they apply to every job with that name, in any workflow.
 * @param {Job[]} jobs
 * @param {{ self: string, ignored: string[] }} opts
 * @returns {Verdict}
 */
export function evaluate(jobs, { self, ignored }) {
  const ignoredSet = new Set(ignored);
  const totalJobs = [];
  const completeJobs = [];
  const errJobs = [];
  let successCount = 0;

  for (const { name, label, state } of jobs) {
    // Ignored jobs and this job itself count as success regardless of state.
    if (ignoredSet.has(name) || name === self) {
      successCount++;
      continue;
    }
    totalJobs.push(label);
    if (state === State.SUCCESS) {
      completeJobs.push(label);
      successCount++;
    } else if (state === State.ERROR || state === State.FAILURE) {
      errJobs.push(label);
    }
  }

  return {
    totalJobs,
    completeJobs,
    errJobs,
    ignoredJobs: [...ignored],
    failed: errJobs.length > 0,
    succeeded: errJobs.length === 0 && successCount === jobs.length,
  };
}

/**
 * @param {Pick<Verdict, "totalJobs" | "completeJobs" | "errJobs" | "ignoredJobs">} v
 * @returns {string[]}
 */
export function incompleteJobs({ totalJobs, completeJobs = [], errJobs = [], ignoredJobs = [] }) {
  const done = new Set([...completeJobs, ...errJobs, ...ignoredJobs]);
  return totalJobs.filter((job) => !done.has(job));
}

function prettyPrintJobList(jobs) {
  if (jobs.length === 0) return "[]";
  return jobs.map((job) => `- ${job}`).join("\n");
}

/**
 * Renders the human readable summary, using GitHub Actions log groups.
 * Output is byte-for-byte compatible with v1.
 * @param {Pick<Verdict, "totalJobs" | "completeJobs" | "errJobs" | "ignoredJobs">} v
 * @returns {string}
 */
export function formatDetail(v) {
  const totalJobs = v.totalJobs ?? [];
  const completeJobs = v.completeJobs ?? [];
  const errJobs = v.errJobs ?? [];
  const ignoredJobs = v.ignoredJobs ?? [];
  const incomplete = incompleteJobs({ totalJobs, completeJobs, errJobs, ignoredJobs });

  return `${completeJobs.length} out of ${totalJobs.length}

Total job count:       ${totalJobs.length}
Completed job count:   ${completeJobs.length}
Incompleted job count: ${incomplete.length}
Failed job count:      ${errJobs.length}
Ignored job count:     ${ignoredJobs.length}

::group::Failed jobs
${prettyPrintJobList(errJobs)}
::endgroup::

::group::Completed jobs
${prettyPrintJobList(completeJobs)}
::endgroup::

::group::Incomplete jobs
${prettyPrintJobList(incomplete)}
::endgroup::

::group::Ignored jobs
${prettyPrintJobList(ignoredJobs)}
::endgroup::

::group::All jobs
${prettyPrintJobList(totalJobs)}
::endgroup::
`;
}
