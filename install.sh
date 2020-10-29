rm -rf build
go build -o build/maestro config.go context.go frontend.go logs.go maestro.go process.go service.go
pushd frontend
yarn build
popd
cp -r frontend/dist build/frontend
