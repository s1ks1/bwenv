package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOnePasswordVaultJSON(t *testing.T) {
	data := `[{"id":"v1","name":"Personal"},{"id":"v2","name":"Team Vault"}]`
	var vaults []opVault
	if err := json.Unmarshal([]byte(data), &vaults); err != nil {
		t.Fatalf("failed to parse vault JSON: %v", err)
	}
	if len(vaults) != 2 {
		t.Fatalf("expected 2 vaults, got %d", len(vaults))
	}
	if vaults[0].ID != "v1" || vaults[0].Name != "Personal" {
		t.Errorf("expected vault[0] = {v1 Personal}, got {%s %s}", vaults[0].ID, vaults[0].Name)
	}
}

func TestOnePasswordItemJSON(t *testing.T) {
	data := `[{"id":"item1","title":"API Keys"},{"id":"item2","title":"DB Creds"}]`
	var items []opItem
	if err := json.Unmarshal([]byte(data), &items); err != nil {
		t.Fatalf("failed to parse item JSON: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Title != "API Keys" {
		t.Errorf("expected item[0].Title = 'API Keys', got %q", items[0].Title)
	}
}

func TestOnePasswordItemDetailJSON(t *testing.T) {
	data := `{
		"id":"item1",
		"title":"API Keys",
		"fields":[
			{"id":"f1","label":"API_KEY","value":"sk-123","type":"CONCEALED","purpose":""},
			{"id":"f2","label":"API_SECRET","value":"ss-456","type":"CONCEALED","purpose":""}
		]
	}`
	var detail opItemDetail
	if err := json.Unmarshal([]byte(data), &detail); err != nil {
		t.Fatalf("failed to parse item detail JSON: %v", err)
	}
	if detail.Title != "API Keys" {
		t.Errorf("expected title 'API Keys', got %q", detail.Title)
	}
	if len(detail.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(detail.Fields))
	}
}

func TestOnePasswordVaultsToFolders(t *testing.T) {
	raw := []opVault{
		{ID: "v1", Name: "Personal"},
		{ID: "v2", Name: "Work"},
	}

	folders := make([]Folder, 0, len(raw))
	for _, v := range raw {
		folders = append(folders, Folder{ID: v.ID, Name: v.Name})
	}

	if len(folders) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(folders))
	}
	if folders[1].Name != "Work" {
		t.Errorf("expected folder[1] = Work, got %q", folders[1].Name)
	}
}

func TestOnePasswordFieldToSecret(t *testing.T) {
	detail := opItemDetail{
		ID:    "item1",
		Title: "API Keys",
		Fields: []opItemField{
			{ID: "f1", Label: "API_KEY", Value: "sk-123", Type: "CONCEALED", Purpose: ""},
			{ID: "f2", Label: "API_SECRET", Value: "ss-456", Type: "CONCEALED", Purpose: ""},
		},
	}

	var secrets []Secret
	for _, field := range detail.Fields {
		if field.Label == "" {
			continue
		}
		if strings.EqualFold(field.Purpose, "NOTES") {
			continue
		}
		if strings.EqualFold(field.Type, "OTP") {
			continue
		}
		if field.Value == "" {
			continue
		}
		secrets = append(secrets, Secret{Key: field.Label, Value: field.Value})
	}

	if len(secrets) != 2 {
		t.Fatalf("expected 2 secrets, got %d", len(secrets))
	}
	if secrets[0].Key != "API_KEY" || secrets[0].Value != "sk-123" {
		t.Errorf("expected {API_KEY sk-123}, got {%s %s}", secrets[0].Key, secrets[0].Value)
	}
}

