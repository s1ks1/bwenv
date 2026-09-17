// fakecli is a deterministic stand-in for bw and op in integration tests.
package main

import (
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
		fmt.Print(`[{"id":"item-1","name":"Test","fields":[{"name":"API_KEY","value":"fake-secret-value"}]}]`)
	case strings.HasPrefix(args, "get item"):
		fmt.Print(`{"id":"item-1","name":"Test","fields":[{"name":"API_KEY","value":"fake-secret-value"}]}`)
	case strings.HasPrefix(args, "unlock"):
		fmt.Print("fake-session")
	case strings.HasPrefix(args, "vault list"):
		fmt.Print(`[{"id":"vault-1","name":"Fixture"}]`)
	case strings.HasPrefix(args, "item list"):
		fmt.Print(`[{"id":"item-1","title":"Test"}]`)
	case strings.HasPrefix(args, "item get"):
		fmt.Print(`{"id":"item-1","title":"Test","fields":[{"label":"API_KEY","value":"fake-secret-value","type":"CONCEALED"}]}`)
	case strings.HasPrefix(args, "signin"), strings.HasPrefix(args, "signout"), strings.HasPrefix(args, "sync"), strings.HasPrefix(args, "lock"):
		return
	default:
		fmt.Fprintln(os.Stderr, "unexpected fake CLI command")
		os.Exit(2)
	}
}
