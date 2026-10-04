// fakecli is a deterministic stand-in for bw and op in integration tests.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if path := os.Getenv("BWENV_FAKE_LOG"); path != "" {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			os.Exit(2)
		}
		_, _ = fmt.Fprintln(file, filepath.Base(os.Args[0])+" "+strings.Join(os.Args[1:min(3, len(os.Args))], " "))
		_ = file.Close()
	}
	if delay := os.Getenv("BWENV_FAKE_DELAY"); delay != "" {
		if duration, err := time.ParseDuration(delay); err == nil {
			time.Sleep(duration)
		}
	}
	args := strings.Join(os.Args[1:], " ")
	if args == "--version" {
		fmt.Print("2026.9.0-fixture")
		return
	}
	if os.Getenv("BWENV_FAKE_SCENARIO") == "lock-error" && (strings.HasPrefix(args, "lock") || strings.HasPrefix(args, "signout")) {
		fmt.Fprintln(os.Stderr, "fake lock failed")
		os.Exit(1)
	}
	switch os.Getenv("BWENV_FAKE_SCENARIO") {
	case "expired":
		fmt.Fprintln(os.Stderr, "session expired")
		os.Exit(1)
	case "provider-error":
		fmt.Fprintln(os.Stderr, "provider unavailable")
		os.Exit(1)
	case "malformed":
		fmt.Print("not-json")
		return
	}
	switch {
	case strings.HasPrefix(args, "list folders"):
		fmt.Print(`[{"id":"folder-1","name":"Fixture"}]`)
	case strings.HasPrefix(args, "list items"):
		if os.Getenv("BWENV_FAKE_SCENARIO") == "large" {
			items := make([]map[string]any, 1000)
			for i := range items {
				items[i] = map[string]any{"id": fmt.Sprintf("item-%d", i), "name": "Fixture", "fields": []map[string]string{{"name": fmt.Sprintf("KEY_%d", i), "value": "fake-large-value"}}}
			}
			_ = json.NewEncoder(os.Stdout).Encode(items)
			return
		}
		fmt.Print(`[{"id":"item-1","name":"Test","fields":[{"name":"API_KEY","value":"fake-secret-value"}]},{"id":"item-2","name":"Database","fields":[{"name":"DB_URL","value":"fake-database-value"}]}]`)
	case strings.HasPrefix(args, "get item"):
		fmt.Print(`{"id":"item-1","name":"Test","fields":[{"name":"API_KEY","value":"fake-secret-value"}]}`)
	case strings.HasPrefix(args, "unlock"):
		fmt.Print("fake-session")
	case strings.HasPrefix(args, "vault list"):
		fmt.Print(`[{"id":"vault-1","name":"Fixture"}]`)
	case strings.HasPrefix(args, "item list"):
		if os.Getenv("BWENV_FAKE_SCENARIO") == "selection-change" {
			fmt.Print(`[{"id":"item-1","title":"Test"},{"id":"item-2","title":"New note"}]`)
		} else {
			fmt.Print(`[{"id":"item-1","title":"Test"}]`)
		}
	case strings.HasPrefix(args, "item get"):
		if os.Getenv("BWENV_FAKE_SCENARIO") != "selection-change" {
			fmt.Print(`{"id":"item-1","title":"Test","fields":[{"label":"API_KEY","value":"fake-secret-value","type":"CONCEALED"}]}`)
		} else if len(os.Args) > 3 && os.Args[3] == "item-2" {
			fmt.Print(`{"id":"item-2","title":"New note","fields":[{"label":"API_KEY","value":"new-note-value","type":"CONCEALED"},{"label":"NEW_ONLY","value":"new-only-value","type":"STRING"}]}`)
		} else {
			fmt.Print(`{"id":"item-1","title":"Test","fields":[{"label":"API_KEY","value":"fake-secret-value","type":"CONCEALED"},{"label":"OLD_ONLY","value":"old-only-value","type":"STRING"}]}`)
		}
	case strings.HasPrefix(args, "signin"), strings.HasPrefix(args, "signout"), strings.HasPrefix(args, "sync"), strings.HasPrefix(args, "lock"):
		return
	default:
		fmt.Fprintln(os.Stderr, "unexpected fake CLI command")
		os.Exit(2)
	}
}
