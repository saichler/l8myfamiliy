echo "Building Linux"
GOOS=linux GOARCH=amd64 go build -o agent-linux
echo "Building Windows"
GOOS=windows GOARCH=amd64 go build -o agent-windows.exe
echo "Building Mac Intel"
GOOS=darwin GOARCH=amd64 go build -o agent-mac-intel
echo "building Mac Apple"
GOOS=darwin GOARCH=arm64 go build -o agent-mac-apple
