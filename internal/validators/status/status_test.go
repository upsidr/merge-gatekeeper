package status

import (
	"testing"
)

func Test_status_Detail(t *testing.T) {
	tests := map[string]struct {
		s    *status
		want string
	}{
		"return detail when totalJobs and completeJobs and errJobs is not empty": {
			s: &status{
				totalJobs: []string{
					"job-1",
					"job-2",
					"job-3",
				},
				completeJobs: []string{
					"job-2",
				},
				errJobs: []string{
					"job-3",
				},
			},
			want: `1 out of 3

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
		},
		"return detail with ignored jobs input": {
			s: &status{
				totalJobs: []string{
					"job-1",
					"job-2",
					"job-3",
					"job-4",
				},
				completeJobs: []string{
					"job-2",
					"job-4",
				},
				errJobs: []string{
					"job-3",
				},
				ignoredJobs: []string{
					"job-4",
				},
			},
			want: `2 out of 4

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
		},
		"return detail when totalJobs and completeJobs is empty": {
			s: &status{
				totalJobs:    []string{},
				completeJobs: []string{},
			},
			want: `0 out of 0

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
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.s.Detail()
			if got != tt.want {
				t.Errorf("status.Detail() didn't match\n  got:\n%s\n\n  want:\n%s", got, tt.want)
			}
		})
	}
}
