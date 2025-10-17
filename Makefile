SHELL := /bin/bash
OS := $(shell uname 2>/dev/null || echo Windows)

INSTALL_LIB := $(HOME)/.config/direnv/lib
INSTALL_BIN := $(HOME)/.local/bin

all: install

install:
	@echo "🔧 Installing Bitwarden + direnv helper..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "New-Item -ItemType Directory -Force -Path $$env:USERPROFILE\\.config\\direnv\\lib | Out-Null; Copy-Item setup\\bitwarden_folders.sh $$env:USERPROFILE\\.config\\direnv\\lib\\bitwarden_folders.sh; New-Item -ItemType Directory -Force -Path $$env:USERPROFILE\\.local\\bin | Out-Null; Copy-Item setup\\bwenv.bat $$env:USERPROFILE\\.local\\bin\\bwenv.bat; Write-Host '✅ bwenv CLI installed. Use \"bwenv init\" or \"bwenv interactive\" in projects.'; Write-Host '📝 Make sure %USERPROFILE%\\.local\\bin is in your PATH environment variable.'"
else
	@mkdir -p $(INSTALL_LIB)
	@cp setup/bitwarden_folders.sh $(INSTALL_LIB)/
	@chmod +x $(INSTALL_LIB)/bitwarden_folders.sh
	@mkdir -p $(INSTALL_BIN)
	@cp setup/bwenv $(INSTALL_BIN)/bwenv
	@chmod +x $(INSTALL_BIN)/bwenv
	@echo "✅ bwenv CLI installed. Use 'bwenv init' or 'bwenv interactive' in projects."
endif

uninstall:
	@echo "🧹 Removing bwenv installation..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "Remove-Item -Force -ErrorAction SilentlyContinue $$env:USERPROFILE\\.config\\direnv\\lib\\bitwarden_folders.sh; Remove-Item -Force -ErrorAction SilentlyContinue $$env:USERPROFILE\\.local\\bin\\bwenv.bat; Write-Host '✅ bwenv removed'"
else
	@rm -f $(INSTALL_LIB)/bitwarden_folders.sh
	@rm -f $(INSTALL_BIN)/bwenv
	@echo "✅ bwenv removed"
endif