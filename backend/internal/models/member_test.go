package models

import "testing"

func TestMember_TableName(t *testing.T) {
	m := Member{}
	if m.TableName() != "members" {
		t.Errorf("expected 'members', got '%s'", m.TableName())
	}
}

func TestRole_TableName(t *testing.T) {
	r := Role{}
	if r.TableName() != "roles" {
		t.Errorf("expected 'roles', got '%s'", r.TableName())
	}
}

func TestAPIKey_TableName(t *testing.T) {
	a := APIKey{}
	if a.TableName() != "api_keys" {
		t.Errorf("expected 'api_keys', got '%s'", a.TableName())
	}
}

func TestExternalPlatform_TableName(t *testing.T) {
	e := ExternalPlatform{}
	if e.TableName() != "external_platforms" {
		t.Errorf("expected 'external_platforms', got '%s'", e.TableName())
	}
}
