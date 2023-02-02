[![Go Reference](https://pkg.go.dev/badge/github.com/eighty4/maestro.svg)](https://pkg.go.dev/github.com/eighty4/maestro)
[![CI](https://img.shields.io/github/actions/workflow/status/eighty4/maestro/test.yml)](https://github.com/eighty4/maestro/actions/workflows/test.yml)

# Maestro

A developer utility that synchronizes and orchestrates local dev environments.

## Workspaces

Maestro will sync a workspace of git repositories with the `maestro git` command. That command will perform a `git pull`
on any repositories found in the directory its run from.
