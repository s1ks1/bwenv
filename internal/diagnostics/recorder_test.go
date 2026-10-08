package diagnostics

import (
	"testing"
	"time"
)

func TestRecorderAccumulatesAndSortsStages(t *testing.T) {
	r := NewRecorder()
	r.AddStage("fetch", 5*time.Millisecond)
	r.AddStage("fetch", 3*time.Millisecond)
	r.AddStage("auth", time.Millisecond)

	stages, _ := r.Snapshot()
	if len(stages) != 2 {
		t.Fatalf("got %d stages, want 2: %+v", len(stages), stages)
	}
	if stages[0].Name != "auth" || stages[1].Name != "fetch" {
		t.Fatalf("stages are not sorted by name: %+v", stages)
	}
	if stages[1].Duration != 8*time.Millisecond {
		t.Fatalf("repeated stage duration = %v, want 8ms", stages[1].Duration)
	}
}

func TestRecorderCountsProcesses(t *testing.T) {
	r := NewRecorder()
	r.AddProcess("bw")
	r.AddProcess("bw")
	r.AddProcess("op")

	_, processes := r.Snapshot()
	if len(processes) != 2 {
		t.Fatalf("got %d processes, want 2: %+v", len(processes), processes)
	}
	if processes[0].Name != "bw" || processes[0].Count != 2 {
		t.Fatalf("bitwarden process count = %+v, want bw=2", processes[0])
	}
}

func TestRecorderStartMeasuresElapsed(t *testing.T) {
	r := NewRecorder()
	stop := r.Start("work")
	time.Sleep(2 * time.Millisecond)
	stop()

	stages, _ := r.Snapshot()
	if len(stages) != 1 || stages[0].Name != "work" {
		t.Fatalf("unexpected stages: %+v", stages)
	}
	if stages[0].Duration < 2*time.Millisecond {
		t.Fatalf("measured duration %v, want at least 2ms", stages[0].Duration)
	}
}

func TestNilRecorderWritesAreSafe(t *testing.T) {
	var r *Recorder
	r.AddStage("x", time.Second) // must not panic
	r.AddProcess("bw")           // must not panic
	stop := r.Start("y")         // must not panic
	stop()                       // must not panic
}
