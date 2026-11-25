// Minimal GitHub REST client built on Node's global fetch.

export const PER_PAGE = 100;
const DEFAULT_API_URL = "https://api.github.com";

/**
 * @param {object} opts
 * @param {string} opts.token
 * @param {string} [opts.apiUrl] base REST URL; GITHUB_API_URL on GHES
 * @param {typeof fetch} [opts.fetch]
 */
export function createClient({ token, apiUrl, fetch: fetchImpl = globalThis.fetch }) {
  const base = (apiUrl || DEFAULT_API_URL).replace(/\/+$/, "");

  /**
   * @param {string} path path starting with "/"
   * @param {AbortSignal} [signal]
   */
  async function get(path, signal) {
    const res = await fetchImpl(`${base}${path}`, {
      headers: {
        Accept: "application/vnd.github+json",
        Authorization: `Bearer ${token}`,
        "User-Agent": "merge-gatekeeper",
        "X-GitHub-Api-Version": "2022-11-28",
      },
      signal,
    });
    if (!res.ok) {
      let message = "";
      try {
        message = (await res.json())?.message ?? "";
      } catch {
        // Body was not JSON; the status code is enough.
      }
      throw new Error(`GET ${path} failed: ${res.status} ${res.statusText}${message ? ` - ${message}` : ""}`);
    }
    return res.json();
  }

  // Escape each segment of the ref but keep "/" so branch names such as
  // "feature/foo" work (same as go-github's refURLEscape).
  const escapeRef = (ref) => ref.split("/").map(encodeURIComponent).join("/");
  const repoPath = (owner, repo, ref) =>
    `/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/commits/${escapeRef(ref)}`;

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
  };
}
