package status

import "strings"

type Option func(s *statusValidator)

func WithSelfJob(name string) Option {
	return func(s *statusValidator) {
		if len(name) != 0 {
			s.selfJobName = name
		}
	}
}

func WithGitHubOwnerAndRepo(owner, repo string) Option {
	return func(s *statusValidator) {
		if len(owner) != 0 {
			s.owner = owner
		}
		if len(repo) != 0 {
			s.repo = repo
		}
	}
}

func WithGitHubRef(ref string) Option {
	return func(s *statusValidator) {
		if len(ref) != 0 {
			s.ref = ref
		}
	}
}

func WithIgnoredJobs(names string) Option {
	return func(s *statusValidator) {
		// TODO: Add more input validation, such as "," should not be a valid input.
		if len(names) == 0 {
			return // TODO: Return some clearer error
		}

		jobs := []string{}
		ss := strings.Split(names, ",")
		for _, s := range ss {
			jobName := strings.TrimSpace(s)
			if len(jobName) == 0 {
				continue // TODO: Provide more clue to users
			}
			jobs = append(jobs, jobName)
		}
		s.ignoredJobs = jobs
	}
}
