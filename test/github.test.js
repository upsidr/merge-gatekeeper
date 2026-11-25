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

describe("retries (#64)", () => {
  const ok = () => json({ total_count: 0, statuses: [] });
  const fail = (status, headers = {}) => () =>
    json({ message: "nope" }, { status, statusText: `status ${status}`, headers });

  /** fetch that plays the given responses in order; functions are called, errors are thrown. */
  function sequence(...steps) {
    let i = 0;
    const fetch = async () => {
      const step = steps[Math.min(i++, steps.length - 1)];
      if (step instanceof Error) throw step;
      return step();
    };
    return { fetch, calls: () => i };
  }

  function client(fetch) {
    const sleeps = [];
    const warnings = [];
    const c = createClient({
      token: "t",
      fetch,
      sleep: async (ms) => sleeps.push(ms),
      warn: (m) => warnings.push(m),
    });
    return { c, sleeps, warnings };
  }

  for (const status of [500, 502, 503, 504]) {
    test(`retries ${status} with exponential backoff, then succeeds`, async () => {
      const seq = sequence(fail(status), fail(status), ok);
      const { c, sleeps, warnings } = client(seq.fetch);
      assert.deepEqual(await c.listStatuses("o", "r", "sha"), []);
      assert.equal(seq.calls(), 3);
      assert.deepEqual(sleeps, [1000, 2000]);
      assert.equal(warnings.length, 2);
      assert.match(warnings[0], new RegExp(`${status}.*retry 1 of 3`));
    });
  }

  test("retries network errors (e.g. EOF, connection reset)", async () => {
    const err = new TypeError("fetch failed", { cause: new Error("other side closed") });
    const seq = sequence(err, ok);
    const { c, warnings } = client(seq.fetch);
    assert.deepEqual(await c.listStatuses("o", "r", "sha"), []);
    assert.match(warnings[0], /network error: other side closed/);
  });

  test("gives up after 3 retries and reports the last error", async () => {
    const seq = sequence(fail(502));
    const { c, sleeps } = client(seq.fetch);
    await assert.rejects(c.listStatuses("o", "r", "sha"), /502/);
    assert.equal(seq.calls(), 4);
    assert.deepEqual(sleeps, [1000, 2000, 4000]);
  });

  test("gives up on network errors after 3 retries", async () => {
    const seq = sequence(new TypeError("fetch failed", { cause: new Error("ECONNRESET") }));
    const { c } = client(seq.fetch);
    await assert.rejects(c.listStatuses("o", "r", "sha"), /ECONNRESET/);
    assert.equal(seq.calls(), 4);
  });

  for (const status of [400, 401, 404, 422, 501]) {
    test(`does not retry ${status}`, async () => {
      const seq = sequence(fail(status), ok);
      const { c, sleeps } = client(seq.fetch);
      await assert.rejects(c.listStatuses("o", "r", "sha"), new RegExp(String(status)));
      assert.equal(seq.calls(), 1);
      assert.deepEqual(sleeps, []);
    });
  }

  test("does not retry a plain 403 (missing permission)", async () => {
    const seq = sequence(fail(403), ok);
    const { c } = client(seq.fetch);
    await assert.rejects(c.listStatuses("o", "r", "sha"), /403/);
    assert.equal(seq.calls(), 1);
  });

  test("honours Retry-After on 429 and secondary rate limits", async () => {
    for (const status of [429, 403]) {
      const seq = sequence(fail(status, { "retry-after": "7" }), ok);
      const { c, sleeps } = client(seq.fetch);
      await c.listStatuses("o", "r", "sha");
      assert.deepEqual(sleeps, [7000], `status ${status}`);
    }
  });

  test("does not wait for a rate limit reset that is too far away", async () => {
    const reset = String(Math.floor(Date.now() / 1000) + 3600);
    const seq = sequence(fail(403, { "x-ratelimit-remaining": "0", "x-ratelimit-reset": reset }), ok);
    const { c, sleeps } = client(seq.fetch);
    await assert.rejects(c.listStatuses("o", "r", "sha"), /403/);
    assert.deepEqual(sleeps, []);
  });

  test("does not retry once the timeout aborted the request", async () => {
    const controller = new AbortController();
    controller.abort();
    const seq = sequence(new DOMException("aborted", "AbortError"), ok);
    const { c } = client(seq.fetch);
    await assert.rejects(c.listStatuses("o", "r", "sha", controller.signal), /aborted/);
    assert.equal(seq.calls(), 1);
  });
});
