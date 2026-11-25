import { setTimeout as sleep } from "node:timers/promises";

import {
  collectJobStates,
  evaluate,
  formatDetail,
  ownerAndRepository,
  parseIgnored,
  parseSeconds,
} from "./core.js";
import { createClient } from "./github.js";

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
    ref: input("ref"),
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
  if (errs.length > 0) throw new Error(errs.join("\n"));
  return config;
}

class TimeoutError extends Error {}

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
  const client = deps.client ?? createClient({ token: config.token, apiUrl: config.apiUrl });
  const wait = deps.sleep ?? ((ms, signal) => sleep(ms, undefined, { signal }));
  const now = deps.now ?? Date.now;
  const logger = deps.logger ?? { log: console.log, warn: console.error };

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
      const verdict = evaluate(collectJobStates(statuses, checkRuns), {
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
