interface IEmbedPlaybackOptions {
  embedSrc: string;
  startTime?: number | null;
  endTime?: number | null;
  resumeTime?: number | null;
  lastPlayedAt?: string | null;
  trimEnabled: boolean;
  initialTimestamp?: number;
  alwaysStartFromBeginning?: boolean;
  rememberPlayPositionFor5Minutes?: boolean;
}

function clampEmbedPlaybackTime(
  time: number,
  sceneStart: number,
  sceneEnd: number
): number {
  if (sceneStart > 0 && time < sceneStart) {
    return sceneStart;
  }
  if (sceneEnd > 0 && time > sceneEnd) {
    return sceneStart > 0 ? sceneStart : 0;
  }
  return time;
}

export function computeEmbedPlaybackStart({
  startTime,
  endTime,
  resumeTime,
  lastPlayedAt,
  initialTimestamp = 0,
  alwaysStartFromBeginning = false,
  rememberPlayPositionFor5Minutes = false,
}: Omit<IEmbedPlaybackOptions, "embedSrc" | "trimEnabled">): number {
  const sceneStart = (startTime ?? 0) > 0 ? (startTime ?? 0) : 0;
  const sceneEnd = (endTime ?? 0) > 0 ? (endTime ?? 0) : 0;

  if (initialTimestamp > 0) {
    return clampEmbedPlaybackTime(initialTimestamp, sceneStart, sceneEnd);
  }

  let effectiveResume = 0;
  let effectiveAlwaysStartFromBeginning = alwaysStartFromBeginning;

  if (alwaysStartFromBeginning && rememberPlayPositionFor5Minutes) {
    const resume = resumeTime ?? 0;
    if (resume > 0 && lastPlayedAt) {
      const lastPlayedAtMs = new Date(lastPlayedAt).getTime();
      const fiveMinutesMs = 5 * 60 * 1000;
      if (Date.now() - lastPlayedAtMs <= fiveMinutesMs) {
        effectiveResume = resume;
        effectiveAlwaysStartFromBeginning = false;
      }
    }
  } else if (!alwaysStartFromBeginning) {
    effectiveResume = resumeTime ?? 0;
  }

  if (!effectiveAlwaysStartFromBeginning && effectiveResume > 0) {
    return clampEmbedPlaybackTime(effectiveResume, sceneStart, sceneEnd);
  }

  return sceneStart;
}

function appendStartParam(url: URL, startSeconds: number): void {
  const host = url.hostname.toLowerCase();

  if (host.includes("pornhub")) {
    const start =
      startSeconds === Math.trunc(startSeconds)
        ? String(Math.round(startSeconds))
        : String(startSeconds);
    url.searchParams.set("t", start);
    return;
  }

  url.searchParams.set("t", String(Math.round(startSeconds)));
}

export function buildEmbedPlaybackUrl(options: IEmbedPlaybackOptions): string {
  const start = computeEmbedPlaybackStart(options);
  if (start <= 0) {
    return options.embedSrc;
  }

  try {
    const url = new URL(options.embedSrc);
    appendStartParam(url, start);
    return url.toString();
  } catch {
    return options.embedSrc;
  }
}
