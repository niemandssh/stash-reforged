import React, {
  useRef,
  useMemo,
  useState,
  useLayoutEffect,
  useEffect,
} from "react";
import * as GQL from "src/core/generated-graphql";
import { useSpriteInfo } from "src/hooks/sprite";
import { useThrottle } from "src/hooks/throttle";
import { HoverScrubber } from "../Shared/HoverScrubber";
import {
  buildSvgFilter,
  getFilterTransformStyle,
  needsColorMatrix,
  needsGammaAdjustment,
} from "src/utils/videoFilters";

interface IScenePreviewProps {
  vttPath: string | undefined;
  onClick?: (timestamp: number) => void;
  sceneId?: string;
  filters?: GQL.Maybe<GQL.VideoFilters>;
  transforms?: GQL.Maybe<GQL.VideoTransforms>;
}

function scaleToFit(dimensions: { w: number; h: number }, bounds: DOMRect) {
  const rw = bounds.width / dimensions.w;
  const rh = bounds.height / dimensions.h;

  // for consistency, use max by default and min for portrait
  if (dimensions.w > dimensions.h) {
    return Math.max(rw, rh);
  }

  return Math.min(rw, rh);
}

const defaultSprites = 81; // 9x9 grid by default

export const PreviewScrubber: React.FC<IScenePreviewProps> = ({
  vttPath,
  onClick,
  sceneId,
  filters,
  transforms,
}) => {
  const imageParentRef = useRef<HTMLDivElement>(null);
  const [style, setStyle] = useState({});

  const [activeIndex, setActiveIndex] = useState<number>();

  const debounceSetActiveIndex = useThrottle(setActiveIndex, 50);

  // hold off on loading vtt until first mouse over
  const [hasLoaded, setHasLoaded] = useState(false);
  const spriteInfo = useSpriteInfo(hasLoaded ? vttPath : undefined);

  const sprite = useMemo(() => {
    if (!spriteInfo || activeIndex === undefined) {
      return undefined;
    }
    return spriteInfo[activeIndex];
  }, [activeIndex, spriteInfo]);

  // mark as loaded on the first hover
  useEffect(() => {
    if (activeIndex !== undefined) {
      setHasLoaded(true);
    }
  }, [activeIndex]);

  const uniqueId = useMemo(
    () =>
      `scene-scrubber-${sceneId ?? "unknown"}-${Math.random()
        .toString(36)
        .slice(2)}`,
    [sceneId]
  );
  const requiresSvg =
    needsColorMatrix(filters ?? null) || needsGammaAdjustment(filters ?? null);
  const svgFilterId = requiresSvg ? `${uniqueId}-svg` : undefined;
  const filterTransformStyle = useMemo(
    () =>
      getFilterTransformStyle(filters ?? null, transforms ?? null, svgFilterId),
    [filters, transforms, svgFilterId]
  );
  const svgFilter = useMemo(
    () => (svgFilterId ? buildSvgFilter(filters ?? null, svgFilterId) : null),
    [filters, svgFilterId]
  );

  useLayoutEffect(() => {
    const imageParent = imageParentRef.current;

    if (!sprite || !imageParent) {
      return setStyle(filterTransformStyle ?? {});
    }

    const clientRect = imageParent.getBoundingClientRect();
    const scale = scaleToFit(sprite, clientRect);

    // Combine transforms from filterTransformStyle with scale
    const transformParts: string[] = [];
    if (filterTransformStyle?.transform) {
      transformParts.push(filterTransformStyle.transform);
    }
    transformParts.push(`scale(${scale})`);

    setStyle({
      ...(filterTransformStyle ?? {}),
      backgroundPosition: `${-sprite.x}px ${-sprite.y}px`,
      backgroundImage: `url(${sprite.url})`,
      width: `${sprite.w}px`,
      height: `${sprite.h}px`,
      transform: transformParts.join(" "),
    });
  }, [sprite, filterTransformStyle]);

  function onScrubberClick(index: number) {
    if (!onClick || !spriteInfo) {
      return;
    }

    const s = spriteInfo[index];
    onClick(s.start);
  }

  if (spriteInfo === null || !vttPath) return null;

  return (
    <div className="preview-scrubber">
      {sprite && (
        <div className="scene-card-preview-image" ref={imageParentRef}>
          <div className="scrubber-image" style={style}></div>
        </div>
      )}
      <HoverScrubber
        totalSprites={spriteInfo?.length ?? defaultSprites}
        activeIndex={activeIndex}
        setActiveIndex={(i) => debounceSetActiveIndex(i)}
        onClick={onScrubberClick}
      />
      {svgFilter}
    </div>
  );
};
