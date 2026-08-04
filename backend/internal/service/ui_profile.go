package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
)

const maxUserUIProfileBytes = 16 << 10

var userUIProfilePackageIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

var userUIProfileAllowedFields = map[string]struct{}{
	"mode":            {},
	"layout":          {},
	"baseColor":       {},
	"chartColor":      {},
	"heading":         {},
	"font":            {},
	"iconLibrary":     {},
	"radius":          {},
	"density":         {},
	"menu":            {},
	"menuSize":        {},
	"motion":          {},
	"tableStyle":      {},
	"activePackageId": {},
}

var userUIProfileReservedFields = map[string]struct{}{
	"style":           {},
	"uiStyle":         {},
	"ui_style":        {},
	"ui-style":        {},
	"componentStyle":  {},
	"component_style": {},
	"component-style": {},
}

// UserUIProfile stores user-level overrides. The built-in component style is
// intentionally absent because LightBridge fixes the trusted modern style to Luma.
type UserUIProfile struct {
	Mode            string `json:"mode,omitempty"`
	Layout          string `json:"layout,omitempty"`
	BaseColor       string `json:"baseColor,omitempty"`
	ChartColor      string `json:"chartColor,omitempty"`
	Heading         string `json:"heading,omitempty"`
	Font            string `json:"font,omitempty"`
	IconLibrary     string `json:"iconLibrary,omitempty"`
	Radius          string `json:"radius,omitempty"`
	Density         string `json:"density,omitempty"`
	Menu            string `json:"menu,omitempty"`
	MenuSize        string `json:"menuSize,omitempty"`
	Motion          string `json:"motion,omitempty"`
	TableStyle      string `json:"tableStyle,omitempty"`
	ActivePackageID string `json:"activePackageId,omitempty"`
}

func userUIProfileSettingKey(userID int64) string {
	return "user_ui_profile:" + strconv.FormatInt(userID, 10)
}

func (s *UserService) GetUserUIProfile(ctx context.Context, userID int64) (UserUIProfile, error) {
	if s == nil || s.settingRepo == nil {
		return UserUIProfile{}, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, userUIProfileSettingKey(userID))
	if err != nil {
		if err == ErrSettingNotFound {
			return UserUIProfile{}, nil
		}
		return UserUIProfile{}, fmt.Errorf("get user UI profile: %w", err)
	}
	profile, err := ParseUserUIProfile(json.RawMessage(raw))
	if err != nil {
		return UserUIProfile{}, fmt.Errorf("parse user UI profile: %w", err)
	}
	return profile, nil
}

func (s *UserService) UpdateUserUIProfile(ctx context.Context, userID int64, raw json.RawMessage) (UserUIProfile, error) {
	if s == nil || s.settingRepo == nil {
		return UserUIProfile{}, infraerrors.New(500, "UI_PROFILE_STORAGE_UNAVAILABLE", "UI profile storage is unavailable")
	}
	profile, err := ParseUserUIProfile(raw)
	if err != nil {
		return UserUIProfile{}, err
	}
	encoded, err := json.Marshal(profile)
	if err != nil {
		return UserUIProfile{}, infraerrors.BadRequest("UI_PROFILE_INVALID", "UI profile could not be encoded")
	}
	if err := s.settingRepo.Set(ctx, userUIProfileSettingKey(userID), string(encoded)); err != nil {
		return UserUIProfile{}, fmt.Errorf("save user UI profile: %w", err)
	}
	return profile, nil
}

func (s *UserService) ResetUserUIProfile(ctx context.Context, userID int64) error {
	if s == nil || s.settingRepo == nil {
		return nil
	}
	if err := s.settingRepo.Delete(ctx, userUIProfileSettingKey(userID)); err != nil && err != ErrSettingNotFound {
		return fmt.Errorf("reset user UI profile: %w", err)
	}
	return nil
}

func ParseUserUIProfile(raw json.RawMessage) (UserUIProfile, error) {
	if len(raw) == 0 {
		return UserUIProfile{}, nil
	}
	if len(raw) > maxUserUIProfileBytes {
		return UserUIProfile{}, infraerrors.BadRequest("UI_PROFILE_TOO_LARGE", "UI profile is too large")
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return UserUIProfile{}, infraerrors.BadRequest("UI_PROFILE_INVALID", "UI profile must be a JSON object")
	}
	for key := range object {
		if _, reserved := userUIProfileReservedFields[key]; reserved {
			return UserUIProfile{}, infraerrors.BadRequest("UI_PROFILE_STYLE_FIXED", "component style is fixed to Luma")
		}
		if _, allowed := userUIProfileAllowedFields[key]; !allowed {
			return UserUIProfile{}, infraerrors.BadRequest("UI_PROFILE_UNKNOWN_FIELD", "UI profile contains an unknown field: "+key)
		}
	}

	var profile UserUIProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		return UserUIProfile{}, infraerrors.BadRequest("UI_PROFILE_INVALID", "UI profile contains invalid values")
	}
	if err := validateUserUIProfile(profile); err != nil {
		return UserUIProfile{}, err
	}
	return profile, nil
}

func validateUserUIProfile(profile UserUIProfile) error {
	if err := validateUIProfileChoice("mode", profile.Mode, "legacy", "modern", "package"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("layout", profile.Layout, "default", "compact", "spacious"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("baseColor", profile.BaseColor, "natural", "neutral", "stone", "zinc"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("chartColor", profile.ChartColor, "natural", "vivid", "muted"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("heading", profile.Heading, "natural", "compact", "editorial", "strong"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("font", profile.Font, "inter", "system"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("iconLibrary", profile.IconLibrary, "lucide"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("radius", profile.Radius, "none", "small", "default", "large"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("density", profile.Density, "compact", "default", "comfortable"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("menu", profile.Menu, "default", "subtle", "outlined", "translucent"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("menuSize", profile.MenuSize, "compact", "suitable", "comfortable", "wide"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("motion", profile.Motion, "reduced", "default", "expressive"); err != nil {
		return err
	}
	if err := validateUIProfileChoice("tableStyle", profile.TableStyle, "compact", "default", "comfortable"); err != nil {
		return err
	}

	if profile.ActivePackageID != "" && !userUIProfilePackageIDPattern.MatchString(profile.ActivePackageID) {
		return infraerrors.BadRequest("UI_PROFILE_INVALID_PACKAGE", "activePackageId is invalid")
	}
	if profile.Mode == "package" && profile.ActivePackageID == "" {
		return infraerrors.BadRequest("UI_PROFILE_PACKAGE_REQUIRED", "package mode requires activePackageId")
	}
	return nil
}

func validateUIProfileChoice(field, value string, allowed ...string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, option := range allowed {
		if value == option {
			return nil
		}
	}
	return infraerrors.BadRequest("UI_PROFILE_INVALID_VALUE", field+" is not supported")
}
