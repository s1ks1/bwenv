# =============================================================================
# bwenv Makefile — build, install, test, and release targets
#
# Usage:
#   make              Build the binary for the current platform
#   make install      Build and install to ~/.local/bin
#   make test         Run all Go tests
#   make lint         Run the Go linter
#   make clean        Remove build artifacts
#   make release      Build release binaries for all platforms
#   make checksums    Generate SHA256 checksums for release artifacts
#   make help         Show this help message
# =============================================================================

# -- Configuration --

# Application name and module path.
APP_NAME    := bwenv
MODULE      := github.com/s1ks1/bwenv

# Version is derived from the latest git tag, falling back to "dev".
VERSION     := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Go build flags — inject version info at compile time via ldflags.
LDFLAGS     := -s -w \
               -X 'main.Version=$(VERSION)'

# Output directories.
BUILD_DIR   := dist
INSTALL_DIR := $(HOME)/.local/bin

# All target platforms for cross-compilation (OS/ARCH pairs).
PLATFORMS   := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

# -- Default target --

.PHONY: all
all: build

# -- Help --

.PHONY: help
help:
	@echo ""
	@echo "  $(APP_NAME) $(VERSION) — Makefile targets"
	@echo "  ─────────────────────────────────────────"
	@echo ""
	@echo "  Development:"
	@echo "    make build        Build binary for current platform"
	@echo "    make run          Build and run the binary"
	@echo "    make test         Run Go tests"
	@echo "    make lint         Run Go vet and staticcheck"
	@echo "    make fmt          Format all Go source files"
	@echo "    make tidy         Run go mod tidy"
	@echo "    make clean        Remove build artifacts"
	@echo ""
	@echo "  Installation:"
	@echo "    make install      Install binary to $(INSTALL_DIR)"
	@echo "    make uninstall    Remove binary from $(INSTALL_DIR)"
	@echo ""
	@echo "  Release:"
	@echo "    make release      Cross-compile for all platforms"
	@echo "    make checksums    Generate SHA256 checksums for releases"
	@echo ""

# -- Build --

.PHONY: build
build:
	@echo "Building $(APP_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "  ✓ $(BUILD_DIR)/$(APP_NAME)"

# -- Run (build + execute) --

.PHONY: run
run: build
	@$(BUILD_DIR)/$(APP_NAME) $(ARGS)

# -- Install --

.PHONY: install
install: build
	@echo "Installing $(APP_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(APP_NAME) $(INSTALL_DIR)/$(APP_NAME)
	@chmod +x $(INSTALL_DIR)/$(APP_NAME)
	@echo "  ✓ Installed to $(INSTALL_DIR)/$(APP_NAME)"
	@echo ""
	@# Check if the install directory is in PATH.
	@if echo "$$PATH" | tr ':' '\n' | grep -qx "$(INSTALL_DIR)"; then \
		echo "  ✓ $(INSTALL_DIR) is in your PATH"; \
	else \
		echo "  ! $(INSTALL_DIR) is NOT in your PATH"; \
		echo "    Add it with: export PATH=\"$(INSTALL_DIR):$$PATH\""; \
	fi
	@echo ""
	@echo "  Run '$(APP_NAME) test' to verify your setup."

# -- Uninstall --

.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(APP_NAME)..."
	@rm -f $(INSTALL_DIR)/$(APP_NAME)
	@echo "  ✓ Removed $(INSTALL_DIR)/$(APP_NAME)"

# -- Test --

.PHONY: test
test:
	@echo "Running tests..."
	go test -v -race -count=1 ./...

# -- Lint --

.PHONY: lint
lint:
	@echo "Running linters..."
	go vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "  (staticcheck not installed — skipping, install with: go install honnef.co/go/tools/cmd/staticcheck@latest)"; \
	fi

# -- Format --

.PHONY: fmt
fmt:
	@echo "Formatting Go files..."
	gofmt -s -w .
	@echo "  ✓ Done"

# -- Tidy --

.PHONY: tidy
tidy:
	@echo "Tidying Go modules..."
	go mod tidy
	@echo "  ✓ Done"

# -- Clean --

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "  ✓ Removed $(BUILD_DIR)/"

# -- Cross-platform release build --

.PHONY: release
release: clean
	@echo "Building release $(VERSION) for all platforms..."
	@echo ""
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		output_name="$(APP_NAME)-$(VERSION)-$${os}-$${arch}"; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "  Building $${os}/$${arch}..."; \
		GOOS=$$os GOARCH=$$arch go build \
			-ldflags "$(LDFLAGS)" \
			-o "$(BUILD_DIR)/$${output_name}/$(APP_NAME)$${ext}" . || exit 1; \
		cp LICENSE "$(BUILD_DIR)/$${output_name}/"; \
		cp README.md "$(BUILD_DIR)/$${output_name}/"; \
		if [ "$$os" = "windows" ]; then \
			cd $(BUILD_DIR) && zip -rq "$${output_name}.zip" "$${output_name}/" && cd ..; \
		else \
			tar -czf "$(BUILD_DIR)/$${output_name}.tar.gz" -C $(BUILD_DIR) "$${output_name}/"; \
		fi; \
		rm -rf "$(BUILD_DIR)/$${output_name}"; \
		echo "    ✓ $${output_name}"; \
	done
	@echo ""
	@echo "  Release archives:"
	@ls -lh $(BUILD_DIR)/*.tar.gz $(BUILD_DIR)/*.zip 2>/dev/null
	@echo ""
	@$(MAKE) checksums --no-print-directory

# -- Checksums --

.PHONY: checksums
checksums:
	@echo "Generating checksums..."
	@cd $(BUILD_DIR) && \
		if command -v sha256sum >/dev/null 2>&1; then \
			sha256sum *.tar.gz *.zip 2>/dev/null > checksums.txt; \
		elif command -v shasum >/dev/null 2>&1; then \
			shasum -a 256 *.tar.gz *.zip 2>/dev/null > checksums.txt; \
		else \
			echo "  ! No sha256sum or shasum found"; exit 1; \
		fi
	@echo "  ✓ $(BUILD_DIR)/checksums.txt"
