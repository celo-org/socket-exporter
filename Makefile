.PHONY: build
build:
	@echo "\n🔧  Building Go binaries..."
	GOOS=darwin GOARCH=amd64 go build -o bin/image-annotator-webhook-darwin-amd64 .
	GOOS=linux GOARCH=amd64 go build -o bin/image-annotator-webhook-linux-amd64 .