func TestOnePasswordSkipsNotesField(t *testing.T) {
	detail := opItemDetail{
		ID:    "item1",
		Title: "Test",
		Fields: []opItemField{
			{ID: "f1", Label: "Notes", Value: "some notes", Type: "STRING", Purpose: "NOTES"},
			{ID: "f2", Label: "API_KEY", Value: "sk-123", Type: "CONCEALED", Purpose: ""},
		},
	}

	var secrets []Secret
	for _, field := range detail.Fields {
		if field.Label == "" {
			continue
		}
		if strings.EqualFold(field.Purpose, "NOTES") {
			continue
		}
		if strings.EqualFold(field.Type, "OTP") {
			continue
		}
		if field.Value == "" {
			continue
		}
		secrets = append(secrets, Secret{Key: field.Label, Value: field.Value})
	}

	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret (NOTES skipped), got %d", len(secrets))
	}
	if secrets[0].Key != "API_KEY" {
		t.Errorf("expected API_KEY, got %q", secrets[0].Key)
	}
}

func TestOnePasswordSkipsOTPField(t *testing.T) {
	detail := opItemDetail{
		ID:    "item1",
		Title: "Test",
		Fields: []opItemField{
			{ID: "f1", Label: "TOTP", Value: "otp-secret", Type: "OTP", Purpose: ""},
			{ID: "f2", Label: "VALID_KEY", Value: "valid", Type: "STRING", Purpose: ""},
		},
	}

	var secrets []Secret
	for _, field := range detail.Fields {
		if field.Label == "" {
			continue
		}
		if strings.EqualFold(field.Purpose, "NOTES") {
			continue
		}
		if strings.EqualFold(field.Type, "OTP") {
			continue
		}
		if field.Value == "" {
			continue
		}
		secrets = append(secrets, Secret{Key: field.Label, Value: field.Value})
	}

	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret (OTP skipped), got %d", len(secrets))
	}
}

func TestOnePasswordSkipsEmptyValue(t *testing.T) {
	detail := opItemDetail{
		ID:    "item1",
		Title: "Test",
		Fields: []opItemField{
			{ID: "f1", Label: "EMPTY_FIELD", Value: "", Type: "STRING", Purpose: ""},
			{ID: "f2", Label: "GOOD_KEY", Value: "good-value", Type: "STRING", Purpose: ""},
		},
	}

	var secrets []Secret
	for _, field := range detail.Fields {
		if field.Label == "" {
			continue
		}
		if strings.EqualFold(field.Purpose, "NOTES") {
			continue
		}
		if strings.EqualFold(field.Type, "OTP") {
			continue
		}
		if field.Value == "" {
			continue
		}
		secrets = append(secrets, Secret{Key: field.Label, Value: field.Value})
	}

	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret (empty value skipped), got %d", len(secrets))
	}
}

func TestOnePasswordSkipsEmptyLabel(t *testing.T) {
	detail := opItemDetail{
		ID:    "item1",
		Title: "Test",
		Fields: []opItemField{
			{ID: "f1", Label: "", Value: "no-label", Type: "STRING", Purpose: ""},
			{ID: "f2", Label: "GOOD_KEY", Value: "good-value", Type: "STRING", Purpose: ""},
		},
	}

	var secrets []Secret
	for _, field := range detail.Fields {
		if field.Label == "" {
			continue
		}
		if strings.EqualFold(field.Purpose, "NOTES") {
			continue
		}
		if strings.EqualFold(field.Type, "OTP") {
			continue
		}
		if field.Value == "" {
			continue
		}
		secrets = append(secrets, Secret{Key: field.Label, Value: field.Value})
	}

	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret (empty label skipped), got %d", len(secrets))
	}
}

func TestOnePasswordItemsToSecretItems(t *testing.T) {
	raw := []opItem{
		{ID: "id-1", Title: "API Keys"},
		{ID: "id-2", Title: "Database"},
	}

	items := make([]SecretItem, 0, len(raw))
	for _, item := range raw {
		items = append(items, SecretItem{ID: item.ID, Name: item.Title})
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Name != "API Keys" {
		t.Errorf("expected items[0] = 'API Keys', got %q", items[0].Name)
	}
}
