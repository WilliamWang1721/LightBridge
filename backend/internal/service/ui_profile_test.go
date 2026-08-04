package service

import (
	"context"
	"encoding/json"
	"testing"
)

type uiProfileSettingRepo struct {
	values map[string]string
}

func newUIProfileSettingRepo() *uiProfileSettingRepo {
	return &uiProfileSettingRepo{values: map[string]string{}}
}

func (r *uiProfileSettingRepo) Get(_ context.Context, key string) (*Setting, error) {
	value, ok := r.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: value}, nil
}

func (r *uiProfileSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (r *uiProfileSettingRepo) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}

func (r *uiProfileSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (r *uiProfileSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *uiProfileSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	out := make(map[string]string, len(r.values))
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *uiProfileSettingRepo) Delete(_ context.Context, key string) error {
	if _, ok := r.values[key]; !ok {
		return ErrSettingNotFound
	}
	delete(r.values, key)
	return nil
}

func TestParseUserUIProfileAcceptsIndependentAxes(t *testing.T) {
	profile, err := ParseUserUIProfile(json.RawMessage(`{
		"mode":"modern",
		"layout":"spacious",
		"baseColor":"natural",
		"chartColor":"vivid",
		"font":"inter",
		"iconLibrary":"lucide",
		"menuSize":"suitable"
	}`))
	if err != nil {
		t.Fatalf("ParseUserUIProfile returned error: %v", err)
	}
	if profile.Mode != "modern" || profile.Layout != "spacious" || profile.ChartColor != "vivid" {
		t.Fatalf("unexpected profile: %#v", profile)
	}
}

func TestParseUserUIProfileRejectsStyleOverride(t *testing.T) {
	if _, err := ParseUserUIProfile(json.RawMessage(`{"componentStyle":"other"}`)); err == nil {
		t.Fatal("expected component style override to be rejected")
	}
}

func TestParseUserUIProfileRejectsUnknownValue(t *testing.T) {
	if _, err := ParseUserUIProfile(json.RawMessage(`{"baseColor":"rainbow"}`)); err == nil {
		t.Fatal("expected unsupported base color to be rejected")
	}
}

func TestUserUIProfilePersistsAndResets(t *testing.T) {
	repo := newUIProfileSettingRepo()
	service := NewUserService(nil, repo, nil, nil)
	ctx := context.Background()

	saved, err := service.UpdateUserUIProfile(ctx, 42, json.RawMessage(`{"mode":"modern","radius":"large"}`))
	if err != nil {
		t.Fatalf("UpdateUserUIProfile returned error: %v", err)
	}
	if saved.Mode != "modern" || saved.Radius != "large" {
		t.Fatalf("unexpected saved profile: %#v", saved)
	}

	loaded, err := service.GetUserUIProfile(ctx, 42)
	if err != nil {
		t.Fatalf("GetUserUIProfile returned error: %v", err)
	}
	if loaded != saved {
		t.Fatalf("loaded profile %#v does not match saved %#v", loaded, saved)
	}

	if err := service.ResetUserUIProfile(ctx, 42); err != nil {
		t.Fatalf("ResetUserUIProfile returned error: %v", err)
	}
	loaded, err = service.GetUserUIProfile(ctx, 42)
	if err != nil {
		t.Fatalf("GetUserUIProfile after reset returned error: %v", err)
	}
	if loaded != (UserUIProfile{}) {
		t.Fatalf("expected empty profile after reset, got %#v", loaded)
	}
}
