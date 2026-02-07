import videojs, { VideoJsPlayer } from "video.js";

const intervalSeconds = 1; // check every second
const sendInterval = 10; // send every 10 seconds

class TrackActivityPlugin extends videojs.getPlugin("plugin") {
  totalPlayDuration = 0;
  currentPlayDuration = 0;
  minimumPlayPercent = 0;
  incrementPlayCount: () => Promise<void> = () => {
    return Promise.resolve();
  };
  saveActivity: (resumeTime: number, playDuration: number) => Promise<void> =
    () => {
      return Promise.resolve();
    };

  private enabled = false;
  private playCountIncremented = false;
  private intervalID: number | undefined;

  private lastResumeTime = 0;
  private lastDuration = 0;

  private boundBeforeUnload: () => void;
  private boundVisibilityChange: () => void;

  constructor(player: VideoJsPlayer) {
    super(player);

    player.on("playing", () => {
      this.start();
    });

    player.on("waiting", () => {
      this.stop();
    });

    player.on("stalled", () => {
      this.stop();
    });

    player.on("pause", () => {
      this.stop();
    });

    player.on("dispose", () => {
      this.stop();
      this.removePageListeners();
    });

    // Save activity before page unload (refresh, close, navigate away)
    this.boundBeforeUnload = () => {
      this.sendActivitySync();
    };
    window.addEventListener("beforeunload", this.boundBeforeUnload);

    // Save activity when page becomes hidden (tab switch, minimize)
    this.boundVisibilityChange = () => {
      if (document.visibilityState === "hidden") {
        this.sendActivitySync();
      }
    };
    document.addEventListener("visibilitychange", this.boundVisibilityChange);
  }

  private removePageListeners() {
    window.removeEventListener("beforeunload", this.boundBeforeUnload);
    document.removeEventListener("visibilitychange", this.boundVisibilityChange);
  }

  private start() {
    if (this.enabled && !this.intervalID) {
      this.intervalID = window.setInterval(() => {
        this.intervalHandler();
      }, intervalSeconds * 1000);
      this.lastResumeTime = this.player.currentTime();
      this.lastDuration = this.player.duration();
    }
  }

  private stop() {
    if (this.intervalID) {
      window.clearInterval(this.intervalID);
      this.intervalID = undefined;
      this.sendActivity();
    }
  }

  reset() {
    this.stop();
    this.totalPlayDuration = 0;
    this.currentPlayDuration = 0;
    this.playCountIncremented = false;
  }

  setEnabled(enabled: boolean) {
    this.enabled = enabled;
    if (!enabled) {
      this.stop();
    } else if (!this.player.paused()) {
      this.start();
    }
  }

  private intervalHandler() {
    if (!this.enabled || !this.player) return;

    this.lastResumeTime = this.player.currentTime();
    this.lastDuration = this.player.duration();

    this.totalPlayDuration += intervalSeconds;
    this.currentPlayDuration += intervalSeconds;
    if (this.totalPlayDuration % sendInterval === 0) {
      this.sendActivity();
    }
  }

  private sendActivity() {
    if (!this.enabled) return;

    if (this.totalPlayDuration > 0) {
      let resumeTime = this.player?.currentTime() ?? this.lastResumeTime;
      const videoDuration = this.player?.duration() ?? this.lastDuration;
      const percentCompleted = (100 / videoDuration) * resumeTime;
      const percentPlayed = (100 / videoDuration) * this.totalPlayDuration;

      if (
        !this.playCountIncremented &&
        percentPlayed >= this.minimumPlayPercent
      ) {
        this.incrementPlayCount();
        this.playCountIncremented = true;
      }

      // if video is 98% or more complete then reset resume_time
      if (percentCompleted >= 98) {
        resumeTime = 0;
      }

      this.saveActivity(resumeTime, this.currentPlayDuration);
      this.currentPlayDuration = 0;
    }
  }

  // Synchronous version for beforeunload/visibilitychange
  // Saves current position even if not enough time has accumulated
  private sendActivitySync() {
    if (!this.enabled) return;

    const resumeTime = this.player?.currentTime() ?? this.lastResumeTime;
    const videoDuration = this.player?.duration() ?? this.lastDuration;

    if (resumeTime > 0 && videoDuration > 0) {
      const percentCompleted = (100 / videoDuration) * resumeTime;

      // if video is 98% or more complete then reset resume_time
      const finalResumeTime = percentCompleted >= 98 ? 0 : resumeTime;

      // Send with current accumulated play duration
      this.saveActivity(finalResumeTime, this.currentPlayDuration);
      this.currentPlayDuration = 0;
    }
  }
}

// Register the plugin with video.js.
videojs.registerPlugin("trackActivity", TrackActivityPlugin);

/* eslint-disable @typescript-eslint/naming-convention */
declare module "video.js" {
  interface VideoJsPlayer {
    trackActivity: () => TrackActivityPlugin;
  }
  interface VideoJsPlayerPluginOptions {
    trackActivity?: {};
  }
}

export default TrackActivityPlugin;
