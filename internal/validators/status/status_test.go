package status

import (
	"testing"
)

func Test_status_Detail(t *testing.T) {
	tests := map[string]struct {
		s    *status
		want string
	}{
		"return detail when totalJobs and completeJobs is not empty": {
			s: &status{
				totalJobs: []string{
					"job-1",
					"job-2",
				},
				completeJobs: []string{
					"job-2",
				},
			},
			want: `1 out of 2

  total job count: 2
    jobs: [job-1 job-2]
  completed job count: 1
    jobs: [job-2]`,
		},
		"return detail when totalJobs and completeJobs is empty": {
			s: &status{
				totalJobs:    []string{},
				completeJobs: []string{},
			},
			want: `0 out of 0

  total job count: 0
    jobs: []
  completed job count: 0
    jobs: []`,
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
