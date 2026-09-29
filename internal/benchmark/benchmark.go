package benchmark

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/s1ks1/bwenv/v3/internal/diagnostics"
	"github.com/s1ks1/bwenv/v3/internal/process"
	"github.com/s1ks1/bwenv/v3/internal/provider"
)

// BenchmarkReport contains only timing and process metadata.
type BenchmarkReport struct {
	Total     time.Duration
	Stages    []diagnostics.Stage
	Processes []diagnostics.Process
	Variables int
}

// Benchmark follows the current non-interactive provider path without printing secrets.
func Benchmark(ctx context.Context, providerSlug, folderName, folderID string, itemIDs []string) (BenchmarkReport, error) {
	recorder := diagnostics.NewRecorder()
	start := time.Now()
	p, err := provider.GetWithRunner(providerSlug, process.ExecRunner{Recorder: recorder})
	if err != nil {
		return BenchmarkReport{}, fmt.Errorf("unknown provider")
	}
	if !p.IsAvailable() {
		return BenchmarkReport{}, fmt.Errorf("provider CLI is not installed")
	}

	auth, err := provider.AsAuthenticator(p)
	if err != nil {
		return BenchmarkReport{}, fmt.Errorf("provider does not support authentication")
	}

	stop := recorder.Start("session check")
	session, err := auth.AuthenticateNonInteractive(ctx)
	stop()
	if err != nil {
		return BenchmarkReport{}, fmt.Errorf("session unavailable; run bwenv login")
	}
	target := provider.Folder{ID: folderID, Name: folderName}
	if folderID == "" {
		lister, listerErr := provider.AsFolderLister(p)
		if listerErr != nil {
			return BenchmarkReport{}, fmt.Errorf("provider does not expose folders")
		}
		stop = recorder.Start("folder resolution")
		folders, listErr := lister.ListFolders(ctx, session)
		stop()
		if listErr != nil {
			return BenchmarkReport{}, fmt.Errorf("folder lookup failed")
		}
		for _, folder := range folders {
			if folder.Name == folderName {
				target = folder
				break
			}
		}
		if target.ID == "" {
			return BenchmarkReport{}, fmt.Errorf("folder not found")
		}
	}
	fetcher, fetchErr := provider.AsSecretFetcher(p)
	if fetchErr != nil {
		return BenchmarkReport{}, fmt.Errorf("provider cannot fetch secrets")
	}
	stop = recorder.Start("secret fetch")
	var secrets []provider.Secret
	if len(itemIDs) > 0 {
		secrets, err = fetcher.GetSecretsByItemIDs(ctx, session, target, itemIDs)
	} else {
		secrets, err = fetcher.GetSecrets(ctx, session, target)
	}
	stop()
	if err != nil {
		return BenchmarkReport{}, fmt.Errorf("secret fetch failed; session may have expired; run bwenv login")
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
