# maestro

`cd` to parent directory of your repositories and keep them up to date with
their remote and see what local state you have.

All with one command.

## `maestro git`

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

Use a stylish interactive mode with access to diffs for pulled commits.

## `maestro git --offline`

Opt to display local repository state without syncing.

