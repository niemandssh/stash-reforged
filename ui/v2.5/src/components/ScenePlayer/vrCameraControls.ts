import { VideoJsPlayer } from "video.js";
import type { Plugin as VideoJsVRPlugin } from "videojs-vr";

interface Vector3Like {
  x: number;
  y: number;
  z: number;
}

interface PerspectiveCameraLike {
  fov: number;
  updateProjectionMatrix: () => void;
}

interface OrbitControlsLike {
  enablePan: boolean;
  enableZoom: boolean;
  object: PerspectiveCameraLike;
  target: Vector3Like;
  rotateLeft: (angle: number) => void;
  rotateUp: (angle: number) => void;
  update: () => void;
}

interface OrbitOrientationControlsLike {
  orbit: OrbitControlsLike;
}

interface VideoJsVRPluginInternal extends VideoJsVRPlugin {
  controls3d?: OrbitOrientationControlsLike | { update: () => void };
  renderedCanvas?: HTMLCanvasElement;
  initialized_?: boolean;
}

const BASE_ZOOM_KEY_STEP = 0.12;
const BASE_ZOOM_WHEEL_SCALE = 0.0025;

export const DEFAULT_VR_ROTATE_SPEED = 0.008;
export const DEFAULT_VR_ZOOM_SPEED = 1;

export interface VRCameraControlSettings {
  rotateSpeed: number;
  zoomKeyStep: number;
  zoomWheelScale: number;
}

export const DEFAULT_VR_CAMERA_SETTINGS: VRCameraControlSettings = {
  rotateSpeed: DEFAULT_VR_ROTATE_SPEED,
  zoomKeyStep: BASE_ZOOM_KEY_STEP,
  zoomWheelScale: BASE_ZOOM_WHEEL_SCALE,
};

export function vrCameraSettingsFromFilters(
  filters?: {
    vr_rotate_speed?: number | null;
    vr_zoom_speed?: number | null;
  } | null
): VRCameraControlSettings {
  const zoomMultiplier = filters?.vr_zoom_speed ?? DEFAULT_VR_ZOOM_SPEED;

  return {
    rotateSpeed: filters?.vr_rotate_speed ?? DEFAULT_VR_ROTATE_SPEED,
    zoomKeyStep: BASE_ZOOM_KEY_STEP * zoomMultiplier,
    zoomWheelScale: BASE_ZOOM_WHEEL_SCALE * zoomMultiplier,
  };
}

function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }

  const tag = target.tagName;
  return (
    tag === "INPUT" ||
    tag === "TEXTAREA" ||
    tag === "SELECT" ||
    target.isContentEditable
  );
}

function getOrbitControls(vr: VideoJsVRPlugin): OrbitControlsLike | null {
  const controls3d = (vr as VideoJsVRPluginInternal).controls3d;
  if (!controls3d || !("orbit" in controls3d) || !controls3d.orbit) {
    return null;
  }

  return controls3d.orbit;
}

export class VRCameraControls {
  private readonly player: VideoJsPlayer;

  private vr: VideoJsVRPlugin | null = null;

  private keyTarget: HTMLElement | null = null;

  private wheelElement: HTMLElement | null = null;

  private active = false;

  private ticking = false;

  private settings: VRCameraControlSettings = { ...DEFAULT_VR_CAMERA_SETTINGS };

  private readonly heldKeys = new Set<string>();

  private readonly keyDownHandler = (event: KeyboardEvent) => {
    if (!this.active || isEditableTarget(event.target)) {
      return;
    }

    const code = event.code;
    if (
      code === "KeyW" ||
      code === "KeyA" ||
      code === "KeyS" ||
      code === "KeyD" ||
      code === "KeyQ" ||
      code === "KeyE"
    ) {
      event.preventDefault();
      event.stopPropagation();
      this.heldKeys.add(code);
    }
  };

  private readonly keyUpHandler = (event: KeyboardEvent) => {
    this.heldKeys.delete(event.code);
  };

