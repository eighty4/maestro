# maestro

## `maestro git`

`cd` to the parent directory of a bunch of repositories and keep all your
repositories up to date with their remote and quickly see what your local
looks like.

```
Syncing 11 repos...

  model-t              ✔ 1 stash
  c2                   ✔
  picking.pl           ✔
  binny.sh             ✔ 1 local change
  plunder              ✔ 1 stash
  cquill               ✔
  changelog            ✔ 2 local changes
  sidelines.dev        ✔ 9 local changes
  l3                   ✔ 47 local changes
  maestro              ✔ pulled 1 commits, 4 local changes
  libtab               ✔ 2 local changes

All repositories synced!
```

## `maestro git --interactive`

Use an interactive mode for viewing diffs of pulled commits.

## `maestro git --offline`

Opt to only print local repository state and do not sync with `--offline`.

