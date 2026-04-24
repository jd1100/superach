MODULE   := github.com/jd1100/superach
APP_ID   := io.superach.app
APP_NAME := SuperACH
BIN      := superach
DIST     := dist
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
# .exe on Windows, empty elsewhere. `go build -o` does NOT auto-append this
# when -o is passed explicitly, so we have to do it ourselves.
GOEXE    := $(shell go env GOEXE)

.PHONY: help run test vet build build-all clean install-fyne-cross package check-cgo

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

# Fyne pulls in github.com/go-gl/gl, which is cgo-only. Without a C compiler on
# PATH, Go silently flips CGO_ENABLED=0 and every file in that package fails
# its build tag, producing the confusing:
#   build constraints exclude all Go files in .../go-gl/gl/v2.1/gl
# Catch that up front with a clear message.
check-cgo:
	@CGO_ENABLED=$$(go env CGO_ENABLED); \
	if [ "$$CGO_ENABLED" != "1" ]; then \
		echo ""; \
		echo "ERROR: CGO is disabled (go env CGO_ENABLED=$$CGO_ENABLED)."; \
		echo ""; \
		echo "Fyne requires cgo + a C compiler on PATH. Install one and reopen your shell:"; \
		echo '  Windows: MSYS2  ->  pacman -S mingw-w64-x86_64-toolchain'; \
		printf '                     then add %s to PATH\n' 'C:\msys64\mingw64\bin'; \
		echo "           or install TDM-GCC / w64devkit."; \
		echo "  macOS:   xcode-select --install"; \
		echo "  Linux:   install gcc + the packages at https://docs.fyne.io/started/"; \
		echo ""; \
		exit 1; \
	fi

run: check-cgo
	go run ./cmd/superach

test:
	go test ./...

vet:
	go vet ./...

$(DIST):
	mkdir -p $(DIST)

build: check-cgo $(DIST)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN)$(GOEXE) ./cmd/superach

install-fyne-cross:
	go install github.com/fyne-io/fyne-cross@latest

# fyne-cross is Docker-based. Needs Docker running. Produces per-OS binaries
# + .app bundles in ./fyne-cross/.
build-all:
	fyne-cross darwin  -arch=amd64,arm64 -app-id $(APP_ID) -name $(APP_NAME) ./cmd/superach
	fyne-cross windows -arch=amd64        -app-id $(APP_ID) -name $(APP_NAME) ./cmd/superach
	fyne-cross linux   -arch=amd64,arm64  -app-id $(APP_ID) -name $(APP_NAME) ./cmd/superach

# Build a host-OS packaged bundle (e.g. .app on macOS) using fyne's own CLI.
package: check-cgo
	go install fyne.io/tools/cmd/fyne@latest
	fyne package -appID $(APP_ID) -name $(APP_NAME) -sourceDir ./cmd/superach -release

clean:
	rm -rf $(DIST) fyne-cross
