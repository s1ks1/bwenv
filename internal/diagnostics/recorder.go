package diagnostics

import (
	"sort"
	"sync"
	"time"
)

// Recorder stores timings and process counts, never command arguments or output.
type Recorder struct {
	mu        sync.Mutex
	stages    map[string]time.Duration
	processes map[string]int
}

type Stage struct {
	Name     string
	Duration time.Duration
}

type Process struct {
	Name  string
	Count int
}

func NewRecorder() *Recorder {
	return &Recorder{stages: make(map[string]time.Duration), processes: make(map[string]int)}
}

func (r *Recorder) Start(name string) func() {
	if r == nil {
		return func() {}
	}
	start := time.Now()
	return func() { r.AddStage(name, time.Since(start)) }
}

func (r *Recorder) AddStage(name string, duration time.Duration) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.stages[name] += duration
	r.mu.Unlock()
}

func (r *Recorder) AddProcess(name string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.processes[name]++
	r.mu.Unlock()
}

func (r *Recorder) Snapshot() ([]Stage, []Process) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stages := make([]Stage, 0, len(r.stages))
	for name, duration := range r.stages {
		stages = append(stages, Stage{Name: name, Duration: duration})
	}
	processes := make([]Process, 0, len(r.processes))
	for name, count := range r.processes {
		processes = append(processes, Process{Name: name, Count: count})
	}
	sort.Slice(stages, func(i, j int) bool { return stages[i].Name < stages[j].Name })
	sort.Slice(processes, func(i, j int) bool { return processes[i].Name < processes[j].Name })
	return stages, processes
}
