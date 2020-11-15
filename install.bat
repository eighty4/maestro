
Remove-Item -Recurse -Force build
Remove-Item -Recurse -Force frontend\dist
New-Item -Name build -ItemType directory
go.exe build -o build\maestro.exe config.go context.go frontend.go healthcheck.go logging.go maestro.go process.go service.go
cd frontend
yarn
yarn build
cd ..
Copy-Item -Path .\frontend\dist\ -Filter "*" -Recurse -Destination .\build\frontend -Container
