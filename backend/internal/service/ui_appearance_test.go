package service

import "testing"

func TestValidateRuntimeAppearancePresetCode(t *testing.T) {
	for _, test := range []struct {
		name string
		code string
		ok   bool
	}{
		{name: "official default", code: " b0 ", ok: true},
		{name: "version a", code: "aZ9", ok: true},
		{name: "empty", code: "", ok: false},
		{name: "wrong version", code: "c0", ok: false},
		{name: "unsafe characters", code: "b0;alert(1)", ok: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateRuntimeAppearancePresetCode(test.code)
			if (err == nil) != test.ok {
				t.Fatalf("ValidateRuntimeAppearancePresetCode(%q) error = %v, want valid = %v", test.code, err, test.ok)
			}
		})
	}
}
