// Package main is the canonical bwenv executable entry point.
package main

import (
	"github.com/s1ks1/bwenv/v3/internal/buildinfo"
	"github.com/s1ks1/bwenv/v3/internal/cli"
	"os"
)

// Version is injected by release builds using -X main.Version.
var Version string

func main() { os.Exit(cli.Run(os.Args[1:], buildinfo.Resolve(Version))) }
