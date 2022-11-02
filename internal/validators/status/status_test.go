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

  Total job count:     3
    jobs: ["job-1" "job-2" "job-3"]
  Completed job count: 1
    jobs: ["job-2"]
  Failed job count:    1
    jobs: ["job-3"]
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

  Total job count:     4
    jobs: ["job-1" "job-2" "job-3" "job-4"]
  Completed job count: 2
    jobs: ["job-2" "job-4"]
  Failed job count:    1
    jobs: ["job-3"]


  --
  Ignored jobs: ["job-4"]`,
		},
		"return detail when totalJobs and completeJobs is empty": {
			s: &status{
				totalJobs:    []string{},
				completeJobs: []string{},
			},
			want: `0 out of 0

  Total job count:     0
    jobs: []
  Completed job count: 0
    jobs: []
  Failed job count:    0
    jobs: []
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
