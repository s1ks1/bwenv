package envrc

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const migrationBackupPath = ".envrc.bwenv.bak"

const (
	generatedDirenvLogFormat = `export DIRENV_LOG_FORMAT=$'\033[2m  \U0001f510 %s\033[0m'`
	generatedDirenvTimeout   = `export DIRENV_WARN_TIMEOUT="10m"`
)

// MigrateProject converts a generated legacy .envrc to canonical project metadata.
func MigrateProject(dryRun bool) (string, error) {
	if _, err := os.Lstat(".bwenv.toml"); err == nil {
		return "", fmt.Errorf(".bwenv.toml already exists; this project does not need migration")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("check .bwenv.toml: %w", err)
	}

	original, err := os.ReadFile(".envrc")
	if err != nil {
		return "", fmt.Errorf("read legacy .envrc: %w", err)
	}
	providerSlug, folderName, folderID, itemIDs, err := ParseEnvrcConfigWithFolderID()
	if err != nil {
		return "", fmt.Errorf("could not read legacy bwenv settings: %w", err)
	}

	projectConfig := ProjectConfig{
		Version:  projectConfigVersion,
		Provider: providerSlug,
		Project:  ProjectMetadata{FolderID: folderID, FolderName: folderName, Items: itemIDs},
		Activation: ActivationConfig{
			Mode: "direnv",
		},
	}
	configData, err := encodeProjectConfig(projectConfig)
	if err != nil {
		return "", fmt.Errorf("legacy settings cannot be migrated: %w", err)
	}
	activationFile, hasSession, err := migratedEnvrc(original)
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(migrationBackupPath); err == nil {
		return "", fmt.Errorf("backup %s already exists; move it before retrying", migrationBackupPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("check migration backup: %w", err)
	}

	if dryRun {
		plan := "Legacy bwenv project detected.\nPlanned changes:\n  + create .bwenv.toml\n  ~ replace .envrc with a canonical activation command\n  + save the original .envrc as " + migrationBackupPath + " (mode 0600 where supported)\n"
		if hasSession {
			plan += "  = retain BW_SESSION in .envrc for v2 compatibility\n"
		}
		return plan + "No files changed.\n", nil
	}

	configTemp, err := writeMigrationTemp(".bwenv.toml-tmp-*", configData, 0644)
	if err != nil {
		return "", err
	}
	defer os.Remove(configTemp)
	activationTemp, err := writeMigrationTemp(".envrc.bwenv-tmp-*", activationFile, 0600)
	if err != nil {
		return "", err
	}
	defer os.Remove(activationTemp)

	if err := writeMigrationBackup(original); err != nil {
		return "", err
	}
	if err := os.Rename(configTemp, ".bwenv.toml"); err != nil {
		return "", fmt.Errorf("write .bwenv.toml; original .envrc is unchanged and backed up: %w", err)
	}
	if err := os.Remove(".envrc"); err != nil {
		removeErr := os.Remove(".bwenv.toml")
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return "", fmt.Errorf("remove legacy .envrc: %w; remove partial project config: %v; original is backed up at %s", err, removeErr, migrationBackupPath)
		}
		return "", fmt.Errorf("remove legacy .envrc: %w; original backup: %s", err, migrationBackupPath)
	}
	if err := os.Rename(activationTemp, ".envrc"); err != nil {
		restoreErr := os.WriteFile(".envrc", original, 0600)
		removeErr := os.Remove(".bwenv.toml")
		if restoreErr != nil {
			return "", fmt.Errorf("replace .envrc: %w; restore original: %v; remove project config: %v; recover from %s", err, restoreErr, removeErr, migrationBackupPath)
		}
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return "", fmt.Errorf("replace .envrc: %w; original restored; remove project config: %v; backup: %s", err, removeErr, migrationBackupPath)
		}
		return "", fmt.Errorf("replace .envrc: %w; original restored; backup: %s", err, migrationBackupPath)
	}

	return "Migration complete. Original .envrc backed up to " + migrationBackupPath + ".\n", nil
}

func migratedEnvrc(original []byte) ([]byte, bool, error) {
	var preserved []string
	var hasSession bool
	var exportCommands int
	for _, line := range strings.Split(string(original), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case line == generatedDirenvLogFormat:
			preserved = append(preserved, line)
		case line == generatedDirenvTimeout:
			preserved = append(preserved, line)
		case strings.HasPrefix(line, "export BW_SESSION="):
			if hasSession {
				return nil, false, fmt.Errorf("legacy .envrc contains multiple BW_SESSION assignments; review it manually")
			}
			sessionValue := strings.TrimPrefix(line, "export BW_SESSION=")
			session, ok := unquoteShellValue(sessionValue)
			if !ok || shellQuote(session) != sessionValue {
				return nil, false, fmt.Errorf("legacy .envrc contains a non-literal BW_SESSION assignment; review it manually")
			}
			hasSession = true
			preserved = append(preserved, line)
		case strings.HasPrefix(line, `eval "$(bwenv export `) && strings.HasSuffix(line, `)"`):
			exportCommands++
		default:
			return nil, hasSession, fmt.Errorf("legacy .envrc contains custom shell code; migrate it manually to avoid losing behavior")
		}
	}
	if exportCommands != 1 {
		return nil, hasSession, fmt.Errorf("legacy .envrc must contain exactly one generated bwenv export command")
	}

	var b strings.Builder
	b.WriteString("# Generated by bwenv migrate.\n")
	b.WriteString("# Project references are stored in .bwenv.toml.\n")
	if hasSession {
		b.WriteString("# BW_SESSION remains in .envrc until runtime session management is available.\n")
	}
	for _, line := range preserved {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("eval \"$(bwenv export --project .)\"\n")
	return []byte(b.String()), hasSession, nil
}

func unquoteShellValue(value string) (string, bool) {
	if len(value) < 2 || value[0] != '\'' {
		return "", false
	}
	var b strings.Builder
	for i := 1; i < len(value); {
		if value[i] != '\'' {
			b.WriteByte(value[i])
			i++
			continue
		}
		if strings.HasPrefix(value[i:], `'\''`) {
			b.WriteByte('\'')
			i += 4
			continue
		}
		if i == len(value)-1 {
			return b.String(), true
		}
		return "", false
	}
	return "", false
}

func writeMigrationTemp(pattern string, content []byte, mode os.FileMode) (string, error) {
	file, err := os.CreateTemp(".", pattern)
	if err != nil {
		return "", fmt.Errorf("create migration temp file: %w", err)
	}
	name := file.Name()
	keep := false
	defer func() {
		_ = file.Close()
		if !keep {
			_ = os.Remove(name)
		}
	}()
	if err := file.Chmod(mode); err != nil {
		return "", fmt.Errorf("set migration temp permissions: %w", err)
	}
	if _, err := file.Write(content); err != nil {
		return "", fmt.Errorf("write migration temp file: %w", err)
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("sync migration temp file: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close migration temp file: %w", err)
	}
	keep = true
	return name, nil
}

func writeMigrationBackup(original []byte) error {
	file, err := os.OpenFile(migrationBackupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create migration backup %s: %w", migrationBackupPath, err)
	}
	keep := false
	defer func() {
		_ = file.Close()
		if !keep {
			_ = os.Remove(migrationBackupPath)
		}
	}()
	if _, err := file.Write(original); err != nil {
		return fmt.Errorf("write migration backup: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync migration backup: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close migration backup: %w", err)
	}
	keep = true
	return nil
}
