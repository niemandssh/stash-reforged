import * as GQL from "src/core/generated-graphql";
import { VIDEO_PLAYER_ID } from "src/components/ScenePlayer/util";
import {
  FILTER_DEFAULTS,
  FILTER_DIVIDERS,
  SCALE_DEFAULT,
  ASPECT_RATIO_DEFAULT,
  ASPECT_RATIO_DIVIDER,
  ROTATE_DEFAULT,
  ROTATE_DIVIDER,
} from "src/utils/videoFilters";

type MediaElement = HTMLVideoElement | HTMLCanvasElement;

type FilterValues = {
  brightness: number;
  contrast: number;
  gamma: number;
  saturate: number;
  hueRotate: number;
  whiteBalance: number;
  red: number;
  green: number;
  blue: number;
  blur: number;
};

type TransformValues = {
  rotate: number;
  scale: number;
  aspectRatio: number;
};

export interface ICapturedSceneScreenshot {
  dataUrl: string;
  at?: number;
}

export async function captureFilteredSceneScreenshot(
  scene: GQL.SceneDataFragment
): Promise<ICapturedSceneScreenshot> {
  const mediaElement = findMediaElement();
  if (!mediaElement) {
    throw new Error("Video player is not available yet.");
  }

  const { width: sourceWidth, height: sourceHeight } =
    getSourceDimensions(mediaElement);

  if (!sourceWidth || !sourceHeight) {
    throw new Error("Video frame is not ready yet. Try again in a moment.");
  }

  const filters = getFilterValues(scene);
  const transforms = getTransformValues(scene);

  const { canvasWidth, canvasHeight, rotationRad, scaleX, scaleY } =
    computeCanvasGeometry(sourceWidth, sourceHeight, transforms);

  if (!canvasWidth || !canvasHeight) {
    throw new Error("Unable to determine output dimensions for screenshot.");
  }

  const canvas = document.createElement("canvas");
  canvas.width = Math.max(1, Math.round(canvasWidth));
  canvas.height = Math.max(1, Math.round(canvasHeight));

  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Unable to initialise drawing context.");
  }

  ctx.save();
  ctx.translate(canvas.width / 2, canvas.height / 2);
  if (rotationRad !== 0) {
    ctx.rotate(rotationRad);
  }
  if (scaleX !== 1 || scaleY !== 1) {
    ctx.scale(scaleX, scaleY);
  }
  const filterString = buildCanvasFilterString(filters);
  ctx.filter = filterString || "none";
  ctx.drawImage(
    mediaElement,
    -sourceWidth / 2,
    -sourceHeight / 2,
    sourceWidth,
    sourceHeight
  );
  ctx.restore();

  applyChannelAdjustments(ctx, canvas, filters);

  return {
    dataUrl: canvas.toDataURL("image/png"),
    at: mediaElement instanceof HTMLVideoElement ? mediaElement.currentTime : undefined,
  };
}

function findMediaElement(): MediaElement | null {
  const container = document.getElementById(VIDEO_PLAYER_ID);
  if (!container) {
    return null;
  }

  const canvasElements = container.getElementsByTagName("canvas");
  if (canvasElements.length > 0) {
    return canvasElements[0];
  }

  const videoElements = container.getElementsByTagName("video");
  if (videoElements.length > 0) {
    const videoEl = videoElements[0];
    if (
      videoEl.readyState === HTMLMediaElement.HAVE_NOTHING ||
      Number.isNaN(videoEl.videoWidth) ||
      videoEl.videoWidth === 0 ||
      videoEl.videoHeight === 0
    ) {
      return null;
    }
    return videoEl;
  }

  return null;
}

function getSourceDimensions(element: MediaElement) {
  if (element instanceof HTMLVideoElement) {
    return {
      width: element.videoWidth,
      height: element.videoHeight,
    };
  }

  return {
    width: element.width,
    height: element.height,
  };
}

function getFilterValues(scene: GQL.SceneDataFragment): FilterValues {
  const filters = scene.video_filters ?? {};
  return {
    brightness: filters.brightness ?? FILTER_DEFAULTS.brightness,
    contrast: filters.contrast ?? FILTER_DEFAULTS.contrast,
    gamma: filters.gamma ?? FILTER_DEFAULTS.gamma,
    saturate: filters.saturate ?? FILTER_DEFAULTS.saturate,
    hueRotate: filters.hue_rotate ?? FILTER_DEFAULTS.hueRotate,
    whiteBalance: filters.white_balance ?? FILTER_DEFAULTS.whiteBalance,
    red: filters.red ?? FILTER_DEFAULTS.red,
    green: filters.green ?? FILTER_DEFAULTS.green,
    blue: filters.blue ?? FILTER_DEFAULTS.blue,
    blur: filters.blur ?? FILTER_DEFAULTS.blur,
  };
}

function getTransformValues(scene: GQL.SceneDataFragment): TransformValues {
  const transforms = scene.video_transforms ?? {};
  return {
    rotate: transforms.rotate ?? ROTATE_DEFAULT,
    scale: transforms.scale ?? SCALE_DEFAULT,
    aspectRatio: transforms.aspect_ratio ?? ASPECT_RATIO_DEFAULT,
  };
}

