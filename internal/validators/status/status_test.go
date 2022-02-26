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
				t.Errorf("status.Detail() str = %s, want: %s", got, tt.want)
			}
		})
	}
}
