import assert from "node:assert/strict";
import { describe, test } from "node:test";

import {
  State,
  collectJobStates,
  evaluate,
  formatDetail,
  ownerAndRepository,
  parseIgnored,
  parseSeconds,
  unknownActionsSuites,
} from "../src/core.js";

const { SUCCESS, ERROR, FAILURE, PENDING } = State;
const status = (context, state) => ({ context, state });
const run = (name, runStatus, conclusion, extra = {}) => ({ name, status: runStatus, conclusion, ...extra });
/** Job list -> [label, state] pairs, for compact assertions. */
const pairs = (jobs) => jobs.map((j) => [j.label, j.state]);

describe("parseIgnored (Go: WithIgnoredJobs)", () => {
  test("splits and trims a comma-separated list", () => {
    assert.deepEqual(parseIgnored("job-01,job-02"), ["job-01", "job-02"]);
    assert.deepEqual(parseIgnored(" job-01 , job-02 "), ["job-01", "job-02"]);
  });
  test("malformed input is handled", () => {
    assert.deepEqual(parseIgnored(","), []);
    assert.deepEqual(parseIgnored("job-01,,job-02,"), ["job-01", "job-02"]);
  });
  test("empty input", () => {
    assert.deepEqual(parseIgnored(""), []);
    assert.deepEqual(parseIgnored(undefined), []);
  });
});

describe("parseSeconds", () => {
  test("a bare number is seconds (v1 compatible)", () => {
    assert.equal(parseSeconds("timeout", "600"), 600);
    assert.equal(parseSeconds("timeout", " 0 "), 0);
  });
  test("accepts durations with h / m / s units", () => {
    assert.equal(parseSeconds("timeout", "90s"), 90);
    assert.equal(parseSeconds("timeout", "10m"), 600);
    assert.equal(parseSeconds("timeout", "1h"), 3600);
    assert.equal(parseSeconds("timeout", "1m30s"), 90);
    assert.equal(parseSeconds("timeout", "1h30m"), 5400);
    assert.equal(parseSeconds("timeout", "2h5m10s"), 7510);
  });
  test("rejects anything else", () => {
    for (const v of ["", "-1", "1.5", "abc", "10x", "30s10m", "1.5m", "m", "10 m"]) {
      assert.throws(() => parseSeconds("timeout", v), /timeout/, `should reject "${v}"`);
    }
  });
});

describe("ownerAndRepository (Go: ownerAndRepository)", () => {
  test("splits owner/repo", () => {
    assert.deepEqual(ownerAndRepository("upsidr/merge-gatekeeper"), { owner: "upsidr", repo: "merge-gatekeeper" });
  });
  test("missing repo", () => {
    assert.deepEqual(ownerAndRepository("upsidr"), { owner: "upsidr", repo: "" });
    assert.deepEqual(ownerAndRepository(""), { owner: "", repo: "" });
    assert.deepEqual(ownerAndRepository(undefined), { owner: "", repo: "" });
  });
  test("extra segments belong to repo", () => {
    assert.deepEqual(ownerAndRepository("a/b/c"), { owner: "a", repo: "b/c" });
  });
});

