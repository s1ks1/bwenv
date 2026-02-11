SHELL := /bin/bash
OS := $(shell uname -s 2>/dev/null || echo Windows_NT)

INSTALL_LIB := $(HOME)/.config/direnv/lib
INSTALL_BIN := $(HOME)/.local/bin

VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.0")

all: install

help:
	@echo "bwenv $(VERSION) - Available commands:"
	@echo ""
	@echo "  make install      Install bwenv CLI and helper script"
	@echo "  make uninstall    Remove bwenv CLI and helper script"
	@echo "  make setup-path   Add ~/.local/bin to PATH (Linux/macOS)"
	@echo "  make check-deps   Check required dependencies"
	@echo "  make help         Show this help message"

check-deps:
	@echo "Checking dependencies..."
	@command -v bw >/dev/null 2>&1 && echo "  [OK] bw (Bitwarden CLI)" || echo "  [MISSING] bw - https://bitwarden.com/help/cli/"
	@command -v direnv >/dev/null 2>&1 && echo "  [OK] direnv" || echo "  [MISSING] direnv - https://direnv.net/"
	@command -v jq >/dev/null 2>&1 && echo "  [OK] jq" || echo "  [MISSING] jq - https://stedolan.github.io/jq/"

install:
	@echo "Installing bwenv..."
	@mkdir -p $(INSTALL_LIB)
	@cp setup/bitwarden_folders.sh $(INSTALL_LIB)/bitwarden_folders.sh
	@chmod +x $(INSTALL_LIB)/bitwarden_folders.sh
	@mkdir -p $(INSTALL_BIN)
	@cp setup/bwenv $(INSTALL_BIN)/bwenv
	@chmod +x $(INSTALL_BIN)/bwenv
	@echo "[OK] bwenv installed"
	@echo ""
	@if echo "$$PATH" | tr ':' '\n' | grep -qx "$(INSTALL_BIN)"; then \
		echo "[OK] $(INSTALL_BIN) is in your PATH"; \
	else \
		echo "[WARN] $(INSTALL_BIN) is not in your PATH"; \
		echo "   Run 'make setup-path' or add it manually to your shell config"; \
	fi
	@echo ""
	@if command -v direnv >/dev/null 2>&1; then \
		SHELL_NAME=$$(basename "$$SHELL"); \
		if [ "$$SHELL_NAME" = "bash" ] && [ -f ~/.bashrc ] && ! grep -q "direnv hook bash" ~/.bashrc; then \
			echo 'eval "$$(direnv hook bash)"' >> ~/.bashrc; \
			echo "[OK] Added direnv hook to ~/.bashrc"; \
		fi; \
		if [ "$$SHELL_NAME" = "zsh" ] && [ -f ~/.zshrc ] && ! grep -q "direnv hook zsh" ~/.zshrc; then \
			echo 'eval "$$(direnv hook zsh)"' >> ~/.zshrc; \
			echo "[OK] Added direnv hook to ~/.zshrc"; \
		fi; \
	else \
		echo "[WARN] direnv not installed - hook setup skipped"; \
	fi
	@echo ""
	@echo "Run 'bwenv test' to verify your setup."

setup-path:
	@SHELL_NAME=$$(basename "$$SHELL"); \
	case "$$SHELL_NAME" in \
		bash) RC="$$HOME/.bashrc" ;; \
		zsh)  RC="$$HOME/.zshrc" ;; \
		fish) RC="$$HOME/.config/fish/config.fish" ;; \
		*)    echo "[WARN] Unknown shell: $$SHELL_NAME - add $(INSTALL_BIN) to PATH manually"; exit 0 ;; \
	esac; \
	if [ -f "$$RC" ]; then \
		if ! grep -q "$(INSTALL_BIN)" "$$RC"; then \
			if [ "$$SHELL_NAME" = "fish" ]; then \
				echo 'set -gx PATH $(INSTALL_BIN) $$PATH' >> "$$RC"; \
			else \
				echo 'export PATH="$(INSTALL_BIN):$$PATH"' >> "$$RC"; \
			fi; \
			echo "[OK] Added $(INSTALL_BIN) to $$RC"; \
			echo "   Restart your terminal or run: source $$RC"; \
		else \
			echo "[OK] $(INSTALL_BIN) already in $$RC"; \
		fi; \
	else \
		echo "[WARN] $$RC not found - add $(INSTALL_BIN) to PATH manually"; \
	fi

uninstall:
	@echo "Uninstalling bwenv..."
	@rm -f $(INSTALL_LIB)/bitwarden_folders.sh
	@rm -f $(INSTALL_BIN)/bwenv
	@echo "[OK] bwenv removed"

.PHONY: all help install uninstall setup-path check-deps