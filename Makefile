MODULE   := github.com/jd1100/superach
APP_ID   := io.superach.app
APP_NAME := SuperACH
BIN      := superach
DIST     := dist
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)

.PHONY: help run test vet build build-all clean install-fyne-cross package

help:
	@echo "Targets:"
	@echo "  run              - go run ./cmd/superach"
	@echo "  test             - go test ./..."
	@echo "  vet              - go vet ./..."
	@echo "  build            - host binary into $(DIST)/"
	@echo "  build-all        - cross-compile mac/win/linux with fyne-cross"
	@echo "  package          - host packaged app bundle via fyne"
	@echo "  install-fyne-cross - go install fyne-cross"
	@echo "  clean            - remove $(DIST)/"

run:
	go run ./cmd/superach

test:
	go test ./...

vet:
	go vet ./...

$(DIST):
	mkdir -p $(DIST)

build: $(DIST)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN) ./cmd/superach

install-fyne-cross:
	go install github.com/fyne-io/fyne-cross@latest

# fyne-cross is Docker-based. Needs Docker running. Produces per-OS binaries
# + .app bundles in ./fyne-cross/.
build-all:
	fyne-cross darwin  -arch=amd64,arm64 -app-id $(APP_ID) -name $(APP_NAME) ./cmd/superach
	fyne-cross windows -arch=amd64        -app-id $(APP_ID) -name $(APP_NAME) ./cmd/superach
	fyne-cross linux   -arch=amd64,arm64  -app-id $(APP_ID) -name $(APP_NAME) ./cmd/superach

# Build a host-OS packaged bundle (e.g. .app on macOS) using fyne's own CLI.
package:
	go install fyne.io/tools/cmd/fyne@latest
	fyne package -appID $(APP_ID) -name $(APP_NAME) -sourceDir ./cmd/superach -release

clean:
	rm -rf $(DIST) fyne-cross
