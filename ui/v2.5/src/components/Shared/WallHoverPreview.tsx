import React, { useCallback, useEffect, useRef, useState } from "react";

interface IWallHoverPreviewProps {
  imageSrc: string;
  videoSrc?: string;
  /** Animated WebP shown when animatePreviews is enabled */
  animationSrc?: string;
  animatePreviews: boolean;
  playSound: boolean;
  width: number;
  height: number;
  alt: string;
  onImageError?: () => void;
  onVideoError?: () => void;
  onClick?: (event: React.MouseEvent) => void;
}

function releaseVideo(video: HTMLVideoElement) {
  video.pause();
  video.removeAttribute("src");
  video.load();
}

export const WallHoverPreview: React.FC<IWallHoverPreviewProps> = ({
  imageSrc,
  videoSrc,
  animationSrc,
  animatePreviews,
  playSound,
  width,
  height,
  alt,
  onImageError,
  onVideoError,
  onClick,
}) => {
  const [hovered, setHovered] = useState(false);
  const [animationFailed, setAnimationFailed] = useState(false);
  const videoRef = useRef<HTMLVideoElement>(null);

  const idleSrc =
    animatePreviews && animationSrc && !animationFailed
      ? animationSrc
      : imageSrc;
  const showVideo = hovered && !!videoSrc;

  const onMouseEnter = useCallback(() => setHovered(true), []);
  const onMouseLeave = useCallback(() => setHovered(false), []);

  const handleImageError = useCallback(() => {
    if (animatePreviews && animationSrc && !animationFailed) {
      setAnimationFailed(true);
      return;
    }
    onImageError?.();
  }, [animatePreviews, animationSrc, animationFailed, onImageError]);

  useEffect(() => {
    const video = videoRef.current;
    if (!showVideo || !video) return;

    video.muted = !playSound;
    video.play().catch(() => {});

    return () => releaseVideo(video);
  }, [showVideo, playSound, videoSrc]);

  return (
    <>
      <img
        loading="lazy"
        src={idleSrc}
        width={width}
        height={height}
        alt={alt}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        onClick={onClick}
        onError={handleImageError}
        style={{ display: showVideo ? "none" : "block" }}
      />
      {showVideo && (
        <video
          ref={videoRef}
          src={videoSrc}
          loop
          playsInline
          preload="auto"
          muted={!playSound}
          width={width}
          height={height}
          onMouseEnter={onMouseEnter}
          onMouseLeave={onMouseLeave}
          onClick={onClick}
          onError={onVideoError}
        />
      )}
    </>
  );
};
