// Minimal GitHub REST client built on Node's global fetch.
import { setTimeout as sleep } from "node:timers/promises";

export const PER_PAGE = 100;
const DEFAULT_API_URL = "https://api.github.com";

/**
 * Retry policy for transient failures: network errors, 5xx (except 501),
 * 429, and rate-limited 403s. Waits baseDelayMs, then doubles. Rate-limit
 * responses honour Retry-After / X-RateLimit-Reset, and wait one minute when
 * GitHub sends neither. A wait longer than maxDelayMs is not attempted, since
 * it would only burn the timeout.
 * https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api#handle-rate-limit-errors-appropriately
 */
export const DEFAULT_RETRY = Object.freeze({ retries: 3, baseDelayMs: 1000, maxDelayMs: 60_000 });
export const RATE_LIMIT_FALLBACK_MS = 60_000;

export class HttpError extends Error {
  /**
   * @param {string} message
   * @param {number} status
   * @param {Headers} [headers]
   */
  constructor(message, status, headers = new Headers()) {
    super(message);
    this.status = status;
    this.headers = headers;
  }
}

/**
 * True when the request failed because the token lacks `permission`.
 * GitHub answers 403 "Resource not accessible by integration" and lists the
 * permissions the endpoint needs in X-Accepted-GitHub-Permissions.
 * https://docs.github.com/en/rest/using-the-rest-api/troubleshooting-the-rest-api#resource-not-accessible
 * @param {unknown} err
 * @param {string} permission e.g. "actions=read"
 */
export function isMissingPermission(err, permission) {
  if (!(err instanceof HttpError) || err.status !== 403) return false;
  const accepted = err.headers.get("x-accepted-github-permissions") ?? "";
  return accepted
    .split(/[;,]/)
    .map((p) => p.trim())
    .includes(permission);
}

/**
 * How long to wait before retrying this response, or null if it should not be
 * retried.
 * @param {Response} res
 * @param {number} attempt 0 for the first retry
 * @param {{ baseDelayMs: number }} retry
 * @param {() => number} now
 * @returns {number | null} milliseconds
 */
export function retryDelay(res, attempt, retry, now = Date.now) {
  const retryAfter = Number(res.headers.get("retry-after"));
  const rateLimited = res.headers.get("x-ratelimit-remaining") === "0";
  const backoff = retry.baseDelayMs * 2 ** attempt;

  if (res.status === 429 || (res.status === 403 && (rateLimited || res.headers.has("retry-after")))) {
    if (Number.isFinite(retryAfter) && retryAfter > 0) return retryAfter * 1000;
    const reset = Number(res.headers.get("x-ratelimit-reset"));
    if (rateLimited && Number.isFinite(reset) && reset > 0) return Math.max(0, reset * 1000 - now());
    // No hint from GitHub: wait at least one minute, as its docs ask.
    // Retrying sooner can extend the limit.
    return RATE_LIMIT_FALLBACK_MS;
  }
  if (res.status >= 500 && res.status !== 501) return backoff;
  return null;
}

/**
 * @param {object} opts
 * @param {string} opts.token
 * @param {string} [opts.apiUrl] base REST URL; GITHUB_API_URL on GHES
 * @param {typeof fetch} [opts.fetch]
 * @param {(ms: number, signal?: AbortSignal) => Promise<unknown>} [opts.sleep]
 * @param {{ retries: number, baseDelayMs: number, maxDelayMs: number }} [opts.retry]
 * @param {(message: string) => void} [opts.warn] called before each retry
 */