describe("collectJobStates (Go: listGhaStatuses)", () => {
  const checkRuns = [
    run("job-02", "failure"), // not completed -> pending
    run("job-03", "completed", "neutral"),
    run("job-04", "completed", "success"),
    run("job-05", "completed", "failure"),
    run("job-06", "completed", "skipped"),
  ];
  const expected = [
    ["job-01", SUCCESS],
    ["job-02", PENDING],
    ["job-03", SUCCESS],
    ["job-04", SUCCESS],
    ["job-05", ERROR],
  ];

  test("returns job states", () => {
    const got = collectJobStates([status("job-01", SUCCESS)], checkRuns);
    assert.deepEqual(pairs(got), expected);
  });

  test("succeeds to get job statuses even if the same job exists (first one wins)", () => {
    const got = collectJobStates(
      [status("job-01", SUCCESS), status("job-01", ERROR)],
      [run("job-02", "failure"), run("job-02", "completed", "neutral"), ...checkRuns.slice(1)],
    );
    assert.deepEqual(pairs(got), expected);
  });

  test("check run with a name already seen in statuses is skipped", () => {
    const got = collectJobStates([status("job", PENDING)], [run("job", "completed", "success")]);
    assert.deepEqual(pairs(got), [["job", PENDING]]);
  });

  test("an older run cannot reappear after a newer skipped run", () => {
    const got = collectJobStates([], [run("job", "completed", "skipped"), run("job", "completed", "failure")]);
    assert.deepEqual(pairs(got), []);
  });

  test("completed check run with null conclusion is pending, not a crash", () => {
    const got = collectJobStates([], [run("job", "completed", null), run("job-2", "completed", undefined)]);
    assert.deepEqual(pairs(got), [["job", PENDING], ["job-2", PENDING]]);
  });

  test("handles hundreds of jobs", () => {
    for (const n of [100, 162, 587]) {
      const names = Array.from({ length: n }, (_, i) => `job-${i}`);
      const got = collectJobStates(
        names.map((j) => status(j, SUCCESS)),
        names.map((j) => run(j, "completed", "neutral")),
      );
      assert.deepEqual(pairs(got), names.map((j) => [j, SUCCESS]));
    }
  });

  test("returns error when the combined status response is invalid", () => {
    assert.throws(() => collectJobStates([{}], []), /combined status response is invalid/);
    assert.throws(() => collectJobStates([{ context: "job" }], []), /combined status response is invalid/);
  });

  test("returns error when the check run response is invalid", () => {
    assert.throws(() => collectJobStates([], [{}]), /checkRun response is invalid/);
    assert.throws(() => collectJobStates([], [{ name: "job" }]), /checkRun response is invalid/);
  });
});

