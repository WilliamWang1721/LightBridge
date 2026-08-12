package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
)

const runtimeAppearanceVersion = 1

var presetCodePattern = regexp.MustCompile(`^[ab][A-Za-z0-9]{1,255}$`)

var ErrInvalidRuntimeAppearancePreset = infraerrors.BadRequest(
	"UI_APPEARANCE_PRESET_INVALID",
	"invalid shadcn preset code",
)

// RuntimeAppearanceSettings is deliberately code-only at the storage boundary.
// The browser uses the official shadcn/preset decoder to resolve the code into
// safe, project-owned UI tokens; the backend never accepts CSS, HTML, or JS.
type RuntimeAppearanceSettings struct {
	Version    int    `json:"version"`
	Source     string `json:"source"`
	PresetCode string `json:"preset_code"`
}

func ValidateRuntimeAppearancePresetCode(raw string) (string, error) {
	code := strings.TrimSpace(raw)
	if !presetCodePattern.MatchString(code) {
		return "", ErrInvalidRuntimeAppearancePreset
	}
	return code, nil
}

func ParseRuntimeAppearance(raw string) (*RuntimeAppearanceSettings, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var stored RuntimeAppearanceSettings
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return nil, fmt.Errorf("decode runtime appearance: %w", err)
	}
	if stored.Version != runtimeAppearanceVersion || stored.Source != "shadcn-preset" {
		return nil, ErrInvalidRuntimeAppearancePreset
	}
	code, err := ValidateRuntimeAppearancePresetCode(stored.PresetCode)
	if err != nil {
		return nil, err
	}
	stored.PresetCode = code
	return &stored, nil
}

func (s *SettingService) GetRuntimeAppearance(ctx context.Context) (*RuntimeAppearanceSettings, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUIAppearance)
	if errors.Is(err, ErrSettingNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get runtime appearance: %w", err)
	}
	appearance, err := ParseRuntimeAppearance(raw)
	if err != nil {
		return nil, fmt.Errorf("parse runtime appearance: %w", err)
	}
	return appearance, nil
}

func (s *SettingService) SetRuntimeAppearance(ctx context.Context, presetCode string) (*RuntimeAppearanceSettings, error) {
	code, err := ValidateRuntimeAppearancePresetCode(presetCode)
	if err != nil {
		return nil, err
	}
	appearance := &RuntimeAppearanceSettings{
		Version:    runtimeAppearanceVersion,
		Source:     "shadcn-preset",
		PresetCode: code,
	}
	raw, err := json.Marshal(appearance)
	if err != nil {
		return nil, fmt.Errorf("encode runtime appearance: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUIAppearance, string(raw)); err != nil {
		return nil, fmt.Errorf("save runtime appearance: %w", err)
	}
	return appearance, nil
}

func (s *SettingService) ResetRuntimeAppearance(ctx context.Context) error {
	if err := s.settingRepo.Delete(ctx, SettingKeyUIAppearance); err != nil && !errors.Is(err, ErrSettingNotFound) {
		return fmt.Errorf("reset runtime appearance: %w", err)
	}
	return nil
}
