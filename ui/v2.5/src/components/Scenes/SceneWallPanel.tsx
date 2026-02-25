import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { faStar, faTags } from "@fortawesome/free-solid-svg-icons";
import { Icon } from "../Shared/Icon";
import * as GQL from "src/core/generated-graphql";
import { SceneQueue } from "src/models/sceneQueue";
import Gallery, {
  GalleryI,
  PhotoProps,
  RenderImageProps,
} from "react-photo-gallery";
import { ConfigurationContext } from "src/hooks/Config";
import { objectTitle } from "src/core/files";
import { Link } from "react-router-dom";
import { TruncatedText } from "../Shared/TruncatedText";
import TextUtils from "src/utils/text";
import {
  convertToRatingFormat,
  defaultRatingSystemOptions,
} from "src/utils/rating";
import { useIntl } from "react-intl";
import cx from "classnames";
import { PreviewScrubber } from "./PreviewScrubber";
import { PerformerPopover } from "../Performers/PerformerPopover";
import { SweatDrops } from "../Shared/SweatDrops";
import { OMGIcon } from "../Shared/OMGIcon";
import { Form } from "react-bootstrap";

interface IScenePhoto {
  scene: GQL.SlimSceneDataFragment;
  link: string;
  onError?: (photo: PhotoProps<IScenePhoto>) => void;
  /** When animate previews disabled: static image URL (screenshot) */
  imageSrc?: string;
  /** When animate previews disabled: video/animation URL (preview) for hover */
  videoSrc?: string;
  /** Selection state and handler for wall selection */
  selected?: boolean;
  onSelectChange?: (selected: boolean, shiftKey: boolean) => void;
}

