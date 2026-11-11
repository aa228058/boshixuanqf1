package account

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	device "github.com/mtgo-labs/device-manager"
	"github.com/mtgo-labs/mtgo/telegram"
)

// DeviceProfile is the typed, backward-compatible view of accounts.device_profile.
// Unknown JSON fields are intentionally ignored because older rows also contain
// account metadata (for example device_type and encrypted 2FA metadata).
type DeviceProfile struct {
	Model          string `json:"model"`
	System         string `json:"system"`
	App            string `json:"app"`
	LangPack       string `json:"lang_pack"`
	LangCode       string `json:"lang_code"`
	SystemLangCode string `json:"system_lang_code"`
	Platform       string `json:"platform,omitempty"`
	DeviceType     string `json:"device_type,omitempty"`
}

// ParseDeviceProfile validates the device fields required to restore a saved
// client identity. It does not mutate the original JSONB map.
func ParseDeviceProfile(data []byte) (*DeviceProfile, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("device profile is empty")
	}

	var profile DeviceProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("decode device profile: %w", err)
	}
	if strings.TrimSpace(profile.Model) == "" ||
		strings.TrimSpace(profile.System) == "" ||
		strings.TrimSpace(profile.App) == "" {
		return nil, fmt.Errorf("invalid device profile: model, system and app are required")
	}
	return &profile, nil
}

// deviceProfileFromMap converts the repository's JSONB map representation to
// the typed profile while retaining compatibility with unrelated legacy keys.
func deviceProfileFromMap(profile map[string]any) (*DeviceProfile, error) {
	if len(profile) == 0 {
		return nil, fmt.Errorf("device profile is empty")
	}
	data, err := json.Marshal(profile)
	if err != nil {
		return nil, fmt.Errorf("encode device profile: %w", err)
	}
	return ParseDeviceProfile(data)
}

func generateDeviceConfig(sessionName string) telegram.DeviceConfig {
	cfg := &telegram.Config{}
	device.IOS.Generate(sessionName).Apply(cfg)
	return cfg.Device
}

// pickDeviceType keeps the account device identity on the iOS client profile.
func pickDeviceType(_ uuid.UUID) string {
	return "ios"
}

// deviceProfileToConfig converts a saved device profile map to telegram.DeviceConfig.
// Reads device_type from profile, falls back to deterministic selection from sessionName.

func deviceProfileToConfig(profile map[string]any, sessionName string) telegram.DeviceConfig {
	if sessionName == "" {
		return telegram.DeviceConfig{}
	}

	model, _ := profile["model"].(string)
	system, _ := profile["system"].(string)
	deviceType, _ := profile["device_type"].(string)
	platform, _ := profile["platform"].(string)
	legacyType := (deviceType != "" && !strings.EqualFold(deviceType, "ios")) ||
		(platform != "" && !strings.EqualFold(platform, "ios")) ||
		strings.Contains(strings.ToLower(model), "mac") ||
		strings.Contains(strings.ToLower(system), "macos")

	// Old macOS/Desktop profiles are migrated to an iOS profile on next connect.
	// Incomplete profiles also receive a complete iOS profile.
	if legacyType {
		return generateDeviceConfig(sessionName)
	}

	cfg := telegram.DeviceConfig{}
	if model != "" {
		cfg.DeviceModel = model
	}
	if system != "" {
		cfg.SystemVersion = system
	}
	if app, ok := profile["app"].(string); ok && app != "" {
		cfg.AppVersion = app
	}
	if langPack, ok := profile["lang_pack"].(string); ok && langPack != "" {
		cfg.LangPack = langPack
	}
	if langCode, ok := profile["lang_code"].(string); ok && langCode != "" {
		cfg.LangCode = langCode
	}
	if sysLangCode, ok := profile["system_lang_code"].(string); ok && sysLangCode != "" {
		cfg.SystemLangCode = sysLangCode
	}
	if cfg.DeviceModel == "" || cfg.SystemVersion == "" || cfg.AppVersion == "" {
		return generateDeviceConfig(sessionName)
	}
	return cfg
}

// hasDeviceIdentityFields reports whether a profile contains any device identity
// keys. Metadata-only profiles (for example encrypted 2FA) can be safely
// initialized for legacy imported sessions.
func hasDeviceIdentityFields(profile map[string]any) bool {
	for _, key := range []string{"model", "system", "app"} {
		if _, ok := profile[key]; ok {
			return true
		}
	}
	return false
}

// mergeDeviceConfigProfile replaces device identity fields while preserving
// unrelated account metadata stored in the same JSONB object.
func mergeDeviceConfigProfile(existing map[string]any, d telegram.DeviceConfig, deviceType string) map[string]any {
	result := make(map[string]any, len(existing)+8)
	for key, value := range existing {
		switch key {
		case "model", "system", "app", "lang_pack", "lang_code", "system_lang_code", "platform", "device_type":
			continue
		default:
			result[key] = value
		}
	}
	for key, value := range deviceConfigToProfile(d, deviceType) {
		result[key] = value
	}
	return result
}

// deviceConfigToProfile converts telegram.DeviceConfig to a map for DB storage.

func deviceConfigToProfile(d telegram.DeviceConfig, deviceType string) map[string]any {
	return map[string]any{
		"model":            d.DeviceModel,
		"system":           d.SystemVersion,
		"app":              d.AppVersion,
		"lang_pack":        d.LangPack,
		"lang_code":        d.LangCode,
		"system_lang_code": d.SystemLangCode,
		"platform":         deviceType,
		"device_type":      deviceType,
	}
}

// StartSessionBackup starts a background goroutine to periodically backup session files.
