package provider

import (
	"encoding/json"
	"testing"
)

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
		folders = append(folders, Folder{ID: f.ID, Name: f.Name})
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
