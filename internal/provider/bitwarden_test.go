package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/s1ks1/bwenv/v2/internal/process"
)

type syncTestRunner struct {
	name    string
	args    []string
	streams process.IO
	err     error
}

func (r *syncTestRunner) Run(_ context.Context, name string, args []string, streams process.IO) (process.Result, error) {
	r.name = name
	r.args = args
	r.streams = streams
	return process.Result{}, r.err
}

// recordingRunner captures every invocation so tests can inspect argv and the
// child environment without running a real CLI.
type recordingRunner struct {
	names []string
	args  [][]string
	envs  [][]string
}

func (r *recordingRunner) Run(_ context.Context, name string, args []string, streams process.IO) (process.Result, error) {
	r.names = append(r.names, name)
	r.args = append(r.args, args)
	r.envs = append(r.envs, streams.Env)
	return process.Result{}, nil
}

// TestBitwardenSessionGoesViaEnvNotArgv is the PER-26 regression: the session
// token must never appear in the child argv (visible in ps) and must be
// delivered through the BW_SESSION environment variable instead.
func TestBitwardenSessionGoesViaEnvNotArgv(t *testing.T) {
	const session = "s3cr3t-session-token"
	t.Setenv("BW_SESSION", session)
	runner := &recordingRunner{}
	b := &Bitwarden{Runner: runner}

	_, _ = b.ListFolders(session)
	_, _ = b.GetSecrets(session, Folder{ID: "folder-1", Name: "Dev"})
	_, _ = b.GetSecretsByItemIDs(session, Folder{ID: "folder-1", Name: "Dev"}, []string{"item-1"})
	_ = b.IsAuthenticated()

	if len(runner.names) == 0 {
		t.Fatal("no bw invocations recorded")
	}
	for i, args := range runner.args {
		for _, a := range args {
			if a == "--session" || a == session {
				t.Errorf("invocation %d leaked session material in argv: %v", i, args)
			}
		}
		var lastSession string
		for _, kv := range runner.envs[i] {
			if strings.HasPrefix(kv, "BW_SESSION=") {
				lastSession = kv
			}
		}
		if lastSession != "BW_SESSION="+session {
			t.Errorf("invocation %d: last BW_SESSION = %q, want the passed session", i, lastSession)
		}
	}
}

func TestBitwardenSyncRunsQuietly(t *testing.T) {
	runner := &syncTestRunner{}
	if err := (&Bitwarden{Runner: runner}).Sync(); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}
	if runner.name != "bw" || len(runner.args) != 1 || runner.args[0] != "sync" {
		t.Fatalf("unexpected provider command: %s %v", runner.name, runner.args)
	}
	if runner.streams.Stdout != io.Discard || runner.streams.Stderr != io.Discard {
		t.Fatal("sync output must not be written to the terminal")
	}
}

func TestBitwardenSyncReturnsCommandError(t *testing.T) {
	wantErr := errors.New("command failed")
	err := (&Bitwarden{Runner: &syncTestRunner{err: wantErr}}).Sync()
	if !errors.Is(err, wantErr) {
		t.Fatalf("Sync() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestBitwardenFolderJSON(t *testing.T) {
	data := `[{"id":"f1","name":"Dev"},{"id":"f2","name":"Production"}]`
	var folders []bwFolder
	if err := json.Unmarshal([]byte(data), &folders); err != nil {
		t.Fatalf("failed to parse folder JSON: %v", err)
	}
	if len(folders) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(folders))
	}
	if folders[0].ID != "f1" || folders[0].Name != "Dev" {
		t.Errorf("expected folder[0] = {f1 Dev}, got {%s %s}", folders[0].ID, folders[0].Name)
	}
}

