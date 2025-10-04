import React, { useCallback, MouseEvent, useEffect, useState, useRef } from "react";
import * as GQL from "src/core/generated-graphql";
import { objectTitle } from "src/core/files";
import { WebDisplayMode } from "src/models/list-filter/types";
import { WebDisplayModeToggle } from "./WebDisplayModeToggle";
import { useIntersectionObserver } from "src/hooks/useIntersectionObserver";
import { PageNavigationInput } from "./PageNavigationInput";
import { useGalleryIncrementPlayCount } from "src/core/StashService";

interface IImageWebViewProps {
  images: GQL.SlimImageDataFragment[];
  onImageClick: (index: number) => void;
  webDisplayMode?: WebDisplayMode;
  onDisplayModeChange?: (mode: WebDisplayMode) => void;
  galleryId?: string;
}

export const ImageWebView: React.FC<IImageWebViewProps> = ({
  images,
  onImageClick,
  webDisplayMode = WebDisplayMode.FitToScreen,
  onDisplayModeChange,
  galleryId,
}) => {
  const [imageSizes, setImageSizes] = useState<{ [key: string]: { width: number; height: number } }>({});
  const [currentImageIndex, setCurrentImageIndex] = useState<number>(0);
  const imageRefs = useRef<(HTMLDivElement | null)[]>([]);
  const spacerRefs = useRef<(HTMLDivElement | null)[]>([]);
  const isModeSwitching = useRef<boolean>(false);
  const hasIncrementedPlayCount = useRef<boolean>(false);

  const [incrementPlayCount] = useGalleryIncrementPlayCount();

  const { observe, unobserve, activeIndex } = useIntersectionObserver({
    threshold: 0.5,
    rootMargin: '-40% 0px -40% 0px'
  });

  // Initialize currentImageIndex to 0 when images change
  useEffect(() => {
    if (images.length > 0) {
      setCurrentImageIndex(0);
      hasIncrementedPlayCount.current = false;
    }
  }, [images.length]);

  // Track gallery view when user reaches 3rd image
  useEffect(() => {
    if (galleryId && activeIndex >= 2 && !hasIncrementedPlayCount.current) {
      hasIncrementedPlayCount.current = true;
      incrementPlayCount({
        variables: {
          id: galleryId,
        },
      });
    }
  }, [activeIndex, galleryId, incrementPlayCount]);

  useEffect(() => {
    if (activeIndex >= 0 && activeIndex < images.length && !isModeSwitching.current) {
      setCurrentImageIndex(activeIndex);
    }
  }, [activeIndex, images.length]);

  useEffect(() => {
    imageRefs.current = imageRefs.current.slice(0, images.length);
    spacerRefs.current = spacerRefs.current.slice(0, images.length);
  }, [images.length]);

  useEffect(() => {
    spacerRefs.current.forEach((ref, index) => {
      if (ref) {
        observe(ref);
      }
    });

    return () => {
      spacerRefs.current.forEach((ref) => {
        if (ref) {
          unobserve(ref);
        }
      });
    };
  }, [images.length, observe, unobserve]);

  const handleImageClick = useCallback(
    (index: number) => (event: MouseEvent) => {
      onImageClick(index);
      event.preventDefault();
    },
    [onImageClick]
  );

  const handleImageLoad = useCallback((imageId: string, naturalWidth: number, naturalHeight: number) => {
    setImageSizes(prev => ({
      ...prev,
      [imageId]: { width: naturalWidth, height: naturalHeight }
    }));
  }, []);

  const scrollToImage = useCallback((index: number) => {
    if (index >= 0 && index < images.length) {
      // Find the spacer element with the matching data-index
      const spacerElement = document.querySelector(`[data-index="${index}"]`);
      if (spacerElement) {
        // Disable intersection observer temporarily
        isModeSwitching.current = true;
        
        spacerElement.scrollIntoView({
          behavior: 'instant',
          block: 'center'
        });
        
        // Force update currentImageIndex after scroll
        setTimeout(() => {
          setCurrentImageIndex(index);
          isModeSwitching.current = false;
        }, 100);
      }
    }
  }, [images.length]);


  // Sync scroll position when switching display modes
  useEffect(() => {
    isModeSwitching.current = true;
    
    const timeoutId = setTimeout(() => {
      if (currentImageIndex >= 0 && currentImageIndex < images.length) {
        const spacerElement = document.querySelector(`[data-index="${currentImageIndex}"]`);
        if (spacerElement) {
          spacerElement.scrollIntoView({
            behavior: 'instant',
            block: 'center'
          });
        }
      }
      isModeSwitching.current = false;
    }, 100);

    return () => {
      clearTimeout(timeoutId);
      isModeSwitching.current = false;
    };
  }, [webDisplayMode, images.length]);

  const [windowHeight, setWindowHeight] = useState(window.innerHeight);
  const [windowWidth, setWindowWidth] = useState(window.innerWidth);
  const [interfaceHeight, setInterfaceHeight] = useState(280);

  useEffect(() => {
    const handleResize = () => {
      setWindowHeight(window.innerHeight);
      setWindowWidth(window.innerWidth);
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  useEffect(() => {
    const calculateInterfaceHeight = () => {
      const header = document.querySelector('header') || document.querySelector('.navbar');
      const tabs = document.querySelector('.nav-tabs') || document.querySelector('[role="tablist"]');
      const toolbar = document.querySelector('.toolbar') || document.querySelector('.filtered-list-toolbar');
      const webControls = document.querySelector('.web-mode-controls');

      let totalHeight = 0;
      if (header) totalHeight += header.getBoundingClientRect().height;
      if (tabs) totalHeight += tabs.getBoundingClientRect().height;
      if (toolbar) totalHeight += toolbar.getBoundingClientRect().height;
      if (webControls) totalHeight += webControls.getBoundingClientRect().height;

      totalHeight += 50;

      setInterfaceHeight(Math.max(totalHeight, 200));
    };
    calculateInterfaceHeight();

    const handleResize = () => {
      calculateInterfaceHeight();
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const getImageStyle = useCallback((image: GQL.SlimImageDataFragment) => {
    if (webDisplayMode === WebDisplayMode.FitToScreen) {
      const imageSize = imageSizes[image.id];
      if (imageSize) {
        const availableHeight = windowHeight - interfaceHeight;
        const availableWidth = windowWidth - 40;

        // If image fits on screen, use its real dimensions
        if (imageSize.height <= availableHeight && imageSize.width <= availableWidth) {
          return {
            height: `${imageSize.height}px`,
            width: `${imageSize.width}px`,
            objectFit: 'contain' as const,
            display: 'block',
            margin: '0 auto'
          };
        }

        // Otherwise scale to fit screen
        const aspectRatio = imageSize.width / imageSize.height;

        const heightByHeight = availableHeight;
        const widthByHeight = heightByHeight * aspectRatio;

        const widthByWidth = availableWidth;
        const heightByWidth = widthByWidth / aspectRatio;

        let finalHeight, finalWidth;

        if (heightByWidth <= availableHeight) {
          finalWidth = widthByWidth;
          finalHeight = heightByWidth;
        } else {
          finalHeight = heightByHeight;
          finalWidth = widthByHeight;
        }

        return {
          height: `${finalHeight}px`,
          width: `${finalWidth}px`,
          objectFit: 'contain' as const,
          display: 'block',
          margin: '0 auto'
        };
      }
    }
    return {};
  }, [webDisplayMode, imageSizes, windowHeight, windowWidth, interfaceHeight]);

  if (!images.length) {
    return (
      <div className="image-web-container">
        <div className="image-web-empty">
          <p>Нет изображений для отображения</p>
        </div>
      </div>
    );
  }

  const containerClass = webDisplayMode === WebDisplayMode.FitToScreen
    ? "image-web-container image-web-container-fit image-web-container-sticky"
    : "image-web-container image-web-container-sticky";

  const imageClass = webDisplayMode === WebDisplayMode.FitToScreen
    ? "image-web-image image-web-image-fit"
    : "image-web-image";

  return (
    <div className={containerClass}>
      <div className="image-web-sticky-controls">
        <PageNavigationInput
          currentPage={currentImageIndex + 1}
          totalPages={images.length}
          onPageChange={scrollToImage}
          className="image-web-page-navigation"
        />

        {onDisplayModeChange && (
          <div className="web-mode-controls">
            <WebDisplayModeToggle
              currentMode={webDisplayMode}
              onModeChange={onDisplayModeChange}
            />
          </div>
        )}
      </div>

      <div className="image-web-list">
        {images.map((image, index) => (
          <React.Fragment key={image.id}>
            <div
              className="image-web-item"
              ref={(el) => {
                imageRefs.current[index] = el;
              }}
              style={{ position: 'relative' }}
            >
              <div
                className="image-web-spacer"
                data-index={index}
                ref={(el) => {
                  spacerRefs.current[index] = el;
                }}
                style={{
                  height: '1px',
                  width: '100%',
                  position: 'absolute',
                  top: '50%',
                  left: 0,
                  pointerEvents: 'none',
                  visibility: 'hidden',
                  zIndex: 1
                }}
              />
              <img
                src={image.paths.image || image.paths.preview || image.paths.thumbnail || ""}
                alt={objectTitle(image)}
                className={imageClass}
                style={getImageStyle(image)}
                onClick={handleImageClick(index)}
                onLoad={(e) => {
                  const img = e.target as HTMLImageElement;
                  handleImageLoad(image.id, img.naturalWidth, img.naturalHeight);
                }}
                loading="lazy"
              />
              {image.title && (
                <div className="image-web-title">
                  {image.title}
                </div>
              )}
            </div>
          </React.Fragment>
        ))}
      </div>
    </div>
  );
};