export const SceneWallItem: React.FC<RenderImageProps<IScenePhoto>> = (
  props: RenderImageProps<IScenePhoto>
) => {
  const intl = useIntl();

  const { configuration } = useContext(ConfigurationContext);
  const playSound = configuration?.interface.soundOnPreview ?? false;
  const showTitle = configuration?.interface.wallShowTitle ?? false;
  const animatePreviews = configuration?.interface.wallAnimatePreviews ?? true;
  const showAdditionalInfo = configuration?.interface.wallShowAdditionalInfo ?? true;
  const ratingSystemOptions =
    configuration?.ui.ratingSystemOptions ?? defaultRatingSystemOptions;

  const [active, setActive] = useState(false);
  const [layoutHidden, setLayoutHidden] = useState(false);
  const videoRef = useRef<HTMLVideoElement>(null);
  const hideLayoutTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cancelHideLayout = useCallback(() => {
    if (hideLayoutTimerRef.current) {
      clearTimeout(hideLayoutTimerRef.current);
      hideLayoutTimerRef.current = null;
    }
    setLayoutHidden(false);
  }, []);

  const startHideLayoutTimer = useCallback(() => {
    if (!showAdditionalInfo) return;
    if (hideLayoutTimerRef.current) clearTimeout(hideLayoutTimerRef.current);
    hideLayoutTimerRef.current = setTimeout(() => setLayoutHidden(true), 1000);
  }, [showAdditionalInfo]);

  const onCardMouseLeave = useCallback(() => {
    if (hideLayoutTimerRef.current) {
      clearTimeout(hideLayoutTimerRef.current);
      hideLayoutTimerRef.current = null;
    }
    setLayoutHidden(false);
  }, []);

  useEffect(() => {
    if (active && videoRef.current) {
      videoRef.current.play().catch(() => {});
    }
  }, [active]);

  useEffect(() => {
    return () => {
      if (hideLayoutTimerRef.current) clearTimeout(hideLayoutTimerRef.current);
    };
  }, []);

  type style = Record<string, string | number | undefined>;
  var divStyle: style = {
    margin: props.margin,
    display: "block",
  };

  if (props.direction === "column") {
    divStyle.position = "absolute";
    divStyle.left = props.left;
    divStyle.top = props.top;
  }

  const openInNewTab = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      window.open(props.photo.link, "_blank", "noopener,noreferrer");
    },
    [props.photo.link]
  );

  const { scene, imageSrc, videoSrc, selected, onSelectChange } = props.photo;
  const useHoverOnly =
    !animatePreviews && imageSrc && videoSrc;
  const video =
    useHoverOnly ? active : props.photo.src.includes("preview");
  const ImagePreview = video ? "video" : "img";
  const mediaSrc = useHoverOnly
    ? active
      ? videoSrc
      : imageSrc
    : props.photo.src;

  const title = objectTitle(scene);

  const tagsCount = scene.tags?.length ?? 0;
  const oCount = scene.o_counter ?? 0;
  const omgCount = scene.omgCounter ?? 0;
  const displayRating = convertToRatingFormat(
    scene.rating100 ?? undefined,
    ratingSystemOptions
  );
  const hasRating = scene.rating100 != null;

  return (
    <div
      className={cx("wall-item", {
        "show-title": showTitle,
        "wall-item-layout-hidden": layoutHidden,
        "wall-item-selected": selected,
      })}
      role="button"
      style={{
        ...divStyle,
        width: props.photo.width,
        height: props.photo.height,
      }}
      onClick={openInNewTab}
      onMouseEnter={startHideLayoutTimer}
      onMouseLeave={onCardMouseLeave}
    >
      {scene.paths.vtt && (
        <div className="wall-item-scrubber-zone">
          <PreviewScrubber
            vttPath={scene.paths.vtt ?? undefined}
            sceneId={scene.id}
            filters={scene.video_filters}
            transforms={scene.video_transforms}
            onProtectedHover={cancelHideLayout}
            onProtectedLeave={startHideLayoutTimer}
          />
        </div>
      )}
      {useHoverOnly ? (
        <>
          <img
            loading="lazy"
            key={`${props.photo.key}-img`}
            src={imageSrc}
            width={props.photo.width}
            height={props.photo.height}
            alt={props.photo.alt}
            onMouseEnter={() => setActive(true)}
            onMouseLeave={() => setActive(false)}
            onError={() => props.photo.onError?.(props.photo)}
            style={{ display: active ? "none" : "block" }}
          />
          <video
            ref={videoRef}
            loop
            playsInline
            preload="auto"
            muted={!playSound || !active}
            key={`${props.photo.key}-video`}
            src={videoSrc}
            width={props.photo.width}
            height={props.photo.height}
            onMouseEnter={() => setActive(true)}
            onMouseLeave={() => setActive(false)}
            onError={() => props.photo.onError?.(props.photo)}
            style={{ display: active ? "block" : "none" }}
          />
        </>
      ) : (
        <ImagePreview
          loading="lazy"
          loop={video}
          muted={!video || !playSound || !active}
          autoPlay={video}
          preload={video ? "auto" : undefined}
          key={props.photo.key}
          src={mediaSrc}
          width={props.photo.width}
          height={props.photo.height}
          alt={props.photo.alt}
          onMouseEnter={() => setActive(true)}
          onMouseLeave={() => setActive(false)}
          onError={() => {
            props.photo.onError?.(props.photo);
          }}
        />
      )}
      {showAdditionalInfo && hasRating && (
        <div
          className="wall-item-additional-rating"
          onMouseEnter={cancelHideLayout}
          onMouseLeave={startHideLayoutTimer}
        >
          <Icon icon={faStar} className="wall-item-rating-icon" />
          <span>{displayRating ?? 0}</span>
        </div>
      )}
      {showAdditionalInfo && (tagsCount > 0 || oCount > 0 || omgCount > 0) && (
        <div
          className="wall-item-additional-counts"
          onMouseEnter={cancelHideLayout}
          onMouseLeave={startHideLayoutTimer}
        >
          {tagsCount > 0 && (
            <span className="wall-item-count" title={intl.formatMessage({ id: "tags" })}>
              <span className="wall-item-count-icon">
                <Icon icon={faTags} />
              </span>
              {tagsCount}
            </span>
          )}
          {oCount > 0 && (
            <span className="wall-item-count" title={intl.formatMessage({ id: "o_counter" })}>
              <span className="wall-item-count-icon">
                <SweatDrops />
              </span>
              {oCount}
            </span>
          )}
          {omgCount > 0 && (
            <span className="wall-item-count" title="OMG">
              <span className="wall-item-count-icon">
                <OMGIcon />
              </span>
              {omgCount}
            </span>
          )}
        </div>
      )}
      <div
        className="lineargradient wall-item-footer-zone"
        onMouseEnter={cancelHideLayout}
        onMouseLeave={startHideLayoutTimer}
      >
        <footer className="wall-item-footer">
          <Link
            to={props.photo.link}
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
          >
            {title && (
              <TruncatedText
                text={title}
                lineCount={1}
                className="wall-item-title"
              />
            )}
            {scene.performers.length > 0 && (
              <span className="wall-item-performers">
                {scene.performers.map((performer, index) => (
                  <React.Fragment key={performer.id}>
                    {index > 0 && ", "}
                    <PerformerPopover id={performer.id} placement="top">
                      <Link
                        to={`/performers/${performer.id}`}
                        onClick={(e) => e.stopPropagation()}
                        className="wall-item-performer-link"
                      >
                        {performer.name}
                      </Link>
                    </PerformerPopover>
                  </React.Fragment>
                ))}
              </span>
            )}
            <div>{scene.date && TextUtils.formatDate(intl, scene.date, true, scene.date_display)}</div>
          </Link>
        </footer>
        {onSelectChange && (
          <Form.Check
            key={`select-${scene.id}-${selected ?? false}`}
            type="checkbox"
            className="wall-item-select-check"
            checked={selected ?? false}
            onClick={(e) => {
              e.stopPropagation();
              onSelectChange(!(selected ?? false), e.shiftKey);
            }}
          />
        )}
      </div>
    </div>
  );
};

