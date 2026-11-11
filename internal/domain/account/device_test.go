package account

import "testing"

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
