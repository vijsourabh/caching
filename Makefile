GOLANGCI_LINT_VERSION ?=

prep: vendor tools fmt lint vet cover

builddir: clean
	mkdir -p -m 0777 build

vet:
	go vet ./...

lint:
	golangci-lint run --timeout 5m

clean:
	rm -rf build/*

fmt:
	go fmt ./...

test:
	go test ./...

cover: builddir
	go test -timeout 40m -v -covermode=count -coverprofile=build/coverage.out -json ./...

	# code coverage using gocover
	go tool cover -html=build/coverage.out -o build/coverage.html
	go tool cover -func build/coverage.out

update:
	go get -u ./...
	go mod tidy
	go mod vendor

vendor:
	go mod tidy
	go mod vendor

tools:
	go install golang.org/x/tools/cmd/cover@latest
	sh -c "$$(wget -O - -q https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh || echo exit 2)" -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

.PHONY: vendor