func TestBitwardenItemJSON(t *testing.T) {
	data := `[{"id":"item1","name":"API Keys","fields":[{"name":"API_KEY","value":"sk-123","type":1},{"name":"API_SECRET","value":"ss-456","type":0}]}]`
	var items []bwItem
	if err := json.Unmarshal([]byte(data), &items); err != nil {
		t.Fatalf("failed to parse item JSON: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.ID != "item1" || item.Name != "API Keys" {
		t.Errorf("expected item = {item1 API Keys}, got {%s %s}", item.ID, item.Name)
	}
	if len(item.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(item.Fields))
	}

	fields := item.Fields
	if fields[0].Name != "API_KEY" || fields[0].Value != "sk-123" || fields[0].Type != 1 {
		t.Errorf("unexpected field[0]: %+v", fields[0])
	}
}

func TestBitwardenItemsToSecrets(t *testing.T) {
	items := []bwItem{
		{
			ID: "item1", Name: "API Keys",
			Fields: []bwField{
				{Name: "API_KEY", Value: "sk-123", Type: 1},
				{Name: "API_SECRET", Value: "ss-456", Type: 0},
			},
		},
		{
			ID: "item2", Name: "Database",
			Fields: []bwField{
				{Name: "DB_HOST", Value: "localhost", Type: 0},
				{Name: "DB_PORT", Value: "5432", Type: 0},
			},
		},
	}

	var secrets []Secret
	for _, item := range items {
		for _, field := range item.Fields {
			if field.Name == "" {
				continue
			}
			secrets = append(secrets, Secret{Key: field.Name, Value: field.Value})
		}
	}

	if len(secrets) != 4 {
		t.Fatalf("expected 4 secrets, got %d", len(secrets))
	}

	secretMap := make(map[string]string)
	for _, s := range secrets {
		secretMap[s.Key] = s.Value
	}

	if secretMap["API_KEY"] != "sk-123" {
		t.Errorf("expected API_KEY = sk-123, got %q", secretMap["API_KEY"])
	}
	if secretMap["DB_PORT"] != "5432" {
		t.Errorf("expected DB_PORT = 5432, got %q", secretMap["DB_PORT"])
	}
}

func TestBitwardenSkipsEmptyFieldNames(t *testing.T) {
	items := []bwItem{
		{
			ID: "item1", Name: "Test",
			Fields: []bwField{
				{Name: "", Value: "should-skip", Type: 0},
				{Name: "VALID_KEY", Value: "valid-value", Type: 1},
			},
		},
	}

	var secrets []Secret
	for _, item := range items {
		for _, field := range item.Fields {
			if field.Name == "" {
				continue
			}
			secrets = append(secrets, Secret{Key: field.Name, Value: field.Value})
		}
	}

	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret (empty name skipped), got %d", len(secrets))
	}
	if secrets[0].Key != "VALID_KEY" {
		t.Errorf("expected VALID_KEY, got %q", secrets[0].Key)
	}
}

func TestBitwardenFolderToFolderStruct(t *testing.T) {
	raw := []bwFolder{
		{ID: "f1", Name: "Dev"},
		{ID: "f2", Name: "Production"},
		{ID: "f3", Name: ""}, // Should be skipped
	}

	folders := make([]Folder, 0, len(raw))
	for _, f := range raw {
		if f.Name == "" {
			continue
		}
		folders = append(folders, Folder(f))
	}

	if len(folders) != 2 {
		t.Fatalf("expected 2 folders (empty name skipped), got %d", len(folders))
	}
	if folders[1].Name != "Production" {
		t.Errorf("expected folder[1] = Production, got %q", folders[1].Name)
	}
}

func TestBitwardenItemsToSecretItems(t *testing.T) {
	raw := []bwItem{
		{ID: "id-1", Name: "API Keys"},
		{ID: "id-2", Name: "Database"},
	}

	items := make([]SecretItem, 0, len(raw))
	for _, item := range raw {
		items = append(items, SecretItem{ID: item.ID, Name: item.Name})
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[1].Name != "Database" {
		t.Errorf("expected items[1] = Database, got %q", items[1].Name)
	}
}
