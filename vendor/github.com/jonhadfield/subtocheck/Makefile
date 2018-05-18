SOURCE_FILES?=$$(go list ./... | grep -v /vendor/ | grep -v /mocks/)
TEST_PATTERN?=.
TEST_OPTIONS?=-race -v

setup:
	go get -u github.com/smartystreets/goconvey/convey
	go get -u github.com/Knetic/govaluate
	go get -u github.com/alecthomas/gometalinter
	go get -u github.com/pierrre/gotestcover
	go get -u golang.org/x/tools/cmd/cover
	gometalinter --install --update

test:
	gotestcover $(TEST_OPTIONS) -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=30s

cover: test
	go tool cover -html=coverage.txt

fmt:
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done

lint:
	gometalinter -e testing.go -e validation_test.go --vendor --disable-all \
		--enable=deadcode \
		--enable=errcheck \
		--enable=gofmt \
		--enable=goimports \
		--enable=golint \
		--enable=gosimple \
		--enable=ineffassign \
		--enable=misspell \
		--enable=unconvert \
		--enable=varcheck \
		--enable=staticcheck \
		--enable=unparam\
		--enable=varcheck \
		--enable=dupl \
		--enable=structcheck \
		--enable=vetshadow \
		--deadline=10m \
		./...

ci: lint test

BUILD_TAG := $(shell git describe --tags 2>/dev/null)
BUILD_SHA := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u '+%Y/%m/%d:%H:%M:%S')

build: fmt
	GOOS=darwin CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/subtocheck_darwin_amd64" cmd/subtocheck/main.go

build-all: fmt
	GOOS=darwin  CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/subtocheck_darwin_amd64"  cmd/subtocheck/main.go
	GOOS=linux   CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/subtocheck_linux_amd64"   cmd/subtocheck/main.go
	GOOS=linux   CGO_ENABLED=0 GOARCH=arm   go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/subtocheck_linux_arm"     cmd/subtocheck/main.go
	GOOS=linux   CGO_ENABLED=0 GOARCH=arm64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/subtocheck_linux_arm64"   cmd/subtocheck/main.go
	GOOS=netbsd  CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/subtocheck_netbsd_amd64"  cmd/subtocheck/main.go
	GOOS=openbsd CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/subtocheck_openbsd_amd64" cmd/subtocheck/main.go
	GOOS=freebsd CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/subtocheck_freebsd_amd64" cmd/subtocheck/main.go

install:
	go install ./cmd/...

bintray:
	curl -X PUT -0 -T .local_dist/subtocheck_darwin_amd64 -ujonhadfield:$(BINTRAY_APIKEY) "https://api.bintray.com/content/jonhadfield/subtocheck/$(BUILD_TAG)/subtocheck_darwin_amd64;bt_package=subtocheck;bt_version=$(BUILD_TAG);publish=1"
	curl -X PUT -0 -T .local_dist/subtocheck_linux_amd64 -ujonhadfield:$(BINTRAY_APIKEY) "https://api.bintray.com/content/jonhadfield/subtocheck/$(BUILD_TAG)/subtocheck_linux_amd64;bt_package=subtocheck;bt_version=$(BUILD_TAG);publish=1"
	curl -X PUT -0 -T .local_dist/subtocheck_linux_arm -ujonhadfield:$(BINTRAY_APIKEY) "https://api.bintray.com/content/jonhadfield/subtocheck/$(BUILD_TAG)/subtocheck_linux_arm;bt_package=subtocheck;bt_version=$(BUILD_TAG);publish=1"
	curl -X PUT -0 -T .local_dist/subtocheck_linux_arm64 -ujonhadfield:$(BINTRAY_APIKEY) "https://api.bintray.com/content/jonhadfield/subtocheck/$(BUILD_TAG)/subtocheck_linux_arm64;bt_package=subtocheck;bt_version=$(BUILD_TAG);publish=1"
	curl -X PUT -0 -T .local_dist/subtocheck_netbsd_amd64 -ujonhadfield:$(BINTRAY_APIKEY) "https://api.bintray.com/content/jonhadfield/subtocheck/$(BUILD_TAG)/subtocheck_netbsd_amd64;bt_package=subtocheck;bt_version=$(BUILD_TAG);publish=1"
	curl -X PUT -0 -T .local_dist/subtocheck_openbsd_amd64 -ujonhadfield:$(BINTRAY_APIKEY) "https://api.bintray.com/content/jonhadfield/subtocheck/$(BUILD_TAG)/subtocheck_openbsd_amd64;bt_package=subtocheck;bt_version=$(BUILD_TAG);publish=1"
	curl -X PUT -0 -T .local_dist/subtocheck_freebsd_amd64 -ujonhadfield:$(BINTRAY_APIKEY) "https://api.bintray.com/content/jonhadfield/subtocheck/$(BUILD_TAG)/subtocheck_freebsd_amd64;bt_package=subtocheck;bt_version=$(BUILD_TAG);publish=1"
	curl -XPOST -0 -ujonhadfield:$(BINTRAY_APIKEY) https://api.bintray.com/content/jonhadfield/subtocheck/$(BUILD_TAG)/publish

release: build-all bintray wait-for-publish build-docker release-docker

wait-for-publish:
	sleep 120

build-docker:
	cd docker ; docker build --no-cache -t quay.io/jonhadfield/subtocheck:$(BUILD_TAG) .
	cd docker ; docker tag quay.io/jonhadfield/subtocheck:$(BUILD_TAG) quay.io/jonhadfield/subtocheck:latest

release-docker:
	cd docker ; docker push quay.io/jonhadfield/subtocheck:$(BUILD_TAG)
	cd docker ; docker push quay.io/jonhadfield/subtocheck:latest

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := build
