package status

import "fmt"

type status struct {
	totalJobs    []string
	completeJobs []string
	errJobs      []string
	ignoredJobs  []string
	succeeded    bool
}

func (s *status) Detail() string {
	result := fmt.Sprintf(
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

	if len(s.ignoredJobs) > 0 {
		result = fmt.Sprintf(
			`%s

  --
  Ignored jobs: %+q`, result, s.ignoredJobs)
	}

	return result
}

func (s *status) IsSuccess() bool {
	// TDOO: Add test case
	return s.succeeded
}
