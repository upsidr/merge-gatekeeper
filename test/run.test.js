import assert from "node:assert/strict";
import { describe, test } from "node:test";

import { HttpError } from "../src/github.js";
import { NO_ACTIONS_PERMISSION_WARNING, readConfig, run } from "../src/run.js";

const baseEnv = {
  "INPUT_TOKEN": "t",
  "INPUT_REF": "sha",
  "GITHUB_REPOSITORY": "upsidr/merge-gatekeeper",
};

describe("readConfig", () => {
  test("applies action.yml defaults", () => {
    assert.deepEqual(readConfig(baseEnv), {
      token: "t",
      self: "merge-gatekeeper",
      interval: 5,
      timeout: 600,
      initialDelay: 0,
      ignored: [],
      ref: "sha",
      owner: "upsidr",
      repo: "merge-gatekeeper",
      apiUrl: "https://api.github.com",
    });
  });

  test("reads hyphenated inputs and GITHUB_API_URL", () => {
    const c = readConfig({
      ...baseEnv,
      "INPUT_SELF": "gate",
      "INPUT_INTERVAL": "10",
      "INPUT_TIMEOUT": "30",
      "INPUT_INITIAL-DELAY": "7",
      "INPUT_IGNORED": "a, b",
      "GITHUB_API_URL": "https://ghes.example.com/api/v3",
    });
    assert.equal(c.self, "gate");
    assert.equal(c.interval, 10);
    assert.equal(c.timeout, 30);
    assert.equal(c.initialDelay, 7);
    assert.deepEqual(c.ignored, ["a", "b"]);
    assert.equal(c.apiUrl, "https://ghes.example.com/api/v3");
  });

  test("falls back to GITHUB_SHA when ref is empty (push, merge_group)", () => {
    assert.equal(readConfig({ ...baseEnv, INPUT_REF: "", GITHUB_SHA: "abc123" }).ref, "abc123");
    assert.equal(readConfig({ ...baseEnv, INPUT_REF: "sha", GITHUB_SHA: "abc123" }).ref, "sha");
  });

  test("accepts durations for interval, timeout and initial-delay", () => {
    const c = readConfig({ ...baseEnv, INPUT_INTERVAL: "30s", INPUT_TIMEOUT: "25m", "INPUT_INITIAL-DELAY": "1m" });
    assert.equal(c.interval, 30);
    assert.equal(c.timeout, 1500);
    assert.equal(c.initialDelay, 60);
  });

  test("errors on missing ref, token, or repository", () => {
    assert.throws(() => readConfig({ ...baseEnv, INPUT_REF: "" }), /reference of repository is empty/);
    assert.throws(() => readConfig({ ...baseEnv, INPUT_TOKEN: "" }), /token is empty/);
    assert.throws(() => readConfig({ ...baseEnv, GITHUB_REPOSITORY: "" }), /owner or repository is empty/);
  });

  test("errors when interval is 0, to avoid hammering the API", () => {
    for (const v of ["0", "0s", "0m"]) {
      assert.throws(() => readConfig({ ...baseEnv, INPUT_INTERVAL: v }), /interval" must be at least 1 second/);
    }
    assert.equal(readConfig({ ...baseEnv, INPUT_INTERVAL: "1" }).interval, 1);
  });

  test("errors on non-numeric seconds", () => {
    assert.throws(() => readConfig({ ...baseEnv, INPUT_TIMEOUT: "ten minutes" }), /timeout/);
  });
});

/** Fake client that returns a scripted sequence of snapshots, one per poll. */
function scriptedClient(snapshots, { listWorkflowRuns } = {}) {
  let i = 0;
  let current;
  return {
    polls: () => i,
    listWorkflowRuns,
    async listStatuses() {
      current = snapshots[Math.min(i, snapshots.length - 1)];
      i++;
      return current.statuses ?? [];
    },
    async listCheckRuns() {
      return current.checkRuns ?? [];
    },
  };
}

function fakeClock() {
  let t = 0;
  const sleeps = [];
  return {
    sleeps,
    now: () => t,
    sleep: async (ms) => {
      sleeps.push(ms);
      t += ms;
    },
  };
}

const silent = { log() {}, warn() {} };
const config = (over = {}) => ({ ...readConfig(baseEnv), self: "self-job", ...over });

describe("run", () => {
  test("succeeds once all jobs complete", async () => {
    const client = scriptedClient([
      { statuses: [{ context: "job", state: "pending" }] },
      { statuses: [{ context: "job", state: "success" }] },
    ]);
    const clock = fakeClock();
    const res = await run(config({ interval: 5 }), { client, ...clock, logger: silent });
    assert.deepEqual(res, { ok: true });
    assert.equal(client.polls(), 2);
    assert.deepEqual(clock.sleeps, [5000]);
  });

  test("fails immediately on a failed job and lists it", async () => {
    const client = scriptedClient([
      { checkRuns: [{ name: "build", status: "completed", conclusion: "failure" }, { name: "lint", status: "in_progress" }] },
    ]);
    const res = await run(config(), { client, ...fakeClock(), logger: silent });
    assert.equal(res.ok, false);
    assert.match(res.message, /build/);
    assert.doesNotMatch(res.message, /lint/);
    assert.equal(client.polls(), 1);
  });

  test("times out when jobs never complete", async () => {
    const client = scriptedClient([{ statuses: [{ context: "job", state: "pending" }] }]);
    const clock = fakeClock();
    const res = await run(config({ interval: 5, timeout: 12 }), { client, ...clock, logger: silent });
    assert.equal(res.ok, false);
    assert.match(res.message, /timed out after 12 seconds/);
    // Polls at t=0, 5, 10; the next one would be at 15 > 12.
    assert.equal(client.polls(), 3);
  });

  test("initial delay runs first and does not count toward the timeout", async () => {
    const client = scriptedClient([
      { statuses: [{ context: "job", state: "pending" }] },
      { statuses: [{ context: "job", state: "success" }] },
    ]);
    const clock = fakeClock();
    const res = await run(config({ interval: 5, timeout: 6, initialDelay: 100 }), { client, ...clock, logger: silent });
    assert.deepEqual(res, { ok: true });
    assert.deepEqual(clock.sleeps, [100000, 5000]);
  });

  test("API errors propagate", async () => {
    const client = {
      async listStatuses() {
        throw new Error("boom");
      },
      async listCheckRuns() {
        return [];
      },
    };
    await assert.rejects(run(config(), { client, ...fakeClock(), logger: silent }), /boom/);
  });

  describe("jobs with the same name in different workflows", () => {
    const ACTIONS = { slug: "github-actions", id: 15368 };
    const job = (name, conclusion, id, suite) => ({
      id,
      name,
      head_sha: "abc123",
      status: conclusion ? "completed" : "in_progress",
      conclusion,
      check_suite: { id: suite },
      app: ACTIONS,
    });
    const logger = () => {
      const lines = [];
      return { lines, log: (s) => lines.push(s), warn: (s) => lines.push(s) };
    };

    test("looks up workflows once per new check suite and keeps the jobs apart", async () => {
      const calls = [];
      const client = scriptedClient(
        [
          { checkRuns: [job("build", "success", 1, 100), job("build", null, 2, 200)] },
          { checkRuns: [job("build", "success", 1, 100), job("build", "failure", 2, 200)] },
        ],
        {
          async listWorkflowRuns(owner, repo, sha) {
            calls.push(sha);
            return [
              { check_suite_id: 100, workflow_id: 1, name: "CI", path: ".github/workflows/ci.yaml" },
              { check_suite_id: 200, workflow_id: 2, name: "Deploy", path: ".github/workflows/deploy.yaml" },
            ];
          },
        },
      );
      const res = await run(config(), { client, ...fakeClock(), logger: silent });
      assert.deepEqual(res, { ok: false, message: "validation failed, failed jobs: build (Deploy)" });
      // Both suites were known after the first poll.
      assert.deepEqual(calls, ["abc123"]);
    });

    test("without actions: read, warns once and identifies jobs by name", async () => {
      let calls = 0;
      const client = scriptedClient(
        [
          { checkRuns: [job("Dependabot", "cancelled", 1, 100), job("Dependabot", null, 2, 101)] },
          { checkRuns: [job("Dependabot", "cancelled", 1, 100), job("Dependabot", "success", 2, 101)] },
        ],
        {
          async listWorkflowRuns() {
            calls++;
            throw new HttpError("403", 403, new Headers({ "x-accepted-github-permissions": "actions=read" }));
          },
        },
      );
      const log = logger();
      const res = await run(config(), { client, ...fakeClock(), logger: log });
      assert.deepEqual(res, { ok: true });
      assert.equal(calls, 1);
      assert.deepEqual(log.lines.filter((l) => l.includes("actions: read")), [`::warning::${NO_ACTIONS_PERMISSION_WARNING}`]);
    });

    test("other errors from the Actions API propagate", async () => {
      const client = scriptedClient([{ checkRuns: [job("build", "success", 1, 100)] }], {
        async listWorkflowRuns() {
          throw new HttpError("GET /actions/runs failed: 404 Not Found", 404);
        },
      });
      await assert.rejects(run(config(), { client, ...fakeClock(), logger: silent }), /404/);
    });
  });
});
