# Details of Merge Gatekeeper

## Background

<!-- == export: background / begin == -->

Pull Request plays a significant part in day-to-day development, and making sure all the merges are controled is essential for building robust system. GitHub provides some control over CI, reviews, etc., but there are some limitations with handling special cases.

At UPSIDER, we have a few internal repositories set up with a monorepo structure, where there are many types of code in the single repository. This comes with its own pros and cons, but with GitHub Action and merge control, there is no way to specify "Ensure Go build and test pass _if and only if_ Go code is updated", or "Ensure E2E tests are run and successful _if and only if_ frontend code is updated". Because of this limitation, we would either need to run all the CI jobs for all the code for any Pull Requests, or do not set any limitation based on the CI status. <sup>(\*1)</sup>

Merge Gatekeeper was created to provide more control over merges.

---

NOTE <sup>(\*1)</sup>: There are some other hacks, such as using an empty job with the same name to override the status, but those solutions do not provide the flexible control we are after.

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