function computeCanvasGeometry(
  sourceWidth: number,
  sourceHeight: number,
  transforms: TransformValues
) {
  const rotationDeg = (transforms.rotate - ROTATE_DEFAULT) / ROTATE_DIVIDER;
  const rotationRad = (rotationDeg * Math.PI) / 180;

  const baseScale = transforms.scale / SCALE_DEFAULT;
  let scaleX = baseScale;
  let scaleY = baseScale;

  if (transforms.aspectRatio > ASPECT_RATIO_DEFAULT) {
    scaleX *=
      (ASPECT_RATIO_DIVIDER +
        transforms.aspectRatio -
        ASPECT_RATIO_DEFAULT) /
      ASPECT_RATIO_DIVIDER;
  } else if (transforms.aspectRatio < ASPECT_RATIO_DEFAULT) {
    scaleY *=
      (ASPECT_RATIO_DIVIDER +
        ASPECT_RATIO_DEFAULT -
        transforms.aspectRatio) /
      ASPECT_RATIO_DIVIDER;
  }

  const scaledWidth = sourceWidth * scaleX;
  const scaledHeight = sourceHeight * scaleY;

  const cos = Math.abs(Math.cos(rotationRad));
  const sin = Math.abs(Math.sin(rotationRad));

  const canvasWidth = scaledWidth * cos + scaledHeight * sin;
  const canvasHeight = scaledWidth * sin + scaledHeight * cos;

  return { canvasWidth, canvasHeight, rotationRad, scaleX, scaleY };
}

function buildCanvasFilterString(filters: FilterValues) {
  const parts: string[] = [];

  if (filters.contrast !== FILTER_DEFAULTS.contrast) {
    parts.push(`contrast(${filters.contrast}%)`);
  }

  if (filters.brightness !== FILTER_DEFAULTS.brightness) {
    parts.push(`brightness(${filters.brightness}%)`);
  }

  if (filters.saturate !== FILTER_DEFAULTS.saturate) {
    parts.push(`saturate(${filters.saturate}%)`);
  }

  if (filters.hueRotate !== FILTER_DEFAULTS.hueRotate) {
    parts.push(`hue-rotate(${filters.hueRotate}deg)`);
  }

  if (filters.blur > FILTER_DEFAULTS.blur) {
    const blurPx = filters.blur / FILTER_DIVIDERS.blur;
    parts.push(`blur(${blurPx}px)`);
  }

  return parts.join(" ").trim();
}

function applyChannelAdjustments(
  ctx: CanvasRenderingContext2D,
  canvas: HTMLCanvasElement,
  filters: FilterValues
) {
  const needsMatrix =
    filters.whiteBalance !== FILTER_DEFAULTS.whiteBalance ||
    filters.red !== FILTER_DEFAULTS.red ||
    filters.green !== FILTER_DEFAULTS.green ||
    filters.blue !== FILTER_DEFAULTS.blue;

  const needsGamma = filters.gamma !== FILTER_DEFAULTS.gamma;

  if (!needsMatrix && !needsGamma) {
    return;
  }

  const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
  const { data } = imageData;

  const wbMatrixValue =
    (filters.whiteBalance - FILTER_DEFAULTS.whiteBalance) /
    FILTER_DIVIDERS.whiteBalance;
  const redAdjust =
    (filters.red - FILTER_DEFAULTS.red) / FILTER_DIVIDERS.colour;
  const greenAdjust =
    (filters.green - FILTER_DEFAULTS.green) / FILTER_DIVIDERS.colour;
  const blueAdjust =
    (filters.blue - FILTER_DEFAULTS.blue) / FILTER_DIVIDERS.colour;

  const redMultiplier = 1 + wbMatrixValue + redAdjust;
  const greenMultiplier = 1 + greenAdjust;
  const blueMultiplier = 1 - wbMatrixValue + blueAdjust;

  const gammaExponent = needsGamma
    ? 1 +
      (FILTER_DEFAULTS.gamma - filters.gamma) / FILTER_DIVIDERS.gamma
    : 1;

  for (let i = 0; i < data.length; i += 4) {
    let r = data[i];
    let g = data[i + 1];
    let b = data[i + 2];

    if (needsMatrix) {
      r *= redMultiplier;
      g *= greenMultiplier;
      b *= blueMultiplier;
    }

    if (needsGamma) {
      r = applyGamma(r, gammaExponent);
      g = applyGamma(g, gammaExponent);
      b = applyGamma(b, gammaExponent);
    }

    data[i] = clampChannel(r);
    data[i + 1] = clampChannel(g);
    data[i + 2] = clampChannel(b);
  }

  ctx.putImageData(imageData, 0, 0);
}

function clampChannel(value: number) {
  return Math.max(0, Math.min(255, Math.round(value)));
}

function applyGamma(value: number, exponent: number) {
  const normalized = Math.max(0, Math.min(1, value / 255));
  return 255 * Math.pow(normalized, exponent);
}

