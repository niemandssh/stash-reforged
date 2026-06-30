package models

// SceneVRProjection selects equirectangular / stereo layout for desktop VR-style playback (videojs-vr).
type SceneVRProjection string

const (
	SceneVRProjectionNone        SceneVRProjection = "NONE"
	SceneVRProjectionStereo180LR SceneVRProjection = "STEREO_180_LR"
	SceneVRProjectionStereo360TB SceneVRProjection = "STEREO_360_TB"
	SceneVRProjectionMono360     SceneVRProjection = "MONO_360"
)