function getDimensions(s: GQL.SlimSceneDataFragment) {
  const defaults = { width: 1280, height: 720 };

  if (!s.files.length) return defaults;

  return {
    width: s.files[0].width || defaults.width,
    height: s.files[0].height || defaults.height,
  };
}

interface ISceneWallProps {
  scenes: GQL.SlimSceneDataFragment[];
  sceneQueue?: SceneQueue;
  zoomIndex: number;
  selectedIds?: Set<string>;
  onSelectChange?: (id: string, selected: boolean, shiftKey: boolean) => void;
}

// HACK: typescript doesn't allow Gallery to accept a parameter for some reason
const SceneGallery = Gallery as unknown as GalleryI<IScenePhoto>;

const breakpointZoomHeights = [
  { minWidth: 576, heights: [100, 120, 240, 360] },
  { minWidth: 768, heights: [120, 160, 240, 480] },
  { minWidth: 1200, heights: [120, 160, 240, 300] },
  { minWidth: 1400, heights: [160, 240, 300, 480] },
];

const SceneWall: React.FC<ISceneWallProps> = ({
  scenes,
  sceneQueue,
  zoomIndex,
  selectedIds,
  onSelectChange,
}) => {
  const margin = 3;
  const direction = "row";

  const [erroredImgs, setErroredImgs] = useState<string[]>([]);

  const handleError = useCallback((photo: PhotoProps<IScenePhoto>) => {
    setErroredImgs((prev) => [...prev, photo.src]);
  }, []);

  useEffect(() => {
    setErroredImgs([]);
  }, [scenes]);

  const photos: PhotoProps<IScenePhoto>[] = useMemo(() => {
    return scenes.map((s, index) => {
      const { width, height } = getDimensions(s);
      const previewOk =
        s.paths.preview && !erroredImgs.includes(s.paths.preview);
      const src = previewOk ? s.paths.preview! : s.paths.screenshot!;

      return {
        scene: s,
        src,
        imageSrc: s.paths.screenshot ?? undefined,
        videoSrc: previewOk ? s.paths.preview ?? undefined : undefined,
        link: sceneQueue
          ? sceneQueue.makeLink(s.id, { sceneIndex: index })
          : `/scenes/${s.id}`,
        width,
        height,
        tabIndex: index,
        key: s.id,
        loading: "lazy",
        alt: objectTitle(s),
        onError: handleError,
        selected: selectedIds?.has(s.id),
        onSelectChange: onSelectChange
          ? (selected: boolean, shiftKey: boolean) =>
              onSelectChange(s.id, selected, shiftKey)
          : undefined,
      };
    });
  }, [scenes, sceneQueue, erroredImgs, handleError, selectedIds, onSelectChange]);

  const onClick = useCallback(
    (event: React.MouseEvent, { index }: { index: number }) => {
      event.preventDefault();
      event.stopPropagation();
      window.open(photos[index].link, "_blank", "noopener,noreferrer");
    },
    [photos]
  );

  function columns(containerWidth: number) {
    let preferredSize = 300;
    let columnCount = containerWidth / preferredSize;
    return Math.round(columnCount);
  }

  function targetRowHeight(containerWidth: number) {
    let zoomHeight = 280;
    breakpointZoomHeights.forEach((e) => {
      if (containerWidth >= e.minWidth) {
        zoomHeight = e.heights[zoomIndex];
      }
    });
    return zoomHeight;
  }

  const renderImage = useCallback((props: RenderImageProps<IScenePhoto>) => {
    return <SceneWallItem {...props} />;
  }, []);

  return (
    <div className="scene-wall">
      {photos.length ? (
        <SceneGallery
          photos={photos}
          renderImage={renderImage}
          onClick={onClick}
          margin={margin}
          direction={direction}
          columns={columns}
          targetRowHeight={targetRowHeight}
        />
      ) : null}
    </div>
  );
};

interface ISceneWallPanelProps {
  scenes: GQL.SlimSceneDataFragment[];
  sceneQueue?: SceneQueue;
  zoomIndex: number;
  selectedIds?: Set<string>;
  onSelectChange?: (id: string, selected: boolean, shiftKey: boolean) => void;
}

export const SceneWallPanel: React.FC<ISceneWallPanelProps> = ({
  scenes,
  sceneQueue,
  zoomIndex,
  selectedIds,
  onSelectChange,
}) => {
  return (
    <SceneWall
      scenes={scenes}
      sceneQueue={sceneQueue}
      zoomIndex={zoomIndex}
      selectedIds={selectedIds}
      onSelectChange={onSelectChange}
    />
  );
};
