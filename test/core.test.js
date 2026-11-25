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
} from "../src/core.js";

const { SUCCESS, ERROR, FAILURE, PENDING } = State;
const status = (context, state) => ({ context, state });
const run = (name, status, conclusion) => ({ name, status, conclusion });

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
  test("accepts non-negative integers", () => {
    assert.equal(parseSeconds("timeout", "600"), 600);
    assert.equal(parseSeconds("timeout", " 0 "), 0);
  });
  test("rejects anything else", () => {
    for (const v of ["", "-1", "1.5", "10s", "abc"]) {
      assert.throws(() => parseSeconds("timeout", v), /timeout/);
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
    assert.deepEqual([...got], expected);
  });

  test("succeeds to get job statuses even if the same job exists (first one wins)", () => {
    const got = collectJobStates(
      [status("job-01", SUCCESS), status("job-01", ERROR)],
      [run("job-02", "failure"), run("job-02", "completed", "neutral"), ...checkRuns.slice(1)],
    );
    assert.deepEqual([...got], expected);
  });

  test("check run with a name already seen in statuses is skipped", () => {
    const got = collectJobStates([status("job", PENDING)], [run("job", "completed", "success")]);
    assert.deepEqual([...got], [["job", PENDING]]);
  });

  test("an older run cannot reappear after a newer skipped run", () => {
    const got = collectJobStates([], [run("job", "completed", "skipped"), run("job", "completed", "failure")]);
    assert.deepEqual([...got], []);
  });

  test("completed check run with null conclusion is pending, not a crash", () => {
    const got = collectJobStates([], [run("job", "completed", null), run("job-2", "completed", undefined)]);
    assert.deepEqual([...got], [["job", PENDING], ["job-2", PENDING]]);
  });

  test("handles hundreds of jobs", () => {
    for (const n of [100, 162, 587]) {
      const names = Array.from({ length: n }, (_, i) => `job-${i}`);
      const got = collectJobStates(
        names.map((j) => status(j, SUCCESS)),
        names.map((j) => run(j, "completed", "neutral")),
      );
      assert.deepEqual([...got], names.map((j) => [j, SUCCESS]));
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

describe("evaluate (Go: statusValidator.Validate)", () => {
  const jobs = (...pairs) => new Map(pairs);
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
