import { GQL } from "src/core/generated-graphql";

/**
 * Detects if a scene is likely an HLS video based on common characteristics
 */
export function isHLSVideo(scene: GQL.SceneDataFragment): boolean {
  if (!scene.files || scene.files.length === 0) {
    return false;
  }

  const primaryFile = scene.files[0];
  if (!primaryFile) {
    return false;
  }

  // HLS videos often have these characteristics:
  // 1. H.264 video and AAC audio (common HLS combination)
  // 2. Duration is often a multiple of 2 seconds (HLS segment length)
  // 3. May have specific metadata or timing issues

  // Check for H.264 video and AAC audio (common HLS combination)
  if (primaryFile.video_codec !== "h264" || primaryFile.audio_codec !== "aac") {
    return false;
  }

  // Check if duration is a multiple of 2 seconds (typical HLS segment length)
  // Allow some tolerance for rounding errors
  if (primaryFile.duration && primaryFile.duration > 0) {
    const segmentLength = 2.0;
    const remainder = primaryFile.duration % segmentLength;
    // Consider it HLS if remainder is very close to 0 or very close to segmentLength
    if (remainder < 0.1 || remainder > (segmentLength - 0.1)) {
      return true;
    }
  }

  return false;
}
