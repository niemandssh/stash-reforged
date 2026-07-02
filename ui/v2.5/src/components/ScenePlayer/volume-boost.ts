import videojs, { VideoJsPlayer } from "video.js";

export const MAX_PLAYER_VOLUME = 2;

function clampVolume(level: number) {
  return Math.max(0, Math.min(MAX_PLAYER_VOLUME, level));
}

function isSingleLeftClick(event: Event) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const mouseEvent = event as any;
  if (mouseEvent.button !== undefined && mouseEvent.button !== 0) {
    return false;
  }
  if (mouseEvent.buttons !== undefined && mouseEvent.buttons !== 1) {
    return false;
  }
  return true;
}

class VolumeBoostPlugin extends videojs.getPlugin("plugin") {
  private level = 1;
  private audioContext: AudioContext | null = null;
  private gainNode: GainNode | null = null;
  private audioBoostAvailable = false;

  constructor(player: VideoJsPlayer, options?: { initialLevel?: number }) {
    super(player, options);

    if (options?.initialLevel !== undefined) {
      this.level = clampVolume(options.initialLevel);
    }

    player.addClass("vjs-volume-boost");

    player.ready(() => {
      this.setupAudioGraph();
      this.applyLevel(false);
      this.patchVolumeMethod();
    });

    player.on("volumechange", () => {
      if (this.player.muted()) {
        this.applyGainOnly(0);
      } else {
        this.applyLevel(false);
      }
    });
  }

  getLevel() {
    return this.level;
  }

  setLevel(level: number, triggerChange = true) {
    this.level = clampVolume(level);
    this.applyLevel(triggerChange);
  }

  private patchVolumeMethod() {
    const player = this.player;

    player.volume = (percent?: number) => {
      if (percent === undefined) {
        return this.level;
      }
      this.setLevel(percent);
      return this.level;
    };
  }

  private setupAudioGraph() {
    try {
      const tech = this.player.tech(true);
      const el = tech?.el() as HTMLVideoElement | undefined;
      if (!el) {
        return;
      }

      this.audioContext = new AudioContext();
      const source = this.audioContext.createMediaElementSource(el);
      this.gainNode = this.audioContext.createGain();
      source.connect(this.gainNode);
      this.gainNode.connect(this.audioContext.destination);
      this.audioBoostAvailable = true;

      this.player.on("play", () => {
        if (this.audioContext?.state === "suspended") {
          void this.audioContext.resume();
        }
      });
    } catch {
      this.audioBoostAvailable = false;
      if (this.level > 1) {
        this.level = 1;
      }
    }
  }

  private getMediaElement(): HTMLVideoElement | null {
    const tech = this.player.tech(true);
    return (tech?.el() as HTMLVideoElement | undefined) ?? null;
  }

  private applyGainOnly(gain: number) {
    if (this.gainNode) {
      this.gainNode.gain.value = gain;
    }
  }

  private applyLevel(triggerChange: boolean) {
    const el = this.getMediaElement();
    if (!el) {
      return;
    }

    if (this.player.muted() || this.level <= 0) {
      this.applyGainOnly(0);
      return;
    }

    if (!this.audioBoostAvailable || this.level <= 1) {
      el.volume = this.level;
      this.applyGainOnly(1);
    } else {
      el.volume = 1;
      this.applyGainOnly(this.level);
    }

    if (triggerChange) {
      this.player.trigger("volumechange");
      this.player.trigger("boostvolumechange");
    }
  }
}

const BaseVolumeBar = videojs.getComponent("VolumeBar");
const BaseMouseVolumeLevelDisplay = videojs.getComponent(
  "MouseVolumeLevelDisplay"
);

class BoostMouseVolumeLevelDisplay extends BaseMouseVolumeLevelDisplay {
  update(
    rangeBarRect: DOMRect,
    rangeBarPoint: number,
    vertical: boolean
  ) {
    const volume = 100 * rangeBarPoint * MAX_PLAYER_VOLUME;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const tooltip = this.getChild("volumeLevelTooltip") as any;
    tooltip.updateVolume(rangeBarRect, rangeBarPoint, vertical, volume, () => {
      if (vertical) {
        this.el_.style.bottom = `${rangeBarRect.height * rangeBarPoint}px`;
      } else {
        this.el_.style.left = `${rangeBarRect.width * rangeBarPoint}px`;
      }
    });
  }
}

class BoostVolumeBar extends BaseVolumeBar {
  private getBoostPlugin(): VolumeBoostPlugin {
    return this.player_.volumeBoost();
  }

  getPercent() {
    if (this.player_.muted()) {
      return 0;
    }
    return this.getBoostPlugin().getLevel() / MAX_PLAYER_VOLUME;
  }

  handleMouseMove(event: Event) {
    const mouseVolumeLevelDisplay = this.getChild("mouseVolumeLevelDisplay");
    if (mouseVolumeLevelDisplay) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const display = mouseVolumeLevelDisplay as any;
      const volumeBarEl = this.el();
      const volumeBarRect = volumeBarEl.getBoundingClientRect();
      const vertical = this.vertical();
      const volumeBarPoint = videojs.dom.getPointerPosition(
        volumeBarEl,
        event
      );
      const point = vertical ? volumeBarPoint.y : volumeBarPoint.x;
      const clamped = Math.max(0, Math.min(1, point));
      display.update(volumeBarRect, clamped, vertical);
    }

    if (!isSingleLeftClick(event)) {
      return;
    }

    this.checkMuted();
    this.getBoostPlugin().setLevel(this.calculateDistance(event) * MAX_PLAYER_VOLUME);
  }

  stepForward() {
    this.checkMuted();
    const plugin = this.getBoostPlugin();
    plugin.setLevel(plugin.getLevel() + 0.2);
  }

  stepBack() {
    this.checkMuted();
    const plugin = this.getBoostPlugin();
    plugin.setLevel(plugin.getLevel() - 0.2);
  }

  updateARIAAttributes() {
    const ariaValue = this.player_.muted()
      ? 0
      : Math.round(this.getBoostPlugin().getLevel() * 100);
    this.el_.setAttribute("aria-valuenow", String(ariaValue));
    this.el_.setAttribute("aria-valuetext", `${ariaValue}%`);
  }
}

const baseVolumeBarOptions = BaseVolumeBar.prototype.options_;
BoostVolumeBar.prototype.options_ = {
  ...baseVolumeBarOptions,
  children: baseVolumeBarOptions.children.map(
    (child: string | { name: string }) => {
      const childName = typeof child === "string" ? child : child.name;
      if (childName === "mouseVolumeLevelDisplay") {
        return {
          name: "mouseVolumeLevelDisplay",
          componentClass: BoostMouseVolumeLevelDisplay,
        };
      }
      return child;
    }
  ),
};

videojs.registerComponent("VolumeBar", BoostVolumeBar);
videojs.registerPlugin("volumeBoost", VolumeBoostPlugin);

/* eslint-disable @typescript-eslint/naming-convention */
declare module "video.js" {
  interface VideoJsPlayer {
    volumeBoost: () => VolumeBoostPlugin;
  }
  interface VideoJsPlayerPluginOptions {
    volumeBoost?: { initialLevel?: number };
  }
}

export default VolumeBoostPlugin;
