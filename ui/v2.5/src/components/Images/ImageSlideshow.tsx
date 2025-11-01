import React, { useState, useEffect, useCallback, useRef } from "react";
import { Button } from "react-bootstrap";
import { Icon } from "src/components/Shared/Icon";
import {
  faPlay,
  faPause,
  faChevronLeft,
  faChevronRight,
  faExpand,
} from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { objectTitle } from "src/core/files";
import Mousetrap from "mousetrap";
import { PageNavigationInput } from "./PageNavigationInput";

interface IImageSlideshowProps {
  images: GQL.SlimImageDataFragment[];
  onImageClick: (index: number) => void;
  autoPlay?: boolean;
  autoPlayInterval?: number;
}

export const ImageSlideshow: React.FC<IImageSlideshowProps> = ({
  images,
  onImageClick,
  autoPlay = false,
  autoPlayInterval = 3000,
}) => {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(autoPlay);
  const containerRef = useRef<HTMLDivElement>(null);

  const currentImage = images[currentIndex];

  const goToPrevious = useCallback(() => {
    setCurrentIndex((prev) => (prev === 0 ? images.length - 1 : prev - 1));
  }, [images.length]);

  const goToNext = useCallback(() => {
    setCurrentIndex((prev) => (prev === images.length - 1 ? 0 : prev + 1));
  }, [images.length]);

  const goToSlide = useCallback(
    (index: number) => {
      if (index >= 0 && index < images.length) {
        setCurrentIndex(index);
      }
    },
    [images.length]
  );

  const togglePlayPause = useCallback(() => {
    setIsPlaying((prev) => !prev);
  }, []);

  const handleImageClick = useCallback(() => {
    onImageClick(currentIndex);
  }, [currentIndex, onImageClick]);

  // Auto-play functionality
  useEffect(() => {
    if (isPlaying && images.length > 1) {
      const interval = setInterval(goToNext, autoPlayInterval);
      return () => clearInterval(interval);
    }
  }, [isPlaying, goToNext, autoPlayInterval, images.length]);

  // Auto-focus on mount
  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.focus();
    }
  }, []);

  // Keyboard navigation
  useEffect(() => {
    Mousetrap.bind("left", goToPrevious);
    Mousetrap.bind("right", goToNext);
    Mousetrap.bind("space", (e) => {
      e.preventDefault();
      togglePlayPause();
    });

    return () => {
      Mousetrap.unbind("left");
      Mousetrap.unbind("right");
      Mousetrap.unbind("space");
    };
  }, [goToPrevious, goToNext, togglePlayPause]);

  if (!images.length) {
    return (
      <div className="image-slideshow-container">
        <div className="image-slideshow-empty">
          <p>Нет изображений для отображения</p>
        </div>
      </div>
    );
  }

  return (
    <div className="image-slideshow-container" tabIndex={0} ref={containerRef}>
      <div className="image-slideshow-main">
        <div className="image-slideshow-image-container">
          <img
            src={
              currentImage.paths.image ||
              currentImage.paths.preview ||
              currentImage.paths.thumbnail ||
              ""
            }
            alt={objectTitle(currentImage)}
            className="image-slideshow-image"
          />
        </div>

        <div className="image-slideshow-controls">
          <Button
            variant="secondary"
            size="sm"
            onClick={goToPrevious}
            disabled={images.length <= 1}
            className="image-slideshow-control-btn"
            title="Предыдущее изображение (←)"
          >
            <Icon icon={faChevronLeft} />
          </Button>

          <Button
            variant="secondary"
            size="sm"
            onClick={togglePlayPause}
            disabled={images.length <= 1}
            className="image-slideshow-control-btn"
            title={
              isPlaying
                ? "Остановить автовоспроизведение"
                : "Запустить автовоспроизведение"
            }
          >
            <Icon icon={isPlaying ? faPause : faPlay} />
          </Button>

          <Button
            variant="secondary"
            size="sm"
            onClick={goToNext}
            disabled={images.length <= 1}
            className="image-slideshow-control-btn"
            title="Следующее изображение (→)"
          >
            <Icon icon={faChevronRight} />
          </Button>

          <Button
            variant="secondary"
            size="sm"
            onClick={handleImageClick}
            className="image-slideshow-control-btn"
            title="Открыть в полноэкранном режиме"
          >
            <Icon icon={faExpand} />
          </Button>
        </div>

        <PageNavigationInput
          currentPage={currentIndex}
          totalPages={images.length}
          onPageChange={goToSlide}
          className="image-slideshow-page-navigation"
        />

        {currentImage.title && (
          <div className="image-slideshow-info">
            <span className="image-slideshow-title">{currentImage.title}</span>
          </div>
        )}
      </div>
    </div>
  );
};
