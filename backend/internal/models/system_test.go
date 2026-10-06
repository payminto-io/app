package models

import "testing"

func TestConfiguration_TableName(t *testing.T) {
	c := Configuration{}
	if c.TableName() != "configurations" {
		t.Errorf("expected 'configurations', got %q", c.TableName())
	}
}

func TestConfigModeKey_Constant(t *testing.T) {
	if ConfigModeKey != "mode" {
		t.Errorf("expected ConfigModeKey = 'mode', got %q", ConfigModeKey)
	}
}
