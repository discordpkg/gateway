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

## Go generate

General constants are automatically extracted from the discord api github repository.

## Testing

Test cases must respect the `short` flag. Writing an integration test must verify that `testing.Short()` is false.

## Error handling
Just like the go std packages, the error naming convention is:
- Err* for variables
- *Error for struct implementations