describe("jobs with the same name in different workflows (#60)", () => {
  const ACTIONS = { slug: "github-actions", id: 15368 };
  /** An Actions check run in check suite `suite`. */
  const job = (name, runStatus, conclusion, id, suite) =>
    run(name, runStatus, conclusion, { id, check_suite: { id: suite }, app: ACTIONS });
  const workflow = (id, name, path = `.github/workflows/${name.toLowerCase()}.yaml`) => ({ id, name, path });
  // Check suites 100 and 101 are two runs of "CI"; 200 is a run of "Deploy".
  const workflows = () =>
    new Map([
      [100, workflow(1, "CI")],
      [101, workflow(1, "CI")],
      [200, workflow(2, "Deploy")],
    ]);
  const verdict = (jobs) => evaluate(jobs, { self: "merge-gatekeeper", ignored: [] });

  test("are kept apart by workflow, and labelled with the workflow name", () => {
    const got = collectJobStates([], [
      job("build", "completed", "success", 1, 100),
      job("build", "completed", "failure", 2, 200),
      job("lint", "completed", "success", 3, 100),
    ], workflows());
    assert.deepEqual(pairs(got), [
      ["build (CI)", SUCCESS],
      ["build (Deploy)", ERROR],
      ["lint", SUCCESS],
    ]);
  });

  test("a failure is not hidden by a success with the same name, whatever the API order", () => {
    const success = job("Plan / Apply", "completed", "success", 10, 100);
    const failure = job("Plan / Apply", "completed", "failure", 11, 200);
    for (const order of [[success, failure], [failure, success]]) {
      const v = verdict(collectJobStates([], order, workflows()));
      assert.equal(v.failed, true);
      assert.deepEqual(v.errJobs, ["Plan / Apply (Deploy)"]);
    }
  });

  test("a running job is not hidden by a finished one with the same name", () => {
    const v = verdict(
      collectJobStates([], [job("build", "completed", "success", 1, 100), job("build", "in_progress", null, 2, 200)], workflows()),
    );
    assert.equal(v.succeeded, false);
    assert.equal(v.failed, false);
  });

  test("a new run of the same workflow replaces the older run, whatever the API order", () => {
    // e.g. concurrency: cancel-in-progress, or a new event on the same commit.
    for (const older of ["cancelled", "failure"]) {
      const old = job("test", "completed", older, 1, 100);
      const latest = job("test", "completed", "success", 2, 101);
      for (const order of [[old, latest], [latest, old]]) {
        assert.deepEqual(pairs(collectJobStates([], order, workflows())), [["test", SUCCESS]], older);
      }
    }
  });

  test("within a check suite, only the latest re-run counts", () => {
    for (const order of [
      [job("build", "completed", "failure", 1, 100), job("build", "completed", "success", 5, 100)],
      [job("build", "completed", "success", 5, 100), job("build", "completed", "failure", 1, 100)],
    ]) {
      assert.deepEqual(pairs(collectJobStates([], order, workflows())), [["build", SUCCESS]]);
    }
  });

  test("a skipped latest run drops the job even if an older run failed", () => {
    const got = collectJobStates([], [
      job("build", "completed", "failure", 1, 100),
      job("build", "completed", "skipped", 2, 101),
    ], workflows());
    assert.deepEqual(pairs(got), []);
  });

  test("workflows that also share a name are labelled with their file", () => {
    const map = new Map([
      [100, workflow(1, "CI", ".github/workflows/ci.yaml")],
      [200, workflow(2, "CI", ".github/workflows/ci-nightly.yaml")],
    ]);
    const got = collectJobStates([], [job("build", "completed", "success", 1, 100), job("build", "completed", "failure", 2, 200)], map);
    assert.deepEqual(pairs(got), [
      ["build (.github/workflows/ci.yaml)", SUCCESS],
      ["build (.github/workflows/ci-nightly.yaml)", ERROR],
    ]);
  });

  test("an Actions check suite not resolved yet is kept apart by its id", () => {
    const got = collectJobStates([], [job("build", "completed", "success", 1, 100), job("build", "completed", "failure", 2, 300)], workflows());
    assert.deepEqual(pairs(got), [
      ["build (CI)", SUCCESS],
      ["build (check suite 300)", ERROR],
    ]);
  });

  test("check runs from other apps are kept apart by app, latest run wins within an app", () => {
    const other = (name, conclusion, id, suite) =>
      run(name, "completed", conclusion, { id, check_suite: { id: suite }, app: { id: 7, slug: "other-ci" } });
    const got = collectJobStates([], [
      job("build", "completed", "success", 1, 100),
      other("build", "failure", 2, 500),
      other("build", "success", 3, 501),
    ], workflows());
    assert.deepEqual(pairs(got), [
      ["build (CI)", SUCCESS],
      ["build (other-ci)", SUCCESS],
    ]);
  });

  test("without workflow info (no actions: read), jobs are identified by name, latest run wins", () => {
    const got = collectJobStates([], [
      job("Dependabot", "completed", "cancelled", 1, 100),
      job("Dependabot", "completed", "success", 4, 103),
      job("Dependabot", "completed", "success", 2, 101),
    ]);
    assert.deepEqual(pairs(got), [["Dependabot", SUCCESS]]);
  });

  test("ignored and self apply to every job with that name", () => {
    const v = evaluate(
      collectJobStates([], [
        job("flaky", "completed", "failure", 1, 100),
        job("flaky", "completed", "failure", 2, 200),
        job("merge-gatekeeper", "in_progress", null, 3, 100),
        job("merge-gatekeeper", "completed", "failure", 4, 200),
        job("build", "completed", "success", 5, 100),
      ], workflows()),
      { self: "merge-gatekeeper", ignored: ["flaky"] },
    );
    assert.equal(v.failed, false);
    assert.equal(v.succeeded, true);
    assert.deepEqual(v.totalJobs, ["build"]);
  });

  test("unknownActionsSuites lists Actions check suites missing from the map", () => {
    const runs = [
      job("a", "completed", "success", 1, 100),
      job("b", "completed", "success", 2, 300),
      job("c", "completed", "success", 3, 300),
      run("d", "completed", "success", { id: 4, check_suite: { id: 400 }, app: { id: 7, slug: "other-ci" } }),
    ];
    assert.deepEqual(unknownActionsSuites(runs, workflows()), [300]);
  });
});

