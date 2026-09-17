package process

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"time"

	"github.com/s1ks1/bwenv/internal/diagnostics"
)

type IO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Result struct {
	Stdout []byte
	Stderr []byte
}

// Runner is the sole execution boundary for password-manager CLI commands.
type Runner interface {
	Run(ctx context.Context, name string, args []string, streams IO) (Result, error)
}

type ExecRunner struct {
	Recorder *diagnostics.Recorder
	Timeout  time.Duration
}

func (r ExecRunner) Run(ctx context.Context, name string, args []string, streams IO) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = streams.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if streams.Stdout != nil {
		cmd.Stdout = streams.Stdout
	}
	if streams.Stderr != nil {
		cmd.Stderr = streams.Stderr
	}
	r.Recorder.AddProcess(name)
	err := cmd.Run()
	return Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
}
