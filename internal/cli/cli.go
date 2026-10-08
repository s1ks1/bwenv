// Package cli validates command syntax before invoking application workflows.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/s1ks1/bwenv/v3/internal/app"
	"github.com/s1ks1/bwenv/v3/internal/output"
)

func Run(args []string, version string) int {
	request, err := Parse(args)
	if err != nil {
		output.Error("Invalid command", err)
		return 1
	}
	request.Version = version
	return app.Run(request)
}

// Parse never contacts a provider, starts a TUI, or writes executable stdout.
func Parse(args []string) (app.Request, error) {
	request := app.Request{Command: "help"}
	if len(args) == 0 {
		return request, nil
	}
	command := args[0]
	aliases := map[string]string{"--help": "help", "-h": "help", "--version": "version", "-v": "version", "load": "export", "clean": "remove", "test": "status", "lock": "logout", "deny": "disallow", "settings": "config", "auth": "login"}
	if canonical, ok := aliases[command]; ok {
		command = canonical
	}
	request.Command = command
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var items string
	switch command {
	case "init":
		flags.StringVar(&request.Activation, "activation", "", "activation backend")
	case "export", "benchmark":
		flags.StringVar(&request.Provider, "provider", "", "provider")
		flags.StringVar(&request.Folder, "folder", "", "folder or vault")
		flags.StringVar(&request.FolderID, "folder-id", "", "folder ID")
		flags.StringVar(&items, "items", "", "comma-separated item IDs")
		if command == "export" {
			flags.StringVar(&request.Project, "project", "", "project path")
			flags.BoolVar(&request.Quiet, "quiet", false, "quiet hook output")
		}
	case "activate", "deactivate", "refresh", "logout":
		flags.StringVar(&request.Shell, "shell", "", "shell dialect")
	case "root":
		flags.BoolVar(&request.ShellOnly, "shell-only", false, "native projects only")
		flags.BoolVar(&request.Fingerprint, "fingerprint", false, "configuration fingerprint")
	case "migrate":
		flags.BoolVar(&request.DryRun, "dry-run", false, "preview migration")
	case "hook", "help", "version", "examples", "allow", "disallow", "remove", "config", "login", "status", "doctor", "login-hint":
	default:
		return request, fmt.Errorf("unknown command %q · run bwenv help", command)
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			request.Command = "help"
			return request, nil
		}
		return request, err
	}
	if command == "hook" && flags.NArg() == 1 {
		request.Shell = flags.Arg(0)
	} else if flags.NArg() != 0 {
		return request, fmt.Errorf("unexpected arguments for %s", command)
	}
	for _, item := range strings.Split(items, ",") {
		if item = strings.TrimSpace(item); item != "" {
			request.Items = append(request.Items, item)
		}
	}
	return request, nil
}
