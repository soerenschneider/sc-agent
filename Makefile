BUILD_DIR = builds
MODULE = github.com/soerenschneider/sc-agent
BINARY_NAME = sc-agent
CHECKSUM_FILE = $(BUILD_DIR)/checksum.sha256
SIGNATURE_KEYFILE = ~/.signify/github.sec
DOCKER_PREFIX = ghcr.io/soerenschneider
OUTPUT = $(BINARY_NAME)

# Reproducible builds: use the exact toolchain pinned in go.mod instead of whatever Go is installed, strip local paths
# and ignore GOFLAGS from the environment.
export GOTOOLCHAIN := $(shell awk '/^toolchain /{print $$2}' go.mod)
LDFLAGS = -w -X '$(MODULE)/internal.BuildVersion=$(VERSION)' -X '$(MODULE)/internal.CommitHash=$(COMMIT_HASH)'
GO_BUILD = CGO_ENABLED=0 GOFLAGS= go build -trimpath -ldflags="$(LDFLAGS)"

generate:
	go generate  ./...

tests:
	go test ./... -race -covermode=atomic -coverprofile=coverage.out
	go tool cover -html=coverage.out -o=coverage.html
	go tool cover -func=coverage.out -o=coverage.out

clean:
	test -z "$$(git status --porcelain)" || { echo 'Dirty work tree' ; false; }
	rm -rf ./$(BUILD_DIR)

build: version-info generate
	$(GO_BUILD) -o $(BINARY_NAME) ./cmd

# compile only runs the build without generating code or reading version info from git, it is used by the Dockerfile.
# VERSION and COMMIT_HASH must be passed.
compile:
	$(GO_BUILD) -o $(OUTPUT) ./cmd

release: clean version-info cross-build
	cd $(BUILD_DIR) && sha256sum $(BINARY_NAME)-* > $(notdir $(CHECKSUM_FILE))

signed-release: release
	pass keys/signify/github | signify -S -s $(SIGNATURE_KEYFILE) -m $(CHECKSUM_FILE)
	gh-upload-assets -o soerenschneider -r sc-agent -f ~/.gh-token builds

cross-build: version-info
	GOOS=linux GOARCH=amd64 GOAMD64=v1     $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64    ./cmd
	GOOS=linux GOARCH=arm GOARM=6          $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv6    ./cmd
	GOOS=linux GOARCH=arm GOARM=7          $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv7    ./cmd
	GOOS=linux GOARCH=arm64 GOARM64=v8.0   $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-aarch64  ./cmd

validate-openapi-spec:
	docker run --rm -it -v $(shell pwd):/tmp stoplight/spectral lint --ruleset "/tmp/.spectral.yaml" "/tmp/openapi.yaml"

docker-build: version-info
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT_HASH=$(COMMIT_HASH) -t "$(DOCKER_PREFIX)/sc-agent-server" .

version-info:
	$(eval VERSION := $(shell git describe --tags --abbrev=0 || echo "dev"))
	$(eval COMMIT_HASH := $(shell git rev-parse HEAD))

fmt:
	find . -iname "*.go" -exec go fmt {} \; 

pre-commit-init:
	pre-commit install
	pre-commit install --hook-type commit-msg

pre-commit-update:
	pre-commit autoupdate

docs:
	rm -rf go-diagrams
	go run doc/main.go
	cd go-diagrams && dot -Tpng diagram.dot > ../overview.png

tofu:
	tofu -chdir=contrib/terraform destroy -auto-approve; tofu -chdir=contrib/terraform apply -auto-approve