  private readonly wheelHandler = (event: WheelEvent) => {
    if (!this.active || !this.vr || !getOrbitControls(this.vr)) {
      return;
    }

    event.preventDefault();
    this.adjustZoom(event.deltaY * this.settings.zoomWheelScale);
  };

  private readonly focusHandler = () => {
    this.keyTarget?.focus();
  };

  private readonly tickHandler = () => {
    if (!this.active || !this.vr) {
      this.ticking = false;
      return;
    }

    const orbit = getOrbitControls(this.vr);
    if (orbit) {
      const { rotateSpeed, zoomKeyStep } = this.settings;

      if (this.heldKeys.has("KeyW")) {
        orbit.rotateUp(-rotateSpeed);
      }
      if (this.heldKeys.has("KeyS")) {
        orbit.rotateUp(rotateSpeed);
      }
      if (this.heldKeys.has("KeyA")) {
        orbit.rotateLeft(-rotateSpeed);
      }
      if (this.heldKeys.has("KeyD")) {
        orbit.rotateLeft(rotateSpeed);
      }
      if (this.heldKeys.has("KeyQ")) {
        this.adjustZoom(zoomKeyStep, orbit);
      }
      if (this.heldKeys.has("KeyE")) {
        this.adjustZoom(-zoomKeyStep, orbit);
      }

      orbit.update();
    }

    this.player.requestAnimationFrame(this.tickHandler);
  };

  constructor(player: VideoJsPlayer) {
    this.player = player;
  }

  setSettings(settings: VRCameraControlSettings) {
    this.settings = settings;
  }

  attach(vr: VideoJsVRPlugin) {
    this.detachListeners();

    this.vr = vr;
    this.active = true;

    this.keyTarget = this.player.el() as HTMLElement;
    const renderedCanvas = (vr as VideoJsVRPluginInternal).renderedCanvas;
    this.wheelElement = renderedCanvas ?? this.keyTarget;

    this.keyTarget.addEventListener("keydown", this.keyDownHandler, true);
    this.keyTarget.addEventListener("keyup", this.keyUpHandler, true);
    window.addEventListener("keydown", this.keyDownHandler, true);
    window.addEventListener("keyup", this.keyUpHandler, true);
    this.wheelElement.addEventListener("wheel", this.wheelHandler, {
      passive: false,
    });
    this.wheelElement.addEventListener("mousedown", this.focusHandler);

    if (!this.ticking) {
      this.ticking = true;
      this.player.requestAnimationFrame(this.tickHandler);
    }
  }

  detach() {
    this.active = false;
    this.ticking = false;
    this.heldKeys.clear();
    this.vr = null;
    this.detachListeners();
  }

  private detachListeners() {
    this.keyTarget?.removeEventListener("keydown", this.keyDownHandler, true);
    this.keyTarget?.removeEventListener("keyup", this.keyUpHandler, true);
    window.removeEventListener("keydown", this.keyDownHandler, true);
    window.removeEventListener("keyup", this.keyUpHandler, true);
    this.wheelElement?.removeEventListener("wheel", this.wheelHandler);
    this.wheelElement?.removeEventListener("mousedown", this.focusHandler);
    this.keyTarget = null;
    this.wheelElement = null;
  }

  private adjustZoom(delta: number, orbit?: OrbitControlsLike | null) {
    const controls = orbit ?? (this.vr ? getOrbitControls(this.vr) : null);
    if (!controls) {
      return;
    }

    const camera = controls.object;
    // No zoom cap — only keep FOV valid for perspective projection.
    camera.fov = Math.max(0.1, camera.fov + delta);
    camera.updateProjectionMatrix();
  }
}

export function attachVRCameraControlsWhenReady(
  player: VideoJsPlayer,
  vr: VideoJsVRPlugin,
  controls: VRCameraControls
) {
  controls.attach(vr);
}
