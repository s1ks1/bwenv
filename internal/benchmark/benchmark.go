package benchmark

import (
	"fmt"
	"io"
	"time"

	"github.com/s1ks1/bwenv/internal/diagnostics"
	"github.com/s1ks1/bwenv/internal/process"
	"github.com/s1ks1/bwenv/internal/provider"
)

// BenchmarkReport contains only timing and process metadata.
type BenchmarkReport struct {
	Total     time.Duration
	Stages    []diagnostics.Stage
	Processes []diagnostics.Process
	Variables int
}

// Benchmark follows the current non-interactive provider path without printing secrets.
func Benchmark(providerSlug, folderName string, itemIDs []string) (BenchmarkReport, error) {
	recorder := diagnostics.NewRecorder()
	start := time.Now()
	p, err := provider.GetWithRunner(providerSlug, process.ExecRunner{Recorder: recorder})
	if err != nil {
		return BenchmarkReport{}, fmt.Errorf("unknown provider")
	}
	if !p.IsAvailable() {
		return BenchmarkReport{}, fmt.Errorf("provider CLI is not installed")
	}

	stop := recorder.Start("session check")
	authenticated := p.IsAuthenticated()
	stop()
	if !authenticated {
		return BenchmarkReport{}, fmt.Errorf("session unavailable; run bwenv login")
	}
	stop = recorder.Start("authentication")
	session, err := p.Authenticate()
	stop()
	if err != nil {
		return BenchmarkReport{}, fmt.Errorf("authentication failed; run bwenv login")
	}
	stop = recorder.Start("folder resolution")
	folders, err := p.ListFolders(session)
	stop()
	if err != nil {
		return BenchmarkReport{}, fmt.Errorf("folder lookup failed")
	}
	var target *provider.Folder
	for i := range folders {
		if folders[i].Name == folderName {
			target = &folders[i]
			break
		}
	}
	if target == nil {
		return BenchmarkReport{}, fmt.Errorf("folder not found")
	}
	stop = recorder.Start("secret fetch")
	var secrets []provider.Secret
	if len(itemIDs) > 0 {
		secrets, err = p.GetSecretsByItemIDs(session, itemIDs)
	} else {
		secrets, err = p.GetSecrets(session, *target)
	}
	stop()
	if err != nil {
		return BenchmarkReport{}, fmt.Errorf("secret fetch failed")
	}
	stages, processes := recorder.Snapshot()
	return BenchmarkReport{Total: time.Since(start), Stages: stages, Processes: processes, Variables: len(secrets)}, nil
}

func (r BenchmarkReport) Print(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "bwenv benchmark"); err != nil {
		return err
	}
	for _, stage := range r.Stages {
		if _, err := fmt.Fprintf(w, "%-20s %8.1f ms\n", stage.Name, float64(stage.Duration)/float64(time.Millisecond)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "%-20s %8.1f ms\n", "total", float64(r.Total)/float64(time.Millisecond)); err != nil {
		return err
	}
	count := 0
	for _, p := range r.Processes {
		count += p.Count
	}
	_, err := fmt.Fprintf(w, "provider processes: %d\nvariables found: %d\n", count, r.Variables)
	return err
}
