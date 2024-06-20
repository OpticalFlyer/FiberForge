GOOS=windows GOARCH=amd64 go build -o bin/Hyperion.exe -ldflags -H=windowsgui
GOOS=darwin GOARCH=arm64 go build -o bin/Hyperion.app/Contents/MacOS/Hyperion
