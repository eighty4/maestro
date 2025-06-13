# Changelog

## Unreleased

- Sync API supports offline flag to collect repository statuses without syncing
- Sync API uses DNS lookup to determine network connectivity before syncing

## 0.2.1 - 2025-03-13

- Sync API collects local change and stash counts in repository

## 0.2.0 - 2025-03-09

- Sync API parallelizes git pulls across multiple git repositories
- Sync API provides RemoteHost for accessing featurs of the remote's Git services such as GitHub's ref compare page

[Unreleased]: https://github.com/eighty4/maestro/compare/maestro_git-v0.2.1...HEAD
[0.2.1]: https://github.com/eighty4/maestro/compare/maestro_git-v0.2.0...maestro_git-v0.2.1
[0.2.0]: https://github.com/eighty4/maestro/releases/tag/maestro_git-v0.2.0
