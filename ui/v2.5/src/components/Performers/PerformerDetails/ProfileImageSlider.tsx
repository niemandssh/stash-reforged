import React, { useState, useEffect, useMemo } from "react";
import { Button } from "react-bootstrap";
import { useIntl } from "react-intl";
import { Icon } from "src/components/Shared/Icon";
import { faTrash, faStar, faChevronLeft, faChevronRight } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { ProfileImageCropper } from "./ProfileImageCropper";
import { usePerformerProfileImageUpdate, usePerformerProfileImageDestroy } from "src/core/StashService";
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
  performerId: number;
}

export const ProfileImageSlider: React.FC<IProfileImageSliderProps> = ({
  profileImages,
  isEditing,
  currentImageIndex = 0,
  onImageChange,
  onDeleteImage,
  onSetPrimary,
  performerId,
}) => {
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

  useEffect(() => {
    setActiveIndex(currentImageIndex);
  }, [currentImageIndex]);

  if (!profileImages || profileImages.length === 0) {
    return null;
  }

  // Sort images: primary first, then by position
  const sortedImages = [...profileImages].sort((a, b) => {
    if (a.is_primary && !b.is_primary) return -1;
    if (!a.is_primary && b.is_primary) return 1;
    return (a.position || 0) - (b.position || 0);
  });

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

  const lightboxState = useMemo(() => ({
    images: lightboxImages,
    showNavigation: false,
    // Hide unnecessary elements for profile images
    hideGallery: true,
    hideRating: true,
    hideOCounter: true,
  }), [lightboxImages]);

  const showLightbox = useLightbox(lightboxState);

  const handleImageClick = (e: React.MouseEvent) => {
    // Don't open lightbox if we're dragging, if it's a touch event, if we just swiped, or if we're cropping
    if (isDragging || touchStart !== null || hasSwiped || isCropping) {
      return;
    }
    e.preventDefault();
    showLightbox({ initialIndex: activeIndex });
  };
  
  const currentImage = sortedImages[activeIndex];
  const hasMultipleImages = sortedImages.length > 1;

  const goToPrevious = () => {
    const newIndex = activeIndex > 0 ? activeIndex - 1 : sortedImages.length - 1;
    setActiveIndex(newIndex);
    onImageChange?.(newIndex);
  };

  const goToNext = () => {
    const newIndex = activeIndex < sortedImages.length - 1 ? activeIndex + 1 : 0;
    setActiveIndex(newIndex);
    onImageChange?.(newIndex);
  };

  // Touch handlers for swipe
  const handleTouchStart = (e: React.TouchEvent) => {
    // Don't handle swipe if cropping is active
    if (isCropping) return;
    
    if (hasMultipleImages) {
      e.preventDefault();
      setIsDragging(true);
      setTouchEnd(null);
      setTouchStart(e.targetTouches[0].clientX);
    }
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    // Don't handle swipe if cropping is active
    if (isCropping) return;
    
    if (isDragging && hasMultipleImages) {
      e.preventDefault();
      setTouchEnd(e.targetTouches[0].clientX);
    }
  };

  const handleTouchEnd = (e: React.TouchEvent) => {
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
  };

  // Mouse handlers for desktop swipe simulation
  const handleMouseDown = (e: React.MouseEvent) => {
    // Don't handle swipe if cropping is active
    if (isCropping) return;
    
    if (hasMultipleImages) {
      e.preventDefault();
      setIsDragging(true);
      setTouchEnd(null);
      setTouchStart(e.clientX);
    }
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    // Don't handle swipe if cropping is active
    if (isCropping) return;
    
    if (isDragging && hasMultipleImages) {
      e.preventDefault();
      setTouchEnd(e.clientX);
    }
  };

  const handleMouseUp = (e: React.MouseEvent) => {
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
  };

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
          id: "toast.set_primary_success",
          defaultMessage: "Set as primary image" 
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

  return (
    <div className={`profile-image-slider ${isEditing ? 'editing' : ''}`}>
      <div 
        className={`image-container ${isDragging ? 'dragging' : ''}`}
        onTouchStart={isCropping ? undefined : handleTouchStart}
        onTouchMove={isCropping ? undefined : handleTouchMove}
        onTouchEnd={isCropping ? undefined : handleTouchEnd}
        onMouseDown={isCropping ? undefined : handleMouseDown}
        onMouseMove={isCropping ? undefined : handleMouseMove}
        onMouseUp={isCropping ? undefined : handleMouseUp}
        onClick={isCropping ? undefined : handleImageClick}
        onDragStart={(e) => e.preventDefault()}
        style={{ 
          userSelect: 'none', 
          WebkitUserSelect: 'none', 
          cursor: isCropping ? 'default' : 'pointer',
          pointerEvents: isCropping ? 'none' : 'auto'
        }}
      >
        {currentImage?.image_path && (
          <ProfileImageCropper
            imageSrc={currentImage.image_path}
            profileImageId={parseInt(currentImage.id, 10)}
            performerId={performerId.toString()}
            onCroppingChange={setIsCropping}
          />
        )}

                {/* Primary indicator */}
                {currentImage?.is_primary && (
                  <div className="primary-indicator">
                    <Icon icon={faStar} className="text-warning" />
                  </div>
                )}

                {/* Navigation arrows */}
                {hasMultipleImages && !isCropping && (
                  <>
                    <button
                      className="nav-arrow nav-arrow-left"
                      onClick={(e) => {
                        e.stopPropagation();
                        goToPrevious();
                      }}
                      title={intl.formatMessage({ 
                        id: "actions.previous", 
                        defaultMessage: "Previous" 
                      })}
                    >
                      <Icon icon={faChevronLeft} />
                    </button>
                    <button
                      className="nav-arrow nav-arrow-right"
                      onClick={(e) => {
                        e.stopPropagation();
                        goToNext();
                      }}
                      title={intl.formatMessage({ 
                        id: "actions.next", 
                        defaultMessage: "Next" 
                      })}
                    >
                      <Icon icon={faChevronRight} />
                    </button>
                  </>
                )}

              </div>

      {/* Dots indicator (for multiple images) - outside image container */}
      {hasMultipleImages && !isCropping && (
        <div className="dots-indicator">
          {sortedImages.map((image, index) => (
            <button
              key={image.id}
              className={`dot ${index === activeIndex ? 'active' : ''} ${image.is_primary ? 'primary' : ''}`}
              onClick={(e) => {
                e.stopPropagation();
                setActiveIndex(index);
                onImageChange?.(index);
              }}
              title={`Image ${index + 1}${image.is_primary ? ' (Primary)' : ''}`}
            />
          ))}
        </div>
      )}

      {/* Slider controls (only in edit mode) */}
      {isEditing && (
        <div className="slider-controls">
          {/* Action buttons */}
          <div className="action-buttons">
            {!currentImage?.is_primary && (
              <Button
                variant="secondary"
                size="sm"
                onClick={handleSetPrimary}
                title={intl.formatMessage({ 
                  id: "actions.set_as_primary", 
                  defaultMessage: "Set as primary" 
                })}
              >
                <Icon icon={faStar} className="mr-2" />
                {intl.formatMessage({ 
                  id: "actions.set_primary", 
                  defaultMessage: "Set Primary" 
                })}
              </Button>
            )}
            
            <Button
              variant="danger"
              size="sm"
              onClick={handleDeleteImage}
              title={intl.formatMessage({ 
                id: "actions.delete_entity", 
                defaultMessage: "Delete {entityType}",
              }, { entityType: intl.formatMessage({ id: "image" }) })}
            >
              <Icon icon={faTrash} className="mr-2" />
              {intl.formatMessage({ 
                id: "actions.delete_entity", 
                defaultMessage: "Delete {entityType}",
              }, { entityType: intl.formatMessage({ id: "image" }) })}
            </Button>
          </div>
        </div>
      )}

    </div>
  );
};
