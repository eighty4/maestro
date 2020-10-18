## Maestro
this is a developer utility that configures and manages services and platform components for local development. built with Go.

#### build
```
git clone https://github.com/eighty4/maestro.git
cd maestro
go build -o build/maestro maestro.go config.go service.go
```
add the location from `cd build && pwd` to your $PATH

#### use
in your gradle project dir, create a file `.maestro` with:
```yaml
---
services:
  my-service-name:
    gradle:
      module: my-service
      task: run
```
replace `my-service` with a sub-module of your project and `run` with the task you wish to run.

then run `maestro` to start your local dev env.
