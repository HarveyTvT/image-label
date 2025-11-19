build:
	rm -rf build/*
	mkdir -p build
	GOOS=linux GOARCH=amd64 go build -o build/image-label-linux-amd64
	GOOS=windows GOARCH=amd64 go build -o build/image-label-windows-amd64.exe
	GOOS=darwin GOARCH=amd64 go build -o build/image-label-mac-amd64
	GOOS=darwin GOARCH=arm64 go build -o build/image-label-mac-arm64
