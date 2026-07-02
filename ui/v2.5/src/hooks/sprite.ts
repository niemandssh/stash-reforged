import { useEffect, useState } from "react";
import { WebVTT } from "videojs-vtt.js";

export interface ISceneSpriteInfo {
  url: string;
  start: number;
  end: number;
  x: number;
  y: number;
  w: number;
  h: number;
}

function getSpriteInfo(vttPath: string, response: string) {
  const sprites: ISceneSpriteInfo[] = [];

  const parser = new WebVTT.Parser(window, WebVTT.StringDecoder());
  parser.oncue = (cue: VTTCue) => {
    const match = cue.text.match(/^([^#]*)#xywh=(\d+),(\d+),(\d+),(\d+)$/i);
    if (!match) return;

    sprites.push({
      url: new URL(match[1], vttPath).href,
      start: cue.startTime,
      end: cue.endTime,
      x: Number(match[2]),
      y: Number(match[3]),
      w: Number(match[4]),
      h: Number(match[5]),
    });
  };
  parser.parse(response);
  parser.flush();

  return sprites;
}

// useSpriteInfo is a hook that fetches a VTT file and parses it for sprite information.
// If the vttPath is undefined, the hook will return undefined.
// If the response is not ok, the hook will return null. This usually indicates missing sprite.
export function useSpriteInfo(vttPath: string | undefined) {
  const [spriteInfo, setSpriteInfo] = useState<
    ISceneSpriteInfo[] | undefined | null
  >();

  useEffect(() => {
    if (!vttPath) {
      setSpriteInfo(undefined);
      return;
    }

    let cancelled = false;
    let attempts = 0;
    const maxAttempts = 120;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;

    const fetchSprites = () => {
      fetch(vttPath).then((response) => {
        if (cancelled) {
          return;
        }

        if (response.ok) {
          response.text().then((text) => {
            if (!cancelled) {
              setSpriteInfo(getSpriteInfo(vttPath, text));
            }
          });
          return;
        }

        if (response.status === 503 && attempts < maxAttempts) {
          attempts += 1;
          retryTimer = setTimeout(fetchSprites, 5000);
          return;
        }

        setSpriteInfo(null);
      });
    };

    setSpriteInfo(undefined);
    fetchSprites();

    return () => {
      cancelled = true;
      if (retryTimer) {
        clearTimeout(retryTimer);
      }
    };
  }, [vttPath]);

  return spriteInfo;
}
