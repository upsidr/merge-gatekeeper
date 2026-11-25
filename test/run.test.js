import assert from "node:assert/strict";
import { describe, test } from "node:test";

import { readConfig, run } from "../src/run.js";

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

  test("errors on missing ref, token, or repository", () => {
    assert.throws(() => readConfig({ ...baseEnv, INPUT_REF: "" }), /reference of repository is empty/);
    assert.throws(() => readConfig({ ...baseEnv, INPUT_TOKEN: "" }), /token is empty/);
    assert.throws(() => readConfig({ ...baseEnv, GITHUB_REPOSITORY: "" }), /owner or repository is empty/);
  });

  test("errors on non-numeric seconds", () => {
    assert.throws(() => readConfig({ ...baseEnv, INPUT_TIMEOUT: "10m" }), /timeout/);
  });
});

/** Fake client that returns a scripted sequence of snapshots, one per poll. */
function scriptedClient(snapshots) {
  let i = 0;
  let current;
  return {
    polls: () => i,
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
});
