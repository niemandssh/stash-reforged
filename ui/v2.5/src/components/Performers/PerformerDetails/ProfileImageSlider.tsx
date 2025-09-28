import React, { useState, useEffect, useMemo, useCallback, useRef } from "react";
import { useIntl } from "react-intl";
import { Icon } from "src/components/Shared/Icon";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { faStar, faChevronLeft, faChevronRight } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { ProfileImageCropper } from "./ProfileImageCropper";
import ImageUtils from "src/utils/image";
import { usePerformerProfileImageUpdate, usePerformerProfileImageDestroy, usePerformerProfileImageCreate } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";
import { useLightbox } from "src/hooks/Lightbox/hooks";
import "./ProfileImageSlider.scss";

interface IProfileImageSliderProps {
  profileImages: GQL.PerformerProfileImage[];
  isEditing: boolean;
  currentImageIndex?: number;
  onImageChange?: (index: number) => void;
  onDeleteImage?: (imageId: string, index: number) => void;
  onSetPrimary?: (imageId: string, index: number) => void;
  onImageUpdate?: () => Promise<void>;
  onAddImage?: () => void;
  performerId: number;
  isNew?: boolean;
  activeImage?: string | null;
  encodingImage?: boolean;
  setImage?: (image?: string | null) => void;
  setEncodingImage?: (loading: boolean) => void;
  onPerformerUpdate?: (updatedPerformer: Partial<any>) => void;
}