export function createClient({
  token,
  apiUrl,
  fetch: fetchImpl = globalThis.fetch,
  sleep: wait = (ms, signal) => sleep(ms, undefined, { signal }),
  retry = DEFAULT_RETRY,
  warn = () => {},
}) {
  const base = (apiUrl || DEFAULT_API_URL).replace(/\/+$/, "");

  async function fetchOnce(path, signal) {
    return fetchImpl(`${base}${path}`, {
      headers: {
        Accept: "application/vnd.github+json",
        Authorization: `Bearer ${token}`,
        "User-Agent": "merge-gatekeeper",
        "X-GitHub-Api-Version": "2022-11-28",
      },
      signal,
    });
  }

  async function toError(path, res) {
    let message = "";
    try {
      message = (await res.json())?.message ?? "";
    } catch {
      // Body was not JSON; the status code is enough.
    }
    return new HttpError(
      `GET ${path} failed: ${res.status} ${res.statusText}${message ? ` - ${message}` : ""}`,
      res.status,
      res.headers,
    );
  }

  /**
   * GET with retries for transient failures.
   * @param {string} path path starting with "/"
   * @param {AbortSignal} [signal]
   */
  async function get(path, signal) {
    for (let attempt = 0; ; attempt++) {
      let res;
      let delay;
      let reason;
      try {
        res = await fetchOnce(path, signal);
      } catch (err) {
        // Aborted by the timeout: stop, do not retry.
        if (signal?.aborted) throw err;
        if (attempt >= retry.retries) throw new Error(`GET ${path} failed: ${err?.cause?.message ?? err?.message ?? err}`);
        delay = retry.baseDelayMs * 2 ** attempt;
        reason = `network error: ${err?.cause?.message ?? err?.message ?? err}`;
      }

      if (res) {
        if (res.ok) return res.json();
        delay = retryDelay(res, attempt, retry);
        if (delay === null || attempt >= retry.retries || delay > retry.maxDelayMs) {
          throw await toError(path, res);
        }
        reason = `${res.status} ${res.statusText}`;
        await res.body?.cancel();
      }

      warn(
        `GitHub API request failed (${reason}). Retrying in ${Math.ceil(delay / 1000)}s ` +
          `(retry ${attempt + 1} of ${retry.retries}): GET ${path}`,
      );
      await wait(delay, signal);
    }
  }

  // Escape each segment of the ref but keep "/" so branch names such as
  // "feature/foo" work (same as go-github's refURLEscape).
  const escapeRef = (ref) => ref.split("/").map(encodeURIComponent).join("/");
  const repoBase = (owner, repo) => `/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`;
  const repoPath = (owner, repo, ref) => `${repoBase(owner, repo)}/commits/${escapeRef(ref)}`;

  return {
    /**
     * All commit statuses for a ref, latest first per context.
     * Paging stops on a short page. total_count on this endpoint is the
     * number of unique contexts, not the number of statuses, so it cannot be
     * used as a stop condition.
     */
    async listStatuses(owner, repo, ref, signal) {
      const all = [];
      for (let page = 1; ; page++) {
        const body = await get(`${repoPath(owner, repo, ref)}/status?per_page=${PER_PAGE}&page=${page}`, signal);
        const statuses = body?.statuses ?? [];
        all.push(...statuses);
        if (statuses.length < PER_PAGE) break;
      }
      return all;
    },

    /** All check runs for a ref. */
    async listCheckRuns(owner, repo, ref, signal) {
      const all = [];
      for (let page = 1; ; page++) {
        const body = await get(`${repoPath(owner, repo, ref)}/check-runs?per_page=${PER_PAGE}&page=${page}`, signal);
        const runs = body?.check_runs ?? [];
        all.push(...runs);
        if (runs.length < PER_PAGE || all.length >= (body?.total_count ?? 0)) break;
      }
      return all;
    },

    /**
     * All workflow runs for a commit SHA. Needs the `actions: read`
     * permission on private repositories.
     */
    async listWorkflowRuns(owner, repo, sha, signal) {
      const all = [];
      for (let page = 1; ; page++) {
        const body = await get(
          `${repoBase(owner, repo)}/actions/runs?head_sha=${encodeURIComponent(sha)}&per_page=${PER_PAGE}&page=${page}`,
          signal,
        );
        const runs = body?.workflow_runs ?? [];
        all.push(...runs);
        if (runs.length < PER_PAGE || all.length >= (body?.total_count ?? 0)) break;
      }
      return all;
    },
  };
}
