# Maestro
a developer utility to configure and manage services and platform components for local development. built with Go.

## build
```
git clone https://github.com/eighty4/maestro.git
cd maestro
./install.sh
```
add the location from `cd dist && pwd` to your $PATH

## use `maestro`
the primary `maestro` program will manage a set of processes defined in a `.maestro` file to simplify local dev env setup.

to run any executable command:
```yaml
---
services:
  my-service-name:
    exec:
      cmd: sleep 9000
```

if you're using gradle:
```yaml
---
services:
  my-service-name:
    gradle:
      module: my-service
      task: run
```
replace `my-service` with the submodule of your project and `run` with the task you wish to run.

if you're using npm scripts:
```yaml
---
services:
  my-service-name:
    npm:
      script: start
```

then run `maestro` to start your local dev env.

## use `maestro git`

this command will do a `git pull` in each git repo subdirectory of your cwd.
