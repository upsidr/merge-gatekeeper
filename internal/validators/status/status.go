package status

import "fmt"

type status struct {
	totalJobs    []string
	completeJobs []string
	errJobs      []string
	succeeded    bool
}

func (s *status) Detail() string {
	return fmt.Sprintf(
		`%d out of %d

  Total job count:     %d
    jobs: %+q
  Completed job count: %d
    jobs: %+q
  Failed job count:    %d
    jobs: %+q
`,
		len(s.completeJobs), len(s.totalJobs),
		len(s.totalJobs), s.totalJobs,
		len(s.completeJobs), s.completeJobs,
		len(s.errJobs), s.errJobs,
	)
}

func (s *status) IsSuccess() bool {
	// TDOO: Add test case
	return s.succeeded
}
