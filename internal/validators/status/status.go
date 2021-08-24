package status

import "fmt"

type status struct {
	totalJobs    []string
	completeJobs []string
	succeeded    bool
}

func (s *status) Detail() string {
	return fmt.Sprintf(
		`%d out of %d

  total job count: %d
    jobs: %v
  completed job count: %d
    jobs: %v`,
		len(s.completeJobs), len(s.totalJobs),
		len(s.totalJobs), s.totalJobs,
		len(s.completeJobs), s.completeJobs,
	)
}

func (s *status) IsSuccess() bool {
	return s.succeeded
}
