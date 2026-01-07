import React from "react";
import * as GQL from "src/core/generated-graphql";

export const FILTER_DEFAULTS = {
  brightness: 100,
  contrast: 100,
  gamma: 100,
  saturate: 100,
  hueRotate: 0,
  whiteBalance: 100,
  red: 100,
  green: 100,
  blue: 100,
  blur: 0,
};

export const FILTER_DIVIDERS = {
  gamma: 200,
  whiteBalance: 200,
  colour: 100,
  blur: 10,
};

export const SCALE_DEFAULT = 100;
export const ASPECT_RATIO_DEFAULT = 150;
export const ASPECT_RATIO_DIVIDER = 100;
export const ROTATE_DEFAULT = 2;
export const ROTATE_DIVIDER = 1 / 90;

type MaybeFilters = GQL.Maybe<GQL.VideoFilters>;
type MaybeTransforms = GQL.Maybe<GQL.VideoTransforms>;

export function needsColorMatrix(filters?: MaybeFilters) {
  if (!filters) {
    return false;
  }

  const whiteBalance = filters.white_balance ?? FILTER_DEFAULTS.whiteBalance;
  const red = filters.red ?? FILTER_DEFAULTS.red;
  const green = filters.green ?? FILTER_DEFAULTS.green;
  const blue = filters.blue ?? FILTER_DEFAULTS.blue;

  return (
    whiteBalance !== FILTER_DEFAULTS.whiteBalance ||
    red !== FILTER_DEFAULTS.red ||
    green !== FILTER_DEFAULTS.green ||
    blue !== FILTER_DEFAULTS.blue
  );
}

export function needsGammaAdjustment(filters?: MaybeFilters) {
  if (!filters) {
    return false;
  }

  const gamma = filters.gamma ?? FILTER_DEFAULTS.gamma;
  return gamma !== FILTER_DEFAULTS.gamma;
}

export function buildSvgFilter(
  filters: MaybeFilters,
  filterId: string
): React.ReactNode {
  if (!filters) {
    return null;
  }

  const needsMatrix = needsColorMatrix(filters);
  const needsGamma = needsGammaAdjustment(filters);

  if (!needsMatrix && !needsGamma) {
    return null;
  }

  const whiteBalance = filters.white_balance ?? FILTER_DEFAULTS.whiteBalance;
  const red = filters.red ?? FILTER_DEFAULTS.red;
  const green = filters.green ?? FILTER_DEFAULTS.green;
  const blue = filters.blue ?? FILTER_DEFAULTS.blue;
  const gamma = filters.gamma ?? FILTER_DEFAULTS.gamma;

  const wbMatrixValue =
    (whiteBalance - FILTER_DEFAULTS.whiteBalance) /
    FILTER_DIVIDERS.whiteBalance;
  const redAdjust = (red - FILTER_DEFAULTS.red) / FILTER_DIVIDERS.colour;
  const greenAdjust = (green - FILTER_DEFAULTS.green) / FILTER_DIVIDERS.colour;
  const blueAdjust = (blue - FILTER_DEFAULTS.blue) / FILTER_DIVIDERS.colour;

  const redMultiplier = 1 + wbMatrixValue + redAdjust;
  const greenMultiplier = 1 + greenAdjust;
  const blueMultiplier = 1 - wbMatrixValue + blueAdjust;

  const gammaExponent =
    1 + (FILTER_DEFAULTS.gamma - gamma) / FILTER_DIVIDERS.gamma;

  return (
    <svg
      className="video-filter-defs"
      style={{ position: "absolute", width: 0, height: 0 }}
      aria-hidden
      focusable="false"
    >
      <filter id={filterId}>
        {needsMatrix && (
          <feColorMatrix
            values={`${redMultiplier} 0 0 0 0   0 ${greenMultiplier} 0 0 0   0 0 ${blueMultiplier} 0 0   0 0 0 1 0`}
          />
        )}
        {needsGamma && (
          <feComponentTransfer>
            <feFuncR
              type="gamma"
              amplitude="1"
              exponent={gammaExponent}
              offset="0"
            />
            <feFuncG
              type="gamma"
              amplitude="1"
              exponent={gammaExponent}
              offset="0"
            />
            <feFuncB
              type="gamma"
              amplitude="1"
              exponent={gammaExponent}
              offset="0"
            />
            <feFuncA type="gamma" amplitude="1" exponent="1" offset="0" />
          </feComponentTransfer>
        )}
      </filter>
    </svg>
  );
}

