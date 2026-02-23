import videojs, { VideoJsPlayer } from "video.js";
import localForage from "localforage";

const levelKey = "volume-level";
const mutedKey = "volume-muted";

interface IPersistVolumeOptions {
  enabled?: boolean;
  onVolumeChange?: (level: number, muted: boolean) => void;
  useLocalForage?: boolean;
}

class PersistVolumePlugin extends videojs.getPlugin("plugin") {
  enabled: boolean;
  private useLocalForage: boolean;
  private onVolumeChange?: (level: number, muted: boolean) => void;

  constructor(player: VideoJsPlayer, options?: IPersistVolumeOptions) {
    super(player, options);

    this.enabled = options?.enabled ?? true;
    this.onVolumeChange = options?.onVolumeChange;
    this.useLocalForage = options?.useLocalForage ?? !options?.onVolumeChange;

    player.on("volumechange", () => {
      if (!this.enabled) return;
      const level = player.volume();
      const muted = player.muted();
      if (this.onVolumeChange) {
        this.onVolumeChange(level, muted);
      }
      if (this.useLocalForage) {
        localForage.setItem(levelKey, level);
        localForage.setItem(mutedKey, muted);
      }
    });

    player.ready(() => {
      this.ready();
    });
  }

  private ready() {
    if (!this.useLocalForage) return;
    localForage.getItem<number>(levelKey).then((value) => {
      if (value !== null) {
        this.player.volume(value);
      }
    });
    localForage.getItem<boolean>(mutedKey).then((value) => {
      if (value !== null) {
        this.player.muted(value);
      }
    });
  }
}

// Register the plugin with video.js.
videojs.registerPlugin("persistVolume", PersistVolumePlugin);

/* eslint-disable @typescript-eslint/naming-convention */
declare module "video.js" {
  interface VideoJsPlayer {
    persistVolume: () => PersistVolumePlugin;
  }
  interface VideoJsPlayerPluginOptions {
    persistVolume?: IPersistVolumeOptions;
  }
}

export default PersistVolumePlugin;
