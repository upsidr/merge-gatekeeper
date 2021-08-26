# Details of Merge Gatekeeper

## Background

<!-- == export: background / begin == -->

Pull Request plays a significant part in day-to-day development, and it is essential to ensure all merges are well controlled and managed to build robust system. GitHub provides some control over CI, reviews, etc., but there are some limitations with handling special cases. Merge Gatekeeper provides that extra control that's not handled by GitHub.

At UPSIDER, we have a few internal repositories set up with a monorepo structure, where there are many types of code in a single repository. This comes with its own pros and cons, but one difficulty is how we end up with various CI jobs, which are only run for some changes that touch relevant files. With GitHub's branch protection, there is no way to specify "Ensure Go build and test pass _if and only if_ Go code is updated", or "Ensure E2E tests are run and successful _if and only if_ frontend code is updated". Because of this limitation, we would either need to run all the CI jobs for any Pull Requests, or do not set any limitation based on the CI status. <sup><sub><sup>(\*1)</sup></sub></sup>

**Merge Gatekeeper** was created to provide more control for merges. By placing Merge Gatekeeper for all PRs, it will check all other CI jobs that get kicked off for any PR, and ensures all the jobs are completed successfully. If there is any job that has failed, Merge Gatekeeper will fail as well. This allows you to control merge protection by looking at Merge Gatekeeper status, which means you can effectively ensure any CI that fails will block the PR merge. All you need is the Merge Gatekeeper as one of the PR based GitHub Action, and set the branch protection rule as shown below.

![Branch protection example](/assets/images/branch-protection-example.png)

<sup><sub>NOTE:  
<sup>(\*1)</sup> There are some other hacks, such as using an empty job with the same name to override the status, but those solutions do not provide the flexible control we are after.</sub></sup>

<!-- == export: background / end == -->

## Support

<!-- == export: support / begin == -->

Merge Gatekeeper provides additional control that may be useful for large and complex repositories.

### Ensure all CI jobs are successful

By default, when Merge Gatekeeper is used for PR, it periodically checks the PR by checking all the other CI jobs. This means if you have complex CI scenarios where some CIs run only for specific changes, you can still ensure all the CI jobs have run successfully in order to merge the PR.

### Other validations

We are currently considering additional validation controls such as:

- extra approval by comment
- label validation

<!-- == export: support / end == -->

## How does Merge Gatekeeper work?

<!-- == implementation-details: support / begin == -->

Merge Gatekeeper periodically validates the PR status by hitting GitHub API. The GitHub token is thus required for Merge Gatekeeper to operato, and it's often enough to have `${{ secrets.GITHUB_TOKEN }}` to be provided. The API call to list PR jobs will reveal how many jobs need to run for the given PR, check each job status, and finally return the validation status - success based on completing all the jobs, or timeout error. It is important for Merge Gatekeeper to know the Job name of itself, so that when API call returns Merge Gatekeeper as a part of the PR jobs, it would ignore its status (otherwise it will never succeed).

<!-- TODO: Add more about other validation types when we add support -->

<!-- == implementation-details: support / end == -->
