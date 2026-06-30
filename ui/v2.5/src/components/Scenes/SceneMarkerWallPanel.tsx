import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import * as GQL from "src/core/generated-graphql";
import Gallery, {
  GalleryI,
  PhotoProps,
  RenderImageProps,
} from "react-photo-gallery";
import { ConfigurationContext } from "src/hooks/Config";
import { objectTitle } from "src/core/files";
import { Link, useHistory } from "react-router-dom";
import { TruncatedText } from "../Shared/TruncatedText";
import TextUtils from "src/utils/text";
import cx from "classnames";
import NavUtils from "src/utils/navigation";
import { markerTitle } from "src/core/markers";
import { WallHoverPreview } from "../Shared/WallHoverPreview";

function wallItemTitle(sceneMarker: GQL.SceneMarkerDataFragment) {
  const newTitle = markerTitle(sceneMarker);
  const seconds = TextUtils.formatTimestampRange(
    sceneMarker.seconds,
    sceneMarker.end_seconds ?? undefined
  );
  if (newTitle) {
    return `${newTitle} - ${seconds}`;
  } else {
    return seconds;
  }
}

interface IMarkerPhoto {
  marker: GQL.SceneMarkerDataFragment;
  link: string;
  onError?: (failedSrc: string) => void;
  imageSrc?: string;
  videoSrc?: string;
  animationSrc?: string;
}

export const MarkerWallItem: React.FC<RenderImageProps<IMarkerPhoto>> = (
  props: RenderImageProps<IMarkerPhoto>
) => {
  const { configuration } = useContext(ConfigurationContext);
  const playSound = configuration?.interface.soundOnPreview ?? false;
  const showTitle = configuration?.interface.wallShowTitle ?? false;
  const animatePreviews = configuration?.interface.wallAnimatePreviews ?? true;

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

  var handleClick = function handleClick(event: React.MouseEvent) {
    if (props.onClick) {
      props.onClick(event, { index: props.index });
    }
  };

  const { marker, imageSrc, videoSrc, animationSrc } = props.photo;

  const title = wallItemTitle(marker);
  const tagNames = marker.tags.map((p) => p.name);

  return (
    <div
      className={cx("wall-item", { "show-title": showTitle })}
      role="button"
      style={{
        ...divStyle,
        width: props.photo.width,
        height: props.photo.height,
      }}
    >
      {imageSrc && (
        <WallHoverPreview
          imageSrc={imageSrc}
          videoSrc={videoSrc}
          animationSrc={animationSrc}
          animatePreviews={animatePreviews}
          playSound={playSound}
          width={props.photo.width}
          height={props.photo.height}
          alt={props.photo.alt}
          onClick={handleClick}
          onImageError={() => imageSrc && props.photo.onError?.(imageSrc)}
          onVideoError={() => videoSrc && props.photo.onError?.(videoSrc)}
        />
      )}
      <div className="lineargradient">
        <footer className="wall-item-footer">
          <Link to={props.photo.link} onClick={(e) => e.stopPropagation()}>
            {title && (
              <TruncatedText
                text={title}
                lineCount={1}
                className="wall-item-title"
              />
            )}
            <TruncatedText text={tagNames.join(", ")} />
          </Link>
        </footer>
      </div>
    </div>
  );
};

interface IMarkerWallProps {
  markers: GQL.SceneMarkerDataFragment[];
  zoomIndex: number;
}

// HACK: typescript doesn't allow Gallery to accept a parameter for some reason
const MarkerGallery = Gallery as unknown as GalleryI<IMarkerPhoto>;

function getFirstValidSrc(srcSet: string[], invalidSrcSet: string[]) {
  if (!srcSet.length) {
    return "";
  }

  return (
    srcSet.find((src) => !invalidSrcSet.includes(src)) ??
    ([...srcSet].pop() as string)
  );
}

interface IFile {
  width: number;
  height: number;
}

function getDimensions(file?: IFile) {
  const defaults = { width: 1280, height: 720 };

  if (!file) return defaults;

  return {
    width: file.width || defaults.width,
    height: file.height || defaults.height,
  };
}

const breakpointZoomHeights = [
  { minWidth: 576, heights: [100, 120, 240, 360] },
  { minWidth: 768, heights: [120, 160, 240, 480] },
  { minWidth: 1200, heights: [120, 160, 240, 300] },
  { minWidth: 1400, heights: [160, 240, 300, 480] },
];

const MarkerWall: React.FC<IMarkerWallProps> = ({ markers, zoomIndex }) => {
  const history = useHistory();

  const margin = 3;
  const direction = "row";

  const [erroredImgs, setErroredImgs] = useState<string[]>([]);

  const handleError = useCallback((failedSrc: string) => {
    setErroredImgs((prev) => [...prev, failedSrc]);
  }, []);

  useEffect(() => {
    setErroredImgs([]);
  }, [markers]);

  const photos: PhotoProps<IMarkerPhoto>[] = useMemo(() => {
    return markers.map((m, index) => {
      const { width = 1280, height = 720 } = getDimensions(m.scene.files[0]);
      const videoSrc = getFirstValidSrc([m.stream, m.preview], erroredImgs);
      const screenshot = m.screenshot ?? "";

      return {
        marker: m,
        src: screenshot,
        imageSrc: screenshot || undefined,
        videoSrc: videoSrc || undefined,
        animationSrc: m.preview ?? undefined,
        link: NavUtils.makeSceneMarkerUrl(m),
        width,
        height,
        tabIndex: index,
        key: m.id,
        loading: "lazy",
        alt: objectTitle(m),
        onError: handleError,
      };
    });
  }, [markers, erroredImgs, handleError]);

  const onClick = useCallback(
    (event, { index }) => {
      history.push(photos[index].link);
    },
    [history, photos]
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

  const renderImage = useCallback((props: RenderImageProps<IMarkerPhoto>) => {
    return <MarkerWallItem {...props} />;
  }, []);

  return (
    <div className="marker-wall">
      {photos.length ? (
        <MarkerGallery
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

interface IMarkerWallPanelProps {
  markers: GQL.SceneMarkerDataFragment[];
  zoomIndex: number;
}

export const MarkerWallPanel: React.FC<IMarkerWallPanelProps> = ({
  markers,
  zoomIndex,
}) => {
  return <MarkerWall markers={markers} zoomIndex={zoomIndex} />;
};
