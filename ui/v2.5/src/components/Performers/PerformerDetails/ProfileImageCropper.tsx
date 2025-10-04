import React, { useRef, useEffect, useState } from "react";
import { Button } from "react-bootstrap";
import Cropper from "cropperjs";
import { faStar, faTrash } from "@fortawesome/free-solid-svg-icons";
import { Icon } from "src/components/Shared/Icon";
import { ImageInput } from "src/components/Shared/ImageInput";
import { usePerformerProfileImageUpdate, usePerformerProfileImageCreate } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";
import { useIntl } from "react-intl";
import ImageUtils from "src/utils/image";

interface IProfileImageCropperProps {
  imageSrc: string;
  profileImageId: number;
  performerId: string;
  isNew?: boolean;
  onCroppingChange?: (cropping: boolean) => void;
  onImageUpdate?: () => Promise<void>;
  onAddImage?: () => void;
  onImageClick?: (e: React.MouseEvent) => void;
  onImageChange?: (index: number) => void;
  profileImages?: any[];
  setImage?: (image?: string | null) => void;
  setEncodingImage?: (loading: boolean) => void;
  onPerformerUpdate?: (updatedPerformer: Partial<any>) => void;
  onSetPrimary?: (imageId: string, index: number) => void;
  onDeleteImage?: (imageId: string, index: number) => void;
}

