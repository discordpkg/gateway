# Contributing

## Pull requests

> Pull requests should be kept small, one intent per PR. Open multiple, instead of a single large Pull request.

Pull requests are squashed to keep the git history clean. To make the commit message informative we use
[commit lint](https://github.com/conventional-changelog/commitlint#what-is-commitlint) to verify that the PR title starts with one of the following prefixes:

- ci
- chore
- docs
- feat
- fix
- perf
- refactor
- revert
- style
- test

Auto-commit is enforced in this repository. The idea is that only what is approved should be merged into master, and extra
commits can not be added later and silently merged in.

