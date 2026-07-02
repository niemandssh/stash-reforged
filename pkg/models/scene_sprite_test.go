package models

import "testing"

func TestRemoteSceneSpriteHash(t *testing.T) {
	if got := RemoteSceneSpriteHash(42); got != "remote42" {
		t.Fatalf("RemoteSceneSpriteHash(42) = %q, want remote42", got)
	}
	if got := LegacyRemoteSceneSpriteHash(42); got != "remote_42" {
		t.Fatalf("LegacyRemoteSceneSpriteHash(42) = %q, want remote_42", got)
	}
}
