package account

import (
	"testing"

	"github.com/google/uuid"
)

func TestParseDeviceProfile(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantApp string
	}{
		{
			name:    "complete profile",
			input:   `{"model":"iPhone 14","system":"iOS 16","app":"10.0","lang_pack":"ios"}`,
			wantApp: "10.0",
		},
		{name: "empty object", input: `{}`, wantErr: true},
		{name: "missing model", input: `{"system":"iOS 16","app":"10.0"}`, wantErr: true},
		{name: "missing system", input: `{"model":"iPhone 14","app":"10.0"}`, wantErr: true},
		{name: "missing app", input: `{"model":"iPhone 14","system":"iOS 16"}`, wantErr: true},
		{name: "wrong field type", input: `{"model":123,"system":"iOS 16","app":"10.0"}`, wantErr: true},
		{name: "invalid JSON", input: `{"model":`, wantErr: true},
		{
			name:    "legacy metadata is ignored",
			input:   `{"model":"MacBook Pro","system":"macOS 14","app":"7.0.9","device_type":"macos","twofa_enc":"ciphertext"}`,
			wantApp: "7.0.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDeviceProfile([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				if got != nil {
					t.Fatalf("expected nil profile on error, got %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDeviceProfile() error = %v", err)
			}
			if got == nil {
				t.Fatal("expected profile")
			}
			if got.App != tt.wantApp {
				t.Errorf("App = %q, want %q", got.App, tt.wantApp)
			}
		})
	}
}

func TestDeviceProfileFromMapRejectsWrongTypes(t *testing.T) {
	_, err := deviceProfileFromMap(map[string]any{
		"model": 123, "system": "macOS 14", "app": "7.0.9",
	})
	if err == nil {
		t.Fatal("expected type mismatch to be rejected")
	}
}

func TestDeviceProfileFromMapAcceptsLegacyMetadata(t *testing.T) {
	got, err := deviceProfileFromMap(map[string]any{
		"model": "MacBook Pro", "system": "macOS 14", "app": "7.0.9",
		"device_type": "macos", "twofa_enc": "ciphertext",
	})
	if err != nil {
		t.Fatalf("deviceProfileFromMap() error = %v", err)
	}
	if got.DeviceType != "macos" || got.App != "7.0.9" {
		t.Fatalf("unexpected parsed profile: %#v", got)
	}
}


func TestPickDeviceTypeUsesIOS(t *testing.T) {
	if got := pickDeviceType(uuid.Nil); got != "ios" {
		t.Fatalf("pickDeviceType() = %q, want ios", got)
	}
}

func TestDeviceProfileToConfigMigratesLegacyMacProfileToIOS(t *testing.T) {
	got := deviceProfileToConfig(map[string]any{
		"model": "MacBook Pro", "system": "macOS 14", "app": "7.0.9",
		"device_type": "macos", "platform": "macos",
	}, "account-session")
	want := generateDeviceConfig("account-session")
	if got != want {
		t.Fatalf("legacy macOS profile was not migrated to iOS: got %#v, want %#v", got, want)
	}
}

func TestMergeDeviceConfigProfilePreservesMetadata(t *testing.T) {
	existing := map[string]any{
		"model": "MacBook Pro", "system": "macOS 14", "app": "7.0.9",
		"device_type": "macos", "twofa_enc": "encrypted-secret",
	}
	got := mergeDeviceConfigProfile(existing, generateDeviceConfig("account-session"), "ios")
	if got["device_type"] != "ios" {
		t.Fatalf("device_type = %v, want ios", got["device_type"])
	}
	if got["twofa_enc"] != "encrypted-secret" {
		t.Fatalf("unrelated metadata was lost: %#v", got)
	}
}

func TestMetadataOnlyProtocolProfileCanBeInitialized(t *testing.T) {
	profile := map[string]any{"twofa_enc": "encrypted-secret"}
	if hasDeviceIdentityFields(profile) {
		t.Fatal("metadata-only profile must not be treated as a partial device identity")
	}
	got := mergeDeviceConfigProfile(profile, generateDeviceConfig("account-session"), "ios")
	if _, err := deviceProfileFromMap(got); err != nil {
		t.Fatalf("initialized profile is invalid: %v", err)
	}
	if got["twofa_enc"] != "encrypted-secret" {
		t.Fatal("2FA metadata was not preserved")
	}
}