export const ProfileImageCropper: React.FC<IProfileImageCropperProps> = ({
  imageSrc,
  profileImageId,
  performerId,
  isNew = false,
  onCroppingChange,
  onImageUpdate,
  onAddImage,
  onImageClick,
  onImageChange,
  profileImages = [],
  setImage,
  setEncodingImage,
  onPerformerUpdate,
  onSetPrimary,
  onDeleteImage,
}) => {
  const imageRef = useRef<HTMLImageElement>(null);
  const cropperRef = useRef<Cropper | null>(null);
  const [cropping, setCropping] = useState(false);
  const [cropInfo, setCropInfo] = useState("");
  const [cropperReady, setCropperReady] = useState(false);
  const [updateProfileImage] = usePerformerProfileImageUpdate();
  const [createProfileImage] = usePerformerProfileImageCreate();
  const Toast = useToast();
  const intl = useIntl();

  useEffect(() => {
    return () => {
      if (cropperRef.current) {
        (cropperRef.current as { destroy: () => void }).destroy();
      }
    };
  }, []);

  const handleCropStart = () => {
    if (!imageRef.current) return;

    setCropping(true);
    setCropperReady(false);
    onCroppingChange?.(true);

    const cropperOptions: Record<string, unknown> = {
      viewMode: 2, // Restrict the crop box not to exceed the size of the canvas
      aspectRatio: 308 / 412, // Maintain aspect ratio
      initialAspectRatio: 308 / 412,
      movable: true, // Allow moving the crop box
      rotatable: false,
      scalable: true, // Allow scaling while maintaining aspect ratio
      zoomable: false,
      zoomOnTouch: false,
      zoomOnWheel: false,
      cropBoxResizable: true, // Allow resizing the crop box
      cropBoxMovable: true, // Allow moving the crop box
      autoCropArea: 0.8, // Start with 80% of the image area
      responsive: true,
      restore: false, // Don't restore crop box
      checkCrossOrigin: false,
      checkOrientation: false,
      ready() {
        setCropperReady(true);
      },
      crop(e: { detail: { x: number; y: number; width: number; height: number } }) {
        setCropInfo(
          `X: ${Math.round(e.detail.x)}, Y: ${Math.round(e.detail.y)}, Width: ${Math.round(e.detail.width)}px, Height: ${Math.round(e.detail.height)}px`
        );
      },
    };

    cropperRef.current = new Cropper(imageRef.current, cropperOptions);
  };

  const handleImageClick = (evt: React.MouseEvent) => {
    if (cropping) {
      evt.preventDefault();
      evt.stopPropagation();
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
            performer_id: performerId,
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

        // Trigger image update to refresh the UI and switch to new image
        await onImageUpdate?.();

        // Switch to the newly uploaded image after data update
        const uploadedImageIndex = profileImages.length; // New image will be at this index
        onImageChange?.(uploadedImageIndex);
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

  const handleCropAccept = async () => {
    if (!cropperRef.current) return;

    setCropping(false);
    setCropperReady(false);
    onCroppingChange?.(false);
    const cropInfoText = cropInfo;
    setCropInfo("");

    try {
      const croppedCanvas = (cropperRef.current as { getCroppedCanvas: () => HTMLCanvasElement }).getCroppedCanvas();

      if (!croppedCanvas) {
        throw new Error("Failed to get cropped canvas");
      }

      const imageDataUrl = croppedCanvas.toDataURL();

      const result = await updateProfileImage({
        variables: {
          input: {
            id: profileImageId.toString(),
            image: imageDataUrl,
          },
        },
      });

      if (result.data?.performerProfileImageUpdate) {
        Toast.success("Image cropped successfully");
        onImageUpdate?.();
      }
    } catch (error) {
      console.error("Error cropping image:", error);
      Toast.error("Failed to crop image");
    }

    if (cropperRef.current) {
      (cropperRef.current as { destroy: () => void }).destroy();
      cropperRef.current = null;
    }
  };

  const handleCropCancel = () => {
    setCropping(false);
    setCropperReady(false);
    onCroppingChange?.(false);
    setCropInfo("");

    if (cropperRef.current) {
      (cropperRef.current as { destroy: () => void }).destroy();
      cropperRef.current = null;
    }
  };

  const hasNoImages = (!profileImages || profileImages.length === 0) && !imageSrc;
  const currentImageIndex = profileImages?.findIndex(img => img.id === profileImageId.toString()) ?? -1;
  const currentImage = profileImages?.find(img => img.id === profileImageId.toString());

  const showAddImageOnly = !imageSrc;
  const hasImageToEdit = !!imageSrc;

  return (
    <div className="image-cropper-container" style={{ pointerEvents: 'auto' }}>
      <div
        className="detail-header-image"
        style={{ flexDirection: "column", pointerEvents: 'auto' }}
        onClick={onImageClick}
      >
        <div
          className="background-image"
          style={{
            backgroundImage: `url(${imageSrc})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            backgroundRepeat: 'no-repeat',
            filter: 'blur(10px)',
            transform: 'scale(1.1)'
          }}
        />
        <img
          src={imageSrc}
          crossOrigin="anonymous"
          style={{ display: 'none' }}
          alt="hidden"
        />

        {imageSrc && (
          <img
            ref={imageRef}
            src={imageSrc}
            crossOrigin="anonymous"
            className="performer"
            alt="Performer Profile Image"
            onClick={handleImageClick}
            style={{ transition: "none" }}
          />
        )}
      </div>

      {(showAddImageOnly || hasImageToEdit) && (
        <div className={`crop-btn-container ${cropping ? 'is-cropping' : ''} ${showAddImageOnly ? 'add-only' : ''}`}>
        {!showAddImageOnly && (
          <Button
            className="crop-start"
            variant="secondary"
            onClick={(e) => {
              e.stopPropagation();
              handleCropStart();
            }}
            style={{ display: cropping ? "none" : "inline-block" }}
          >
            Crop Image
          </Button>
        )}

          <div
            className="add-image-btn"
            style={{ display: cropping ? "none" : "flex", marginLeft: "8px", gap: "8px" }}
            onClick={(e) => e.stopPropagation()}
          >
            {showAddImageOnly ? (
              <ImageInput
                isEditing={true}
                onImageChange={handleImageChange}
                onImageURL={handleImageURL}
                text={intl.formatMessage({ id: "actions.set_photo" })}
              />
            ) : (
              <>
                <ImageInput
                  isEditing={true}
                  onImageChange={handleImageChange}
                  onImageURL={handleImageURL}
                  text={intl.formatMessage({ id: "actions.set_photo" })}
                />

                {currentImage && (
                  <Button
                    variant="secondary"
                    disabled={currentImage.is_primary}
                    onClick={(e) => {
                      e.stopPropagation();
                      onSetPrimary?.(currentImage.id, currentImageIndex);
                    }}
                    style={{ display: cropping ? "none" : "inline-block" }}
                    title={intl.formatMessage({
                      id: "actions.set_as_primary",
                      defaultMessage: "Set as primary"
                    })}
                  >
                    <Icon icon={faStar} />
                  </Button>
                )}

                {currentImage && (
                  <Button
                    variant="danger"
                    onClick={(e) => {
                      e.stopPropagation();
                      onDeleteImage?.(currentImage.id, currentImageIndex);
                    }}
                    style={{ display: cropping ? "none" : "inline-block" }}
                    title={intl.formatMessage({
                      id: "actions.delete_entity",
                      defaultMessage: "Delete {entityType}",
                    }, { entityType: intl.formatMessage({ id: "image" }) })}
                  >
                    <Icon icon={faTrash} />
                  </Button>
                )}
              </>
            )}
          </div>

        <Button
          className="crop-cancel"
          variant="danger"
          onClick={(e) => {
            e.stopPropagation();
            handleCropCancel();
          }}
          style={{ display: cropping ? "inline-block" : "none" }}
        >
          Cancel
        </Button>

        <Button
          className="crop-accept"
          variant="success"
          onClick={(e) => {
            e.stopPropagation();
            handleCropAccept();
          }}
          style={{ display: cropping && cropperReady ? "inline-block" : "none" }}
        >
          OK
        </Button>

        {cropInfo && <p>{cropInfo}</p>}
        </div>
      )}
    </div>
  );
};
