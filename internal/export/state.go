package export

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/s1ks1/bwenv/v3/internal/provider"
	"github.com/s1ks1/bwenv/v3/internal/shell"
)

const stateVariable = "_BWENV_STATE"
const lockedVariable = "_BWENV_LOCKED"

type environmentState struct {
	Root   string
	Values map[string]*string
}

func currentShell() string { return filepath.Base(os.Getenv("SHELL")) }

func validateShell(name string) error {
	if name != "bash" && name != "zsh" && name != "fish" {
		return fmt.Errorf("unsupported shell %q", name)
	}
	return nil
}

func validName(name string) bool {
	return name != "" && shell.SanitizeKey(name) == name && !strings.HasPrefix(name, "_BWENV_") && !strings.HasPrefix(name, "_bwenv_")
}

func assignment(name, value, shellName string) string {
	if shellName == "fish" {
		quoted := "'" + strings.ReplaceAll(strings.ReplaceAll(value, "\\", "\\\\"), "'", "\\'") + "'"
		return fmt.Sprintf("set -gx %s %s\n", name, quoted)
	}
	return fmt.Sprintf("export %s=%s\n", name, shell.Quote(value))
}

func unassignment(name, shellName string) string {
	if shellName == "fish" {
		return "set -e " + name + "\n"
	}
	return "unset " + name + "\n"
}

func decodeState(encoded string) (environmentState, error) {
	var state environmentState
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		err = json.Unmarshal(data, &state)
	}
	if err != nil || state.Values == nil || state.Root == "" {
		return state, fmt.Errorf("invalid bwenv environment state; start a fresh shell")
	}
	for name := range state.Values {
		if !validName(name) || (state.Values[name] != nil && strings.ContainsRune(*state.Values[name], 0)) {
			return state, fmt.Errorf("invalid variable name or value in bwenv environment state")
		}
	}
	return state, nil
}

func rememberState(secrets []provider.Secret) (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	state := environmentState{Root: root, Values: map[string]*string{}}
	if encoded, ok := os.LookupEnv(stateVariable); ok {
		state, err = decodeState(encoded)
		if err != nil {
			return "", err
		}
		if state.Root != root {
			return "", fmt.Errorf("deactivate the previous bwenv project first")
		}
	}
	for _, secret := range secrets {
		name := shell.SanitizeKey(secret.Key)
		if !validName(name) || (state.Values[name] != nil && strings.ContainsRune(*state.Values[name], 0)) {
			return "", fmt.Errorf("reserved environment variable %q", name)
		}
		if _, saved := state.Values[name]; !saved {
			state.Values[name] = nil
			if value, exists := os.LookupEnv(name); exists {
				state.Values[name] = &value
			}
		}
	}
	data, err := json.Marshal(state)
	return base64.StdEncoding.EncodeToString(data), err
}

func restoreState(encoded, shellName string) ([]string, error) {
	state, err := decodeState(encoded)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(state.Values))
	for name := range state.Values {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if value := state.Values[name]; value != nil {
			fmt.Print(assignment(name, *value, shellName))
		} else {
			fmt.Print(unassignment(name, shellName))
		}
	}
	fmt.Print(unassignment(stateVariable, shellName))
	return names, nil
}
