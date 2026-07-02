package models

import "fmt"

// RemoteSceneSpriteHash returns the generated-asset key for remote embed scenes.
// Must not contain underscores before the "_thumbs.vtt" suffix (chi router limitation).
func RemoteSceneSpriteHash(sceneID int) string {
	return fmt.Sprintf("remote%d", sceneID)
}

// LegacyRemoteSceneSpriteHash is the previous naming scheme, kept for existing files.
func LegacyRemoteSceneSpriteHash(sceneID int) string {
	return fmt.Sprintf("remote_%d", sceneID)
}
