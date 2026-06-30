const BUFFER_MAX = 100;
const STORAGE_PREFIX = "stash-random-scene-buffer:";

function storageKey(key: string): string {
  return `${STORAGE_PREFIX}${key}`;
}

export function getRandomSceneBuffer(key: string): string[] {
  try {
    const raw = localStorage.getItem(storageKey(key));
    if (!raw) {
      return [];
    }
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed.filter((id): id is string => typeof id === "string");
  } catch {
    return [];
  }
}

function setRandomSceneBuffer(key: string, buffer: string[]): void {
  try {
    localStorage.setItem(storageKey(key), JSON.stringify(buffer));
  } catch {
    // ignore quota / private mode errors
  }
}

export function clearRandomSceneBuffer(key: string): void {
  try {
    localStorage.removeItem(storageKey(key));
  } catch {
    // ignore
  }
}

/** True when the pool is smaller than the buffer cap and every scene has been seen. */
export function shouldClearRandomSceneBuffer(
  buffer: string[],
  totalCount: number
): boolean {
  return totalCount < BUFFER_MAX && buffer.length >= totalCount;
}

/**
 * Prepends a scene id. Skips if already present. Trims to BUFFER_MAX from the end.
 */
export function addToRandomSceneBuffer(key: string, sceneId: string): void {
  const buffer = getRandomSceneBuffer(key);
  if (buffer.includes(sceneId)) {
    return;
  }
  const next = [sceneId, ...buffer].slice(0, BUFFER_MAX);
  setRandomSceneBuffer(key, next);
}

export function randomSceneBufferKeyForRating(
  ratingThreshold: number
): string {
  return `rating-${ratingThreshold}`;
}

export const RANDOM_SCENE_BUFFER_KEY_UNORGANISED = "unorganised";
