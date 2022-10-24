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
		if len(names) != 0 {
			jobs := strings.Split(strings.ReplaceAll(names, "\n", ""), ",")

			for _, job := range jobs {
				s.ignoredJobs = append(s.ignoredJobs, strings.TrimSpace(job))
			}
		}
	}
}
