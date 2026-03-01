package main

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestServer(t *testing.T) *server {
	t.Helper()
	return &server{
		storePath:    filepath.Join(t.TempDir(), "totp_devices.json"),
		store:        totpStore{Users: map[string][]totpDevice{}},
		enrollStates: map[string]pendingEnroll{},
		stateTTL:     20 * time.Millisecond,
		stateMaxSkew: 0,
	}
}

func TestVerifyForUserPreventsReplay(t *testing.T) {
	s := newTestServer(t)
	secret := "JBSWY3DPEHPK3PXP"
	username := "admin"
	now := time.Unix(1700000000, 0).UTC()
	code := testTOTPCode(t, secret, now)

	if err := s.addDevice(username, totpDevice{
		DeviceID:  "dev-1",
		Label:     "phone",
		Secret:    secret,
		CreatedAt: now.Add(-time.Minute).Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("add device: %v", err)
	}

	dev, _, ok, err := s.verifyForUser(username, code, now)
	if err != nil || !ok {
		t.Fatalf("first verify failed: ok=%v err=%v", ok, err)
	}
	if dev.DeviceID != "dev-1" {
		t.Fatalf("unexpected device: %+v", dev)
	}

	_, _, ok, err = s.verifyForUser(username, code, now)
	if err != nil {
		t.Fatalf("second verify error: %v", err)
	}
	if ok {
		t.Fatal("expected replayed code to be rejected")
	}
}

func TestDeleteDeviceRemovesUserBucket(t *testing.T) {
	s := newTestServer(t)
	if err := s.addDevice("admin", totpDevice{
		DeviceID: "dev-1",
		Label:    "phone",
		Secret:   "JBSWY3DPEHPK3PXP",
	}); err != nil {
		t.Fatalf("add device: %v", err)
	}

	deleted, err := s.deleteDevice("admin", "dev-1")
	if err != nil {
		t.Fatalf("delete device: %v", err)
	}
	if !deleted {
		t.Fatal("expected device to be deleted")
	}
	if _, ok := s.store.Users["admin"]; ok {
		t.Fatal("expected empty user bucket to be removed")
	}
}

func TestEnrollStateExpires(t *testing.T) {
	s := newTestServer(t)
	token, err := s.newEnrollState(pendingEnroll{
		Username: "admin",
		Secret:   "JBSWY3DPEHPK3PXP",
	})
	if err != nil {
		t.Fatalf("newEnrollState: %v", err)
	}
	if _, ok := s.getEnrollState(token); !ok {
		t.Fatal("expected enroll state to exist immediately")
	}

	time.Sleep(40 * time.Millisecond)
	if _, ok := s.getEnrollState(token); ok {
		t.Fatal("expected enroll state to expire")
	}
}