export const ProfileImageSlider: React.FC<IProfileImageSliderProps> = ({
  profileImages,
  isEditing,
  currentImageIndex = 0,
  onImageChange,
  onDeleteImage,
  onSetPrimary,
  onImageUpdate,
  onAddImage,
  performerId,
  isNew = false,
  activeImage,
  encodingImage = false,
  setImage,
  setEncodingImage,
  onPerformerUpdate,
}) => {
  // Allow rendering when no images for add image button

  const intl = useIntl();
  const Toast = useToast();
  const [activeIndex, setActiveIndex] = useState(currentImageIndex);
  const [touchStart, setTouchStart] = useState<number | null>(null);
  const [touchEnd, setTouchEnd] = useState<number | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [hasSwiped, setHasSwiped] = useState(false);
  const [isCropping, setIsCropping] = useState(false);

  const [updateProfileImage] = usePerformerProfileImageUpdate();
  const [destroyProfileImage] = usePerformerProfileImageDestroy();
  const [createProfileImage] = usePerformerProfileImageCreate();

  const prevCurrentImageIndex = useRef(currentImageIndex);

  useEffect(() => {
    if (currentImageIndex !== prevCurrentImageIndex.current) {
      setActiveIndex(currentImageIndex);
      prevCurrentImageIndex.current = currentImageIndex;
    }
  }, [currentImageIndex]);


  // Sort images: primary first, then by position
  const sortedImages = useMemo(() => {
    if (!profileImages || profileImages.length === 0) return [];
    return [...profileImages].sort((a, b) => {
      if (a.is_primary && !b.is_primary) return -1;
      if (!a.is_primary && b.is_primary) return 1;
      return (a.position || 0) - (b.position || 0);
    });
  }, [profileImages]);

  // Convert profile images to lightbox format
  const lightboxImages = useMemo(() => {
    return sortedImages.map(image => ({
      id: image.id,
      title: `Profile Image ${sortedImages.indexOf(image) + 1}`,
      paths: {
        image: image.image_path || "",
        thumbnail: image.image_path || "",
        preview: image.image_path || "",
      },
      visual_files: [{
        __typename: "ImageFile" as const,
        path: image.image_path || "",
        width: 1920, // Higher resolution for better lightbox display
        height: 1080, // Higher resolution for better lightbox display
      }],
    }));
  }, [sortedImages]);

  // Create stable lightbox state
  const lightboxState = useMemo(() => ({
    images: lightboxImages,
    showNavigation: false,
    hideGallery: true,
    hideRating: true,
    hideOCounter: true,
  }), [lightboxImages]);

  // Always call useLightbox hook - no conditional calls
  const showLightbox = useLightbox(lightboxState);

  const handleImageClick = useCallback((e: React.MouseEvent) => {
    // Don't open lightbox if we're dragging, if it's a touch event, if we just swiped, or if we're cropping
    if (isDragging || touchStart !== null || hasSwiped || isCropping) {
      return;
    }
    e.preventDefault();
    showLightbox({ initialIndex: activeIndex });
  }, [isDragging, touchStart, hasSwiped, isCropping, showLightbox, activeIndex]);

  const currentImage = sortedImages.length > 0 ? sortedImages[activeIndex] : null;
  const hasMultipleImages = sortedImages.length > 1;
  const hasNoImages = sortedImages.length === 0 && !activeImage;

  const goToPrevious = useCallback(() => {
    const newIndex = activeIndex > 0 ? activeIndex - 1 : sortedImages.length - 1;
    setActiveIndex(newIndex);
    onImageChange?.(newIndex);
  }, [activeIndex, sortedImages.length, onImageChange]);

  const goToNext = useCallback(() => {
    const newIndex = activeIndex < sortedImages.length - 1 ? activeIndex + 1 : 0;
    setActiveIndex(newIndex);
    onImageChange?.(newIndex);
  }, [activeIndex, sortedImages.length, onImageChange]);

  // Touch handlers for swipe
  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    // Don't handle swipe if cropping is active
    if (isCropping) return;

    if (hasMultipleImages) {
      e.preventDefault();
      setIsDragging(true);
      setTouchEnd(null);
      setTouchStart(e.targetTouches[0].clientX);
    }
  }, [isCropping, hasMultipleImages]);

  const handleTouchMove = useCallback((e: React.TouchEvent) => {
    // Don't handle swipe if cropping is active
    if (isCropping) return;

    if (isDragging && hasMultipleImages) {
      e.preventDefault();
      setTouchEnd(e.targetTouches[0].clientX);
    }
  }, [isCropping, isDragging, hasMultipleImages]);

  const handleTouchEnd = useCallback((e: React.TouchEvent) => {
    // Don't handle swipe if cropping is active
    if (isCropping) return;

    if (isDragging && hasMultipleImages) {
      e.preventDefault();
      setIsDragging(false);

      if (touchStart && touchEnd) {
        const distance = touchStart - touchEnd;
        const isLeftSwipe = distance > 50;
        const isRightSwipe = distance < -50;

        if (isLeftSwipe || isRightSwipe) {
          setHasSwiped(true);
          if (isLeftSwipe) {
            goToNext();
          } else if (isRightSwipe) {
            goToPrevious();
          }
          // Reset hasSwiped after a short delay
          setTimeout(() => setHasSwiped(false), 100);
        }
      }
    }

    setTouchStart(null);
    setTouchEnd(null);
  }, [isCropping, isDragging, hasMultipleImages, touchStart, touchEnd, goToNext, goToPrevious]);

  // Mouse handlers for desktop swipe simulation
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    // Don't handle swipe if cropping is active
    if (isCropping) return;

    if (hasMultipleImages) {
      e.preventDefault();
      setIsDragging(true);
      setTouchEnd(null);
      setTouchStart(e.clientX);
    }
  }, [isCropping, hasMultipleImages]);

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    // Don't handle swipe if cropping is active
    if (isCropping) return;

    if (isDragging && hasMultipleImages) {
      e.preventDefault();
      setTouchEnd(e.clientX);
    }
  }, [isCropping, isDragging, hasMultipleImages]);

  const handleMouseUp = useCallback((e: React.MouseEvent) => {
    // Don't handle swipe if cropping is active
    if (isCropping) return;

    if (isDragging && hasMultipleImages) {
      e.preventDefault();
      setIsDragging(false);

      if (touchStart && touchEnd) {
        const distance = touchStart - touchEnd;
        const isLeftSwipe = distance > 50;
        const isRightSwipe = distance < -50;

        if (isLeftSwipe || isRightSwipe) {
          setHasSwiped(true);
          if (isLeftSwipe) {
            goToNext();
          } else if (isRightSwipe) {
            goToPrevious();
          }
          // Reset hasSwiped after a short delay
          setTimeout(() => setHasSwiped(false), 100);
        }
      }
    }

    setTouchStart(null);
    setTouchEnd(null);

    // Small delay to prevent click after drag
    setTimeout(() => {
      setIsDragging(false);
    }, 100);
  }, [isCropping, isDragging, hasMultipleImages, touchStart, touchEnd, goToNext, goToPrevious]);

  const handleDeleteImage = async () => {
    if (!currentImage) return;

    const confirmDelete = window.confirm(
      intl.formatMessage({
        id: "dialogs.delete_confirm",
        defaultMessage: "Are you sure you want to delete this image?"
      })
    );

    if (!confirmDelete) return;

    try {
      await destroyProfileImage({
        variables: {
          input: { id: currentImage.id },
        },
      });

      Toast.success(
        intl.formatMessage({
          id: "toast.deleted_entity",
          defaultMessage: "Deleted {entityType}",
        }, { entityType: intl.formatMessage({ id: "image" }) })
      );

      // Adjust active index if needed
      if (activeIndex >= profileImages.length - 1 && activeIndex > 0) {
        setActiveIndex(activeIndex - 1);
      }
    } catch (error) {
      console.error("Error deleting profile image:", error);
      Toast.error(
        intl.formatMessage({
          id: "toast.delete_failed",
          defaultMessage: "Delete failed"
        })
      );
    }

    // Also call prop handler if provided
    if (onDeleteImage) {
      onDeleteImage(currentImage.id, activeIndex);
    }
  };

  const handleSetPrimary = async () => {
    if (!currentImage) return;

    try {
      // Set the selected image as primary
      await updateProfileImage({
        variables: {
          input: {
            id: currentImage.id,
            is_primary: true,
          },
        },
      });

      // Unset all other images as primary
      await Promise.all(
        profileImages
          .filter(img => img.id !== currentImage.id && img.is_primary)
          .map(img =>
            updateProfileImage({
              variables: {
                input: {
                  id: img.id,
                  is_primary: false,
                },
              },
            })
          )
      );

      Toast.success(
        intl.formatMessage({
          id: "toast.set_primary_success"
        })
      );

      // Update the current image in the sorted array
      const updatedImages = profileImages.map(img =>
        img.id === currentImage.id
          ? { ...img, is_primary: true }
          : { ...img, is_primary: false }
      );

      // Re-sort images with primary first
      const newSortedImages = [...updatedImages].sort((a, b) => {
        if (a.is_primary && !b.is_primary) return -1;
        if (!a.is_primary && b.is_primary) return 1;
        return (a.position || 0) - (b.position || 0);
      });

      // Find new index of the primary image
      const newIndex = newSortedImages.findIndex(img => img.id === currentImage.id);
      if (newIndex !== -1) {
        setActiveIndex(newIndex);
        onImageChange?.(newIndex);
      }
    } catch (error) {
      console.error("Error setting primary image:", error);
      Toast.error(
        intl.formatMessage({
          id: "toast.set_primary_failed",
          defaultMessage: "Failed to set as primary"
        })
      );
    }

    // Also call prop handler if provided
    if (onSetPrimary) {
      onSetPrimary(currentImage.id, activeIndex);
    }
  };

  const handleImageUpload = async (imageData: string | null) => {
    if (!imageData) {
      setImage?.(null);
      return;
    }

    // If this is a new performer, just set the image field for now
    if (isNew) {
      setImage?.(imageData);
      return;
    }

    try {
      setEncodingImage?.(true);

      // Create a new profile image
      const result = await createProfileImage({
        variables: {
          input: {
            performer_id: performerId.toString(),
            image: imageData,
            is_primary: profileImages.length === 0, // First image becomes primary
            position: profileImages.length,
          },
        },
      });

      if (result.data?.performerProfileImageCreate) {
        const newImageIndex = profileImages.length + 1;
        Toast.success(
          intl.formatMessage(
            { id: "toast.created_entity" },
            {
              entity: `${intl.formatMessage({ id: "image" }).toLocaleLowerCase()} ${newImageIndex}`,
            }
          )
        );

        // Update performer with new profile image
        const newProfileImage = result.data.performerProfileImageCreate;
        const updatedProfileImages = [...profileImages, newProfileImage];
        const updatedPerformer = {
          profile_images: updatedProfileImages,
          primary_image_path: newProfileImage.is_primary ? newProfileImage.image_path : undefined,
        };
        onPerformerUpdate?.(updatedPerformer);

        // Also set the legacy image field for backward compatibility
        setImage?.(imageData);

        // Trigger image update to refresh the UI and wait for completion
        await onImageUpdate?.();

        // Switch to the newly uploaded image after data update
        const uploadedImageIndex = profileImages.length; // New image will be at this index
        onImageChange?.(uploadedImageIndex);

        onAddImage?.();
      }
    } catch (error) {
      console.error("Error creating profile image:", error);
      Toast.error(
        intl.formatMessage({
          id: "toast.upload_failed",
          defaultMessage: "Failed to add image"
        })
      );

      // Fallback to legacy behavior
      setImage?.(imageData);
    } finally {
      setEncodingImage?.(false);
    }
  };

  const handleImageChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    ImageUtils.onImageChange(event, handleImageUpload);
  };

  const handleImageURL = (url: string) => {
    if (url) {
      handleImageUpload(url);
    }
  };

  // Navigation button handlers
  const handlePreviousClick = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    goToPrevious();
  }, [goToPrevious]);

  const handleNextClick = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    goToNext();
  }, [goToNext]);

  const handleDotClick = useCallback((index: number) => {
    return (e: React.MouseEvent) => {
      e.stopPropagation();
      setActiveIndex(index);
      onImageChange?.(index);
    };
  }, [onImageChange]);

  return (
    <div className={`profile-image-slider ${isEditing ? 'editing' : ''}`}>
      <div className="performer-image-container">
        <div
          className={`image-container ${isDragging ? 'dragging' : ''}`}
          onTouchStart={isCropping ? undefined : handleTouchStart}
          onTouchMove={isCropping ? undefined : handleTouchMove}
          onTouchEnd={isCropping ? undefined : handleTouchEnd}
          onMouseDown={isCropping ? undefined : handleMouseDown}
          onMouseMove={isCropping ? undefined : handleMouseMove}
          onMouseUp={isCropping ? undefined : handleMouseUp}
          onDragStart={(e) => e.preventDefault()}
          style={{
            userSelect: 'none',
            WebkitUserSelect: 'none',
            cursor: isCropping ? 'default' : 'pointer',
            pointerEvents: isCropping ? 'none' : 'auto'
          }}
        >
          <div className="image-wrapper">
            {sortedImages.length === 0 ? (
              <ProfileImageCropper
                imageSrc={activeImage || ""}
                profileImageId={0}
                performerId={performerId.toString()}
                isNew={isNew}
                onCroppingChange={() => {}}
                onImageUpdate={onImageUpdate}
                onAddImage={onAddImage}
                onImageClick={() => {}}
                onImageChange={onImageChange}
                profileImages={profileImages}
                setImage={setImage}
                setEncodingImage={setEncodingImage}
                onPerformerUpdate={onPerformerUpdate}
              />
            ) : (
              <>
                {currentImage?.image_path && (
                <ProfileImageCropper
                  imageSrc={currentImage.image_path}
                  profileImageId={parseInt(currentImage.id, 10)}
                  performerId={performerId.toString()}
                  isNew={isNew}
                  onCroppingChange={setIsCropping}
                  onImageUpdate={onImageUpdate}
                  onAddImage={onAddImage}
                  onImageClick={handleImageClick}
                  onImageChange={onImageChange}
                  profileImages={profileImages}
                  setImage={setImage}
                  setEncodingImage={setEncodingImage}
                  onPerformerUpdate={onPerformerUpdate}
                  onSetPrimary={onSetPrimary}
                  onDeleteImage={onDeleteImage}
                />
                )}

                {/* Primary indicator */}
                {currentImage?.is_primary && (
                  <div className="primary-indicator">
                    <Icon icon={faStar} className="text-warning" />
                  </div>
                )}
              </>
            )}

            {/* Loading overlay for image encoding */}
            {encodingImage && (
              <div className="image-loading-overlay">
                <LoadingIndicator
                  message={intl.formatMessage({ id: "actions.encoding_image" })}
                />
              </div>
            )}

            {/* Navigation arrows */}
            {hasMultipleImages && !isCropping && (
              <>
                <button
                  className="nav-arrow nav-arrow-left"
                  onClick={handlePreviousClick}
                  title={intl.formatMessage({
                    id: "actions.previous_action",
                    defaultMessage: "Previous"
                  })}
                >
                  <Icon icon={faChevronLeft} />
                </button>
                <button
                  className="nav-arrow nav-arrow-right"
                  onClick={handleNextClick}
                  title={intl.formatMessage({
                    id: "actions.next_action",
                    defaultMessage: "Next"
                  })}
                >
                  <Icon icon={faChevronRight} />
                </button>
              </>
            )}
          </div>
        </div>

        {/* Dots indicator (for multiple images) - outside image container */}
        {hasMultipleImages && !isCropping && (
          <div className="dots-indicator">
            {sortedImages.map((image, index) => (
              <button
                key={image.id}
                className={`dot ${index === activeIndex ? 'active' : ''} ${image.is_primary ? 'primary' : ''}`}
                onClick={handleDotClick(index)}
                title={`Image ${index + 1}${image.is_primary ? ' (Primary)' : ''}`}
              />
            ))}
          </div>
        )}

        {/* Action buttons */}
      </div>
    </div>
  );
};
