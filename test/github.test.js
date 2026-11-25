import assert from "node:assert/strict";
import { describe, test } from "node:test";

import { PER_PAGE, createClient } from "../src/github.js";

const json = (body, init = {}) => new Response(JSON.stringify(body), { status: 200, ...init });

/** Fake fetch that serves paged statuses / check runs like the real API. */
function fakeGitHub({ statuses = [], checkRuns = [] }) {
  const calls = [];
  const fetch = async (url, init) => {
    calls.push({ url: String(url), init });
    const u = new URL(url);
    const page = Number(u.searchParams.get("page"));
    const perPage = Number(u.searchParams.get("per_page"));
    const slice = (arr) => arr.slice((page - 1) * perPage, page * perPage);
    if (u.pathname.endsWith("/status")) {
      // Like GitHub, total_count stays constant across pages.
      return json({ total_count: statuses.length, statuses: slice(statuses) });
    }
    if (u.pathname.endsWith("/check-runs")) {
      return json({ total_count: checkRuns.length, check_runs: slice(checkRuns) });
    }
    return json({ message: "Not Found" }, { status: 404, statusText: "Not Found" });
  };
  return { fetch, calls };
}

const statuses = (n) => Array.from({ length: n }, (_, i) => ({ context: `job-${i}`, state: "success" }));
const runs = (n) => Array.from({ length: n }, (_, i) => ({ name: `job-${i}`, status: "completed", conclusion: "neutral" }));

describe("listStatuses pagination", () => {
  for (const n of [0, 1, 99, 100, 112, 162, 587]) {
    test(`retrieves ${n} statuses and stops`, async () => {
      const gh = fakeGitHub({ statuses: statuses(n) });
      const client = createClient({ token: "t", fetch: gh.fetch });
      const got = await client.listStatuses("o", "r", "sha");
      assert.equal(got.length, n);
      // One request per full page, plus the final short (possibly empty) page.
      assert.equal(gh.calls.length, Math.floor(n / PER_PAGE) + 1);
    });
  }
});

describe("listCheckRuns pagination", () => {
  for (const n of [0, 1, 100, 162, 587]) {
    test(`retrieves ${n} check runs and stops`, async () => {
      const gh = fakeGitHub({ checkRuns: runs(n) });
      const client = createClient({ token: "t", fetch: gh.fetch });
      const got = await client.listCheckRuns("o", "r", "sha");
      assert.equal(got.length, n);
      assert.ok(gh.calls.length <= Math.ceil(n / PER_PAGE) + 1);
    });
  }
});

describe("requests", () => {
  test("uses the given API URL, auth header, and keeps '/' in refs", async () => {
    const gh = fakeGitHub({});
    const client = createClient({ token: "secret", apiUrl: "https://ghes.example.com/api/v3/", fetch: gh.fetch });
    await client.listStatuses("my-org", "my-repo", "feature/foo bar");
    const { url, init } = gh.calls[0];
    assert.equal(url, "https://ghes.example.com/api/v3/repos/my-org/my-repo/commits/feature/foo%20bar/status?per_page=100&page=1");
    assert.equal(init.headers.Authorization, "Bearer secret");
  });

  test("defaults to api.github.com", async () => {
    const gh = fakeGitHub({});
    await createClient({ token: "t", fetch: gh.fetch }).listCheckRuns("o", "r", "sha");
    assert.match(gh.calls[0].url, /^https:\/\/api\.github\.com\/repos\/o\/r\/commits\/sha\/check-runs\?/);
  });

  test("non-2xx responses throw with status and message", async () => {
    const fetch = async () => json({ message: "Bad credentials" }, { status: 401, statusText: "Unauthorized" });
    const client = createClient({ token: "t", fetch });
    await assert.rejects(client.listStatuses("o", "r", "sha"), /401 Unauthorized - Bad credentials/);
  });
});
