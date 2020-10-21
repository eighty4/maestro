## Maestro
this is a developer utility to configure and manage services and platform components for local development. built with Go.

#### build
```
git clone https://github.com/eighty4/maestro.git
cd maestro
go build -o build/maestro maestro.go config.go service.go
```
add the location from `cd build && pwd` to your $PATH

#### use
to run any executable command, use the following `.maestro` file:
```yaml
---
services:
  my-service-name:
    exec: sleep 9000
```

if you're using gradle, in your gradle project dir, create a file `.maestro` with:
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
