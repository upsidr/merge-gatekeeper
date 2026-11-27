import { setTimeout as sleep } from "node:timers/promises";

import {
  collectJobStates,
  evaluate,
  formatDetail,
  ownerAndRepository,
  parseIgnored,
  parseSeconds,
  unknownActionsSuites,
} from "./core.js";
import { createClient, isMissingPermission } from "./github.js";

/**
 * Reads action inputs and the runner environment into a config object.
 * Inputs come from INPUT_<NAME> with the name upper-cased and hyphens kept
 * (e.g. INPUT_INITIAL-DELAY), which is how the runner passes them to node
 * actions.
 * @param {NodeJS.ProcessEnv} env
 */
export function readConfig(env) {
  const input = (name, fallback = "") => {
    const v = env[`INPUT_${name.toUpperCase()}`];
    return v === undefined || v === "" ? fallback : v.trim();
  };

  const { owner, repo } = ownerAndRepository(env.GITHUB_REPOSITORY);
  const config = {
    token: input("token"),
    self: input("self", "merge-gatekeeper"),
    interval: parseSeconds("interval", input("interval", "5")),
    timeout: parseSeconds("timeout", input("timeout", "600")),
    initialDelay: parseSeconds("initial-delay", input("initial-delay", "0")),
    ignored: parseIgnored(input("ignored")),
    // The default `ref` is the PR head SHA, which is empty outside pull_request
    // events (e.g. push, merge_group). Fall back to the commit that triggered
    // the workflow.
    ref: input("ref") || (env.GITHUB_SHA ?? "").trim(),
    owner,
    repo,
    apiUrl: env.GITHUB_API_URL || "https://api.github.com",
  };

  const errs = [];
  if (!config.token) errs.push("github token is empty");
  if (!config.owner || !config.repo) {
    errs.push(`github owner or repository is empty. GITHUB_REPOSITORY: "${env.GITHUB_REPOSITORY ?? ""}"`);
  }
  if (!config.ref) errs.push("reference of repository is empty");
  // Each check makes at least two API calls; without a pause between checks
  // the action would hit GitHub's secondary rate limits right away.
  if (config.interval < 1) errs.push(`input "interval" must be at least 1 second, got: "${input("interval")}"`);
  if (errs.length > 0) throw new Error(errs.join("\n"));
  return config;
}

class TimeoutError extends Error {}

export const NO_ACTIONS_PERMISSION_WARNING =
  "Could not read workflow runs: the token has no `actions: read` permission. " +
  "Jobs with the same name in different workflows are checked as one job (only the latest run counts). " +
  "Add `actions: read` to the job's permissions to check them separately.";

/**
 * Keeps a map of check suite id -> workflow, filled from the Actions API as
 * new check suites show up. Becomes null (and stays null) when the token
 * cannot read workflow runs; jobs are then identified by name only.
 */
function workflowResolver(client, config, logger) {
  /** @type {Map<number, import("./core.js").Workflow> | null} */
  let workflows = new Map();
  return async (checkRuns, signal) => {
    if (!workflows || unknownActionsSuites(checkRuns, workflows).length === 0) return workflows;
    // The `ref` input may be a branch name; the Actions API needs the SHA.
    const sha = checkRuns.find((r) => typeof r.head_sha === "string")?.head_sha;
    if (!sha) return workflows;
    try {
      for (const wr of await client.listWorkflowRuns(config.owner, config.repo, sha, signal)) {
        if (wr?.check_suite_id == null || wr.workflow_id == null) continue;
        workflows.set(wr.check_suite_id, { id: wr.workflow_id, name: wr.name ?? "", path: wr.path ?? "" });
      }
    } catch (err) {
      if (!isMissingPermission(err, "actions=read")) throw err;
      // Shown as an annotation on the run, so it is noticed without opening the log.
      logger.log(`::warning::${NO_ACTIONS_PERMISSION_WARNING}`);
      workflows = null;
    }
    return workflows;
  };
}

/**
 * Polls GitHub until every job is done, any job fails, or the timeout passes.
 *
 * @param {ReturnType<typeof readConfig>} config
 * @param {object} [deps] injectable for tests
 * @param {ReturnType<typeof createClient>} [deps.client]
 * @param {(ms: number, signal?: AbortSignal) => Promise<void>} [deps.sleep]
 * @param {() => number} [deps.now]
 * @param {{ log: (s: string) => void, warn: (s: string) => void }} [deps.logger]
 * @returns {Promise<{ ok: true } | { ok: false, message: string }>}
 */
export async function run(config, deps = {}) {
  const logger = deps.logger ?? { log: console.log, warn: console.error };
  const client =
    deps.client ?? createClient({ token: config.token, apiUrl: config.apiUrl, warn: (m) => logger.warn(m) });
  const wait = deps.sleep ?? ((ms, signal) => sleep(ms, undefined, { signal }));
  const now = deps.now ?? Date.now;
  const resolveWorkflows = workflowResolver(client, config, logger);

  if (config.initialDelay > 0) {
    logger.log(`Waiting ${config.initialDelay} seconds before the first validation...`);
    await wait(config.initialDelay * 1000);
  }

  // The deadline starts after the initial delay, so the delay is not part of the timeout.
  const timeoutMs = config.timeout * 1000;
  const deadline = now() + timeoutMs;
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(new TimeoutError()), timeoutMs);
  timer.unref?.();

  try {
    while (true) {
      logger.log(`Start processing validator: ${config.self}....`);
      const [statuses, checkRuns] = await Promise.all([
        client.listStatuses(config.owner, config.repo, config.ref, controller.signal),
        client.listCheckRuns(config.owner, config.repo, config.ref, controller.signal),
      ]);
      const workflows = await resolveWorkflows(checkRuns, controller.signal);
      const verdict = evaluate(collectJobStates(statuses, checkRuns, workflows), {
        self: config.self,
        ignored: config.ignored,
      });
      logger.log(formatDetail(verdict));
      logger.log(`Finish validator: ${config.self} processing.`);

      if (verdict.failed) {
        return { ok: false, message: `validation failed, failed jobs: ${verdict.errJobs.join(", ")}` };
      }
      if (verdict.succeeded) {
        logger.log("All validations were successful!");
        return { ok: true };
      }

      if (now() + config.interval * 1000 >= deadline) {
        throw new TimeoutError();
      }
      logger.warn("");
      logger.warn("  WARNING: Validation is yet to be completed. This is most likely due to some other jobs still running.");
      logger.warn(`           Waiting for ${config.interval} seconds before retrying.\n`);
      await wait(config.interval * 1000, controller.signal);
    }
  } catch (err) {
    if (err instanceof TimeoutError || controller.signal.aborted) {
      return {
        ok: false,
        message: `validation timed out after ${config.timeout} seconds; some jobs are still incomplete`,
      };
    }
    throw err;
  } finally {
    clearTimeout(timer);
  }
}
