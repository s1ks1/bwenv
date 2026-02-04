SHELL := /bin/bash
OS := $(shell uname 2>/dev/null || echo Windows_NT)

INSTALL_LIB := $(HOME)/.config/direnv/lib
INSTALL_BIN := $(HOME)/.local/bin

all: install

help:
	@echo "🔧 Bitwarden + direnv helper - Available commands:"
	@echo "  make install     - Install bwenv CLI"
	@echo "  make setup-path  - Add ~/.local/bin to PATH automatically"
	@echo "  make uninstall   - Remove bwenv CLI"
	@echo "  make help        - Show this help message"

install:
	@echo "🔧 Installing Bitwarden + direnv helper..."
ifneq (,$(findstring NT,$(OS)))
	@powershell -Command "New-Item -ItemType Directory -Force -Path \$$env:USERPROFILE\\.config\\direnv\\lib | Out-Null; Copy-Item setup\\bitwarden_folders.sh \$$env:USERPROFILE\\.config\\direnv\\lib\\bitwarden_folders.sh; New-Item -ItemType Directory -Force -Path \$$env:USERPROFILE\\.local\\bin | Out-Null; Copy-Item setup\\bwenv.bat \$$env:USERPROFILE\\.local\\bin\\bwenv.bat; Write-Host '✅ bwenv CLI installed. Use \"bwenv init\" or \"bwenv interactive\" in projects.'; Write-Host '📝 Make sure %USERPROFILE%\\.local\\bin is in your PATH environment variable.'"
else
	@mkdir -p $(INSTALL_LIB)
	@cp setup/bitwarden_folders.sh $(INSTALL_LIB)/
	@chmod +x $(INSTALL_LIB)/bitwarden_folders.sh
	@mkdir -p $(INSTALL_BIN)
	@cp setup/bwenv $(INSTALL_BIN)/bwenv
	@chmod +x $(INSTALL_BIN)/bwenv
	@echo "✅ bwenv CLI installed. Use 'bwenv init' or 'bwenv interactive' in projects."
	@echo ""
	@echo "📝 Important: Make sure $(INSTALL_BIN) is in your PATH"
	@if ! echo "$$PATH" | grep -q "$(INSTALL_BIN)"; then \
		echo "⚠️  $(INSTALL_BIN) is not in your PATH"; \
		echo "   To fix this automatically, run: make setup-path"; \
		echo "   Or manually add this line to your shell config file (~/.bashrc, ~/.zshrc, etc.):"; \
		echo "   export PATH=\"$(INSTALL_BIN):\$$PATH\""; \
		echo "   Then restart your terminal or run: source ~/.bashrc"; \
	else \
		echo "✅ $(INSTALL_BIN) is already in your PATH"; \
	fi
	@echo ""
	@echo "📝 Setting up direnv hook..."
	@if command -v direnv >/dev/null 2>&1; then \
		if [ -f ~/.bashrc ] && ! grep -q "direnv hook bash" ~/.bashrc; then \
			echo 'eval "$$(direnv hook bash)"' >> ~/.bashrc; \
			echo "✅ Added direnv hook to ~/.bashrc"; \
		fi; \
		if [ -f ~/.zshrc ] && ! grep -q "direnv hook zsh" ~/.zshrc; then \
			echo 'eval "$$(direnv hook zsh)"' >> ~/.zshrc; \
			echo "✅ Added direnv hook to ~/.zshrc"; \
		fi; \
		echo "📝 Please restart your terminal for direnv to work properly"; \
	else \
		echo "⚠️  direnv is not installed. Please install it first:"; \
		echo "   Ubuntu/Debian: sudo apt install direnv"; \
		echo "   Arch: sudo pacman -S direnv"; \
		echo "   macOS: brew install direnv"; \
	fi
endif

setup-path:
	@echo "🔧 Setting up PATH for bwenv..."
	@if ! echo "$$PATH" | grep -q "$(INSTALL_BIN)"; then \
		if [ -f ~/.bashrc ]; then \
			if ! grep -q "$(INSTALL_BIN)" ~/.bashrc; then \
				echo 'export PATH="$(INSTALL_BIN):$$PATH"' >> ~/.bashrc; \
				echo "✅ Added $(INSTALL_BIN) to ~/.bashrc"; \
			else \
				echo "✅ $(INSTALL_BIN) already in ~/.bashrc"; \
			fi; \
		fi; \
		if [ -f ~/.zshrc ]; then \
			if ! grep -q "$(INSTALL_BIN)" ~/.zshrc; then \
				echo 'export PATH="$(INSTALL_BIN):$$PATH"' >> ~/.zshrc; \
				echo "✅ Added $(INSTALL_BIN) to ~/.zshrc"; \
			else \
				echo "✅ $(INSTALL_BIN) already in ~/.zshrc"; \
			fi; \
		fi; \
		echo "📝 Please restart your terminal or run: source ~/.bashrc (or ~/.zshrc)"; \
	else \
		echo "✅ $(INSTALL_BIN) is already in your PATH"; \
	fi

uninstall:
	@echo "🧹 Removing bwenv installation..."
ifneq (,$(findstring NT,$(OS)))
	@powershell -Command "Remove-Item -Force -ErrorAction SilentlyContinue \$$env:USERPROFILE\\.config\\direnv\\lib\\bitwarden_folders.sh; Remove-Item -Force -ErrorAction SilentlyContinue \$$env:USERPROFILE\\.local\\bin\\bwenv.bat; Write-Host '✅ bwenv removed'"
else
	@rm -f $(INSTALL_LIB)/bitwarden_folders.sh
	@rm -f $(INSTALL_BIN)/bwenv
	@echo "✅ bwenv removed"
endif