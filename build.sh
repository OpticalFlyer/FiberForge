GOOS=windows GOARCH=amd64 go build -o bin/fiberforge.exe -ldflags -H=windowsgui
GOOS=darwin GOARCH=arm64 go build -o bin/fiberforge.app