function formatNumber(value: number) {
  return Number.parseFloat(value.toFixed(4));
}

export function buildCssFilterString(
  filters?: MaybeFilters,
  svgFilterId?: string
) {
  if (!filters && !svgFilterId) {
    return undefined;
  }

  const parts: string[] = [];

  if (svgFilterId) {
    parts.push(`url(#${svgFilterId})`);
  }

  const contrast = filters?.contrast ?? FILTER_DEFAULTS.contrast;
  if (contrast !== FILTER_DEFAULTS.contrast) {
    parts.push(`contrast(${contrast}%)`);
  }

  const brightness = filters?.brightness ?? FILTER_DEFAULTS.brightness;
  if (brightness !== FILTER_DEFAULTS.brightness) {
    parts.push(`brightness(${brightness}%)`);
  }

  const saturate = filters?.saturate ?? FILTER_DEFAULTS.saturate;
  if (saturate !== FILTER_DEFAULTS.saturate) {
    parts.push(`saturate(${saturate}%)`);
  }

  const hueRotate = filters?.hue_rotate ?? FILTER_DEFAULTS.hueRotate;
  if (hueRotate !== FILTER_DEFAULTS.hueRotate) {
    parts.push(`hue-rotate(${hueRotate}deg)`);
  }

  const blur = filters?.blur ?? FILTER_DEFAULTS.blur;
  if (blur > FILTER_DEFAULTS.blur) {
    parts.push(`blur(${blur / FILTER_DIVIDERS.blur}px)`);
  }

  if (!parts.length) {
    return undefined;
  }

  return parts.join(" ");
}

export function buildTransformString(transforms?: MaybeTransforms) {
  if (!transforms) {
    return undefined;
  }

  const rotateValue = transforms.rotate ?? ROTATE_DEFAULT;
  const scaleValue = transforms.scale ?? SCALE_DEFAULT;
  const aspectRatio = transforms.aspect_ratio ?? ASPECT_RATIO_DEFAULT;

  const parts: string[] = [];

  if (rotateValue !== ROTATE_DEFAULT) {
    const degrees = (rotateValue - ROTATE_DEFAULT) / ROTATE_DIVIDER;
    parts.push(`rotate(${degrees}deg)`);
  }

  let xScale = scaleValue / SCALE_DEFAULT;
  let yScale = scaleValue / SCALE_DEFAULT;

  if (aspectRatio > ASPECT_RATIO_DEFAULT) {
    xScale *=
      (ASPECT_RATIO_DIVIDER + (aspectRatio - ASPECT_RATIO_DEFAULT)) /
      ASPECT_RATIO_DIVIDER;
  } else if (aspectRatio < ASPECT_RATIO_DEFAULT) {
    yScale *=
      (ASPECT_RATIO_DIVIDER + (ASPECT_RATIO_DEFAULT - aspectRatio)) /
      ASPECT_RATIO_DIVIDER;
  }

  xScale = formatNumber(xScale);
  yScale = formatNumber(yScale);

  if (xScale !== 1 || yScale !== 1) {
    parts.push(`scale(${xScale}, ${yScale})`);
  }

  if (!parts.length) {
    return undefined;
  }

  return parts.join(" ");
}

export function getFilterStyle(filters?: MaybeFilters, svgFilterId?: string) {
  const filter = buildCssFilterString(filters, svgFilterId);
  if (!filter) {
    return undefined;
  }
  return { filter };
}

export function getFilterTransformStyle(
  filters?: MaybeFilters,
  transforms?: MaybeTransforms,
  svgFilterId?: string
) {
  const filterStyle = getFilterStyle(filters, svgFilterId);
  const transform = buildTransformString(transforms);

  if (!filterStyle && !transform) {
    return undefined;
  }

  return {
    ...(filterStyle ?? {}),
    ...(transform ? { transform, transformOrigin: "center center" } : {}),
  };
}
