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
 * Parses a non-negative integer number of seconds from an action input.
 * @param {string} name input name, used in the error message
 * @param {string} value raw input value
 * @returns {number}
 */
export function parseSeconds(name, value) {
  const trimmed = String(value ?? "").trim();
  if (!/^\d+$/.test(trimmed)) {
    throw new Error(`input "${name}" must be a non-negative integer (seconds), got: "${value}"`);
  }
  return Number(trimmed);
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
 * Builds the per-job state map from the raw API responses.
 *
 * - Commit statuses are deduplicated by context. GitHub returns the latest
 *   status first, so the first one seen wins.
 * - A check run whose name was already seen (in statuses or earlier check
 *   runs) is skipped.
 * - A check run that is not completed is pending. A completed run is success
 *   for success/neutral, dropped for skipped, and error for anything else.
 *   A completed run with no conclusion yet is treated as pending.
 *
 * @param {Array<{context?: string, state?: string}>} statuses
 * @param {Array<{name?: string, status?: string, conclusion?: string | null}>} checkRuns
 * @returns {Map<string, string>} job name -> state, in discovery order
 */
export function collectJobStates(statuses, checkRuns) {
  /** @type {Map<string, string>} */
  const jobs = new Map();
  // Names seen so far, including skipped check runs, so an older run with the
  // same name cannot reappear.
  const seen = new Set();

  for (const s of statuses) {
    if (!s || typeof s.context !== "string" || typeof s.state !== "string") {
      throw new Error(
        `github combined status response is invalid context: ${s?.context}, status: ${s?.state}`,
      );
    }
    if (seen.has(s.context)) continue;
    seen.add(s.context);
    jobs.set(s.context, s.state);
  }

  for (const run of checkRuns) {
    if (!run || typeof run.name !== "string" || typeof run.status !== "string") {
      throw new Error(
        `github checkRun response is invalid name: ${run?.name}, status: ${run?.status}`,
      );
    }
    if (seen.has(run.name)) continue;
    seen.add(run.name);

    if (run.status !== CHECK_RUN_COMPLETED || run.conclusion == null) {
      jobs.set(run.name, State.PENDING);
      continue;
    }
    if (run.conclusion === SKIPPED_CONCLUSION) continue;
    jobs.set(run.name, SUCCESS_CONCLUSIONS.has(run.conclusion) ? State.SUCCESS : State.ERROR);
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
 * Turns the per-job state map into a verdict.
 * @param {Map<string, string>} jobs
 * @param {{ self: string, ignored: string[] }} opts
 * @returns {Verdict}
 */
export function evaluate(jobs, { self, ignored }) {
  const ignoredSet = new Set(ignored);
  const totalJobs = [];
  const completeJobs = [];
  const errJobs = [];
  let successCount = 0;

  for (const [job, state] of jobs) {
    // Ignored jobs and this job itself count as success regardless of state.
    if (ignoredSet.has(job) || job === self) {
      successCount++;
      continue;
    }
    totalJobs.push(job);
    if (state === State.SUCCESS) {
      completeJobs.push(job);
      successCount++;
    } else if (state === State.ERROR || state === State.FAILURE) {
      errJobs.push(job);
    }
  }

  return {
    totalJobs,
    completeJobs,
    errJobs,
    ignoredJobs: [...ignored],
    failed: errJobs.length > 0,
    succeeded: errJobs.length === 0 && successCount === jobs.size,
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
