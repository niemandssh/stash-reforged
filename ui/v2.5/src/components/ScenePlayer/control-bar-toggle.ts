/* eslint-disable @typescript-eslint/naming-convention */
import videojs, { VideoJsPlayer } from "video.js";

// Register translations for Video.js
videojs.addLanguage("en", {
  "Toggle Control Bar": "Toggle Control Bar",
  "Hide Control Bar": "Hide Control Bar",
  "Show Control Bar": "Show Control Bar",
});

videojs.addLanguage("ru", {
  "Toggle Control Bar": "Переключить панель управления",
  "Hide Control Bar": "Скрыть панель управления",
  "Show Control Bar": "Показать панель управления",
});

class ControlBarToggleButton extends videojs.getComponent("button") {
  public isControlBarVisible: boolean = true;

  constructor(player: VideoJsPlayer, options?: videojs.ComponentOptions) {
    super(player, options);
    this.controlText(this.localize("Toggle Control Bar"));
    this.addClass("vjs-control-bar-toggle");

    // Initial icon
    this.updateIcon();
  }

  buildCSSClass() {
    return `vjs-control-bar-toggle-button ${super.buildCSSClass()}`;
  }

  updateIcon() {
    // Remove existing icon classes
    this.removeClass("vjs-icon-eye");
    this.removeClass("vjs-icon-eye-slash");

    if (this.isControlBarVisible) {
      this.addClass("vjs-icon-eye-slash"); // Eye-slash icon for "hide"
      this.controlText(this.localize("Hide Control Bar"));
    } else {
      this.addClass("vjs-icon-eye"); // Eye icon for "show"
      this.controlText(this.localize("Show Control Bar"));
    }
  }

  handleClick() {
    const player = this.player();
    const { controlBar } = player;

    if (!controlBar) return;

    this.isControlBarVisible = !this.isControlBarVisible;

    if (this.isControlBarVisible) {
      // Show control bar
      player.removeClass("vjs-control-bar-hidden");
      controlBar.show();
    } else {
      // Hide control bar
      player.addClass("vjs-control-bar-hidden");
      controlBar.hide();
    }

    this.updateIcon();

    // Store preference in localStorage
    try {
      localStorage.setItem(
        "stash-control-bar-visible",
        String(this.isControlBarVisible)
      );
    } catch (e) {
      // Ignore localStorage errors
    }
  }
}

class ControlBarTogglePlugin extends videojs.getPlugin("plugin") {
  private toggleButton?: ControlBarToggleButton;
  private floatingButton?: HTMLDivElement;

  constructor(player: VideoJsPlayer) {
    super(player);
    player.ready(() => {
      this.ready();
    });
  }

  private createFloatingButton(): HTMLDivElement {
    const button = document.createElement("div");
    button.className = "vjs-floating-toggle-button";
    button.innerHTML =
      '<span class="vjs-icon-placeholder vjs-icon-eye"></span>';

    // Set localized title
    button.title = "Show Control Bar (C)";

    button.addEventListener("click", () => {
      if (this.toggleButton) {
        this.toggleButton.handleClick();
      }
    });

    return button;
  }

  private updateFloatingButtonVisibility(isControlBarVisible: boolean) {
    if (!this.floatingButton) return;

    if (isControlBarVisible) {
      this.floatingButton.style.display = "none";
    } else {
      this.floatingButton.style.display = "flex";
    }
  }

  ready() {
    const { player } = this;

    // Create floating button and add to player
    this.floatingButton = this.createFloatingButton();
    player.el().appendChild(this.floatingButton);

    // Add button to control bar (at the end)
    this.toggleButton = player.controlBar.addChild(
      "ControlBarToggleButton",
      {},
      999
    ) as ControlBarToggleButton;

    // Override the toggle button's handleClick to also update floating button
    const originalHandleClick = this.toggleButton.handleClick.bind(
      this.toggleButton
    );
    this.toggleButton.handleClick = () => {
      originalHandleClick();
      this.updateFloatingButtonVisibility(
        this.toggleButton!.isControlBarVisible
      );
    };

    // Restore preference from localStorage
    try {
      const savedState = localStorage.getItem("stash-control-bar-visible");
      if (savedState === "false" && this.toggleButton) {
        player.addClass("vjs-control-bar-hidden");
        player.controlBar.hide();
        this.toggleButton.isControlBarVisible = false;
        this.toggleButton.updateIcon();
        this.updateFloatingButtonVisibility(false);
      } else {
        this.updateFloatingButtonVisibility(true);
      }
    } catch (e) {
      // Ignore localStorage errors
      this.updateFloatingButtonVisibility(true);
    }

    // Add keyboard shortcut (C key) to toggle control bar
    player.on("keydown", (event: KeyboardEvent) => {
      if (event.key === "c" || event.key === "C") {
        if (this.toggleButton) {
          this.toggleButton.handleClick();
        }
      }
    });
  }
}

videojs.registerComponent("ControlBarToggleButton", ControlBarToggleButton);
videojs.registerPlugin("controlBarToggle", ControlBarTogglePlugin);

declare module "video.js" {
  interface VideoJsPlayer {
    controlBarToggle: () => ControlBarTogglePlugin;
  }
  interface VideoJsPlayerPluginOptions {
    controlBarToggle?: {};
  }
}

export default ControlBarTogglePlugin;