describe("evaluate (Go: statusValidator.Validate)", () => {
  const jobs = (...entries) => entries.map(([name, state]) => ({ name, label: name, state }));
  const verdict = (v) => ({ ignoredJobs: [], ...v });

  test("succeeded when there is no job", () => {
    assert.deepEqual(
      evaluate(jobs(), { self: "", ignored: [] }),
      verdict({ totalJobs: [], completeJobs: [], errJobs: [], failed: false, succeeded: true }),
    );
  });

  test("succeeded when the only job is itself", () => {
    assert.deepEqual(
      evaluate(jobs(["self-job", PENDING]), { self: "self-job", ignored: [] }),
      verdict({ totalJobs: [], completeJobs: [], errJobs: [], failed: false, succeeded: true }),
    );
  });

  test("not succeeded when there is one pending job", () => {
    assert.deepEqual(
      evaluate(jobs(["job", PENDING]), { self: "", ignored: [] }),
      verdict({ totalJobs: ["job"], completeJobs: [], errJobs: [], failed: false, succeeded: false }),
    );
  });

  for (const state of [ERROR, FAILURE]) {
    test(`failed when there is a job in ${state} state`, () => {
      const v = evaluate(jobs(["job-01", SUCCESS], ["job-02", state], ["self-job", PENDING]), {
        self: "self-job",
        ignored: [],
      });
      assert.deepEqual(
        v,
        verdict({ totalJobs: ["job-01", "job-02"], completeJobs: ["job-01"], errJobs: ["job-02"], failed: true, succeeded: false }),
      );
    });
  }

  test("not succeeded when successful job count is less than total", () => {
    assert.deepEqual(
      evaluate(jobs(["job-01", SUCCESS], ["job-02", PENDING], ["self-job", PENDING]), { self: "self-job", ignored: [] }),
      verdict({ totalJobs: ["job-01", "job-02"], completeJobs: ["job-01"], errJobs: [], failed: false, succeeded: false }),
    );
  });

  test("succeeded when all jobs succeed", () => {
    assert.deepEqual(
      evaluate(jobs(["job-01", SUCCESS], ["job-02", SUCCESS], ["self-job", PENDING]), { self: "self-job", ignored: [] }),
      verdict({
        totalJobs: ["job-01", "job-02"],
        completeJobs: ["job-01", "job-02"],
        errJobs: [],
        failed: false,
        succeeded: true,
      }),
    );
  });

  for (const state of [ERROR, FAILURE]) {
    test(`succeeded when only an ignored job is in ${state} state`, () => {
      assert.deepEqual(
        evaluate(jobs(["job-01", SUCCESS], ["job-02", state], ["self-job", PENDING]), {
          self: "self-job",
          ignored: ["job-02", "job-03"],
        }),
        {
          totalJobs: ["job-01"],
          completeJobs: ["job-01"],
          errJobs: [],
          ignoredJobs: ["job-02", "job-03"],
          failed: false,
          succeeded: true,
        },
      );
    });
  }
});

describe("formatDetail (Go: status.Detail)", () => {
  test("totalJobs, completeJobs and errJobs are not empty", () => {
    const got = formatDetail({ totalJobs: ["job-1", "job-2", "job-3"], completeJobs: ["job-2"], errJobs: ["job-3"] });
    assert.equal(
      got,
      `1 out of 3

Total job count:       3
Completed job count:   1
Incompleted job count: 1
Failed job count:      1
Ignored job count:     0

::group::Failed jobs
- job-3
::endgroup::

::group::Completed jobs
- job-2
::endgroup::

::group::Incomplete jobs
- job-1
::endgroup::

::group::Ignored jobs
[]
::endgroup::

::group::All jobs
- job-1
- job-2
- job-3
::endgroup::
`,
    );
  });

  test("with ignored jobs input", () => {
    const got = formatDetail({
      totalJobs: ["job-1", "job-2", "job-3", "job-4"],
      completeJobs: ["job-2", "job-4"],
      errJobs: ["job-3"],
      ignoredJobs: ["job-4"],
    });
    assert.equal(
      got,
      `2 out of 4

Total job count:       4
Completed job count:   2
Incompleted job count: 1
Failed job count:      1
Ignored job count:     1

::group::Failed jobs
- job-3
::endgroup::

::group::Completed jobs
- job-2
- job-4
::endgroup::

::group::Incomplete jobs
- job-1
::endgroup::

::group::Ignored jobs
- job-4
::endgroup::

::group::All jobs
- job-1
- job-2
- job-3
- job-4
::endgroup::
`,
    );
  });

  test("totalJobs and completeJobs are empty", () => {
    const got = formatDetail({ totalJobs: [], completeJobs: [] });
    assert.equal(
      got,
      `0 out of 0

Total job count:       0
Completed job count:   0
Incompleted job count: 0
Failed job count:      0
Ignored job count:     0

::group::Failed jobs
[]
::endgroup::

::group::Completed jobs
[]
::endgroup::

::group::Incomplete jobs
[]
::endgroup::

::group::Ignored jobs
[]
::endgroup::

::group::All jobs
[]
::endgroup::
`,
    );
  });
});
