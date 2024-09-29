APP_NAME = ai-cli
VERSION = 1.0.0
PLATFORMS = linux/amd64 windows/amd64 darwin/amd64
DOCKER_IMAGE = your-docker-username/$(APP_NAME)

build:
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		OS=$${platform%/*}; ARCH=$${platform##*/}; \
		OUTPUT_NAME=$(APP_NAME)-$$OS-$$ARCH; \
		if [ $$OS = "windows" ]; then OUTPUT_NAME=$$OUTPUT_NAME.exe; fi; \
		GOOS=$$OS GOARCH=$$ARCH go build -o dist/$$OUTPUT_NAME ./cmd/ai-cli; \
		echo "Built $$OUTPUT_NAME"; \
	done

docker-build:
	@docker build -t $(DOCKER_IMAGE):$(VERSION) .

docker-run:
	@docker run --rm -it $(DOCKER_IMAGE):$(VERSION)

clean:
	@rm -rf dist
	@echo "Cleaned build files."

test:
	@go test ./...

fmt:
	@go fmt ./...

release: clean build docker-build
	@echo "Release $(VERSION) prepared for $(APP_NAME)"
