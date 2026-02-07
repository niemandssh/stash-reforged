import React, { useState } from "react";
import { Button, Form } from "react-bootstrap";
import { useIntl } from "react-intl";
import { Icon } from "src/components/Shared/Icon";
import { ImageInput } from "src/components/Shared/ImageInput";
import { faPlus } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { ProfileImageSlider } from "./ProfileImageSlider";
import {
  usePerformerProfileImageCreate,
  usePerformerProfileImageUpdate,
  usePerformerProfileImageDestroy,
} from "src/core/StashService";
import { useToast } from "src/hooks/Toast";
import ImageUtils from "src/utils/image";

interface IProfileImageEditorProps {
  performer: GQL.PerformerDataFragment;
  isVisible: boolean;
  onImagesChange?: (images: GQL.PerformerProfileImage[]) => void;
}

export const ProfileImageEditor: React.FC<IProfileImageEditorProps> = ({
  performer,
  isVisible,
  onImagesChange,
}) => {
  const intl = useIntl();
  const Toast = useToast();
  const [currentImageIndex, setCurrentImageIndex] = useState(0);
  const [isUploading, setIsUploading] = useState(false);

  const [createProfileImage] = usePerformerProfileImageCreate();
  const [updateProfileImage] = usePerformerProfileImageUpdate();
  const [destroyProfileImage] = usePerformerProfileImageDestroy();

  const profileImages = performer.profile_images || [];

  // This function is used by parent components but not directly in this file
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const handleImageUpload = async (imageData: string | null) => {
    if (!imageData) return;

    setIsUploading(true);
    try {
      const result = await createProfileImage({
        variables: {
          input: {
            performer_id: performer.id,
            image: imageData,
            is_primary: profileImages.length === 0, // First image becomes primary
            position: profileImages.length,
          },
        },
      } as any);

      if ((result.data as any)?.performerProfileImageCreate) {
        const newImageIndex = profileImages.length + 1; // +1 because image is added to array after this
        Toast.success(
          intl.formatMessage(
            { id: "toast.created_entity" },
            {
              entity: `${intl
                .formatMessage({ id: "image" })
                .toLocaleLowerCase()} ${newImageIndex}`,
            }
          )
        );

        // Update the images list and set current index to the new image
        const updatedImages = [
          ...profileImages,
          (result.data as any).performerProfileImageCreate,
        ];
        onImagesChange?.(updatedImages);

        // Set current index to the newly created image
        setCurrentImageIndex(updatedImages.length - 1);
      }
    } catch (error) {
      console.error("Error uploading profile image:", error);
      Toast.error(
        intl.formatMessage({
          id: "toast.upload_failed",
          defaultMessage: "Upload failed",
        })
      );
    } finally {
      setIsUploading(false);
    }
  };

  const handleDeleteImage = async (imageId: string, index: number) => {
    try {
      await destroyProfileImage({
        variables: {
          input: { id: imageId },
        },
      } as any);

      Toast.success(
        intl.formatMessage(
          {
            id: "toast.deleted_entity",
            defaultMessage: "Deleted {entityType}",
          },
          { entityType: intl.formatMessage({ id: "image" }) }
        )
      );

      const updatedImages = profileImages.filter((img) => img.id !== imageId);
      onImagesChange?.(updatedImages);

      // Adjust current index if needed
      if (index <= currentImageIndex && currentImageIndex > 0) {
        setCurrentImageIndex(currentImageIndex - 1);
      }
    } catch (error) {
      console.error("Error deleting profile image:", error);
      Toast.error(
        intl.formatMessage({
          id: "toast.delete_failed",
          defaultMessage: "Delete failed",
        })
      );
    }
  };

  const handleSetPrimary = async (imageId: string) => {
    try {
      // First, unset all other images as primary
      await Promise.all(
        profileImages
          .filter((img) => img.is_primary && img.id !== imageId)
          .map((img) =>
            updateProfileImage({
              variables: {
                input: {
                  id: img.id,
                  is_primary: false,
                },
              },
            } as any)
          )
      );

      // Then set the selected image as primary
      await updateProfileImage({
        variables: {
          input: {
            id: imageId,
            is_primary: true,
          },
        },
      } as any);

      Toast.success(
        intl.formatMessage(
          {
            id: "toast.updated_entity",
            defaultMessage: "Updated {entityType}",
          },
          { entityType: intl.formatMessage({ id: "image" }) }
        )
      );

      // Update the images list
      const updatedImages = profileImages.map((img) => ({
        ...img,
        is_primary: img.id === imageId,
      }));
      onImagesChange?.(updatedImages);
    } catch (error) {
      console.error("Error setting primary image:", error);
      Toast.error(
        intl.formatMessage({
          id: "toast.update_failed",
          defaultMessage: "Update failed",
        })
      );
    }
  };

  if (!isVisible) {
    return null;
  }

  return (
    <div className="profile-image-editor">
      <h6>
        {intl.formatMessage({
          id: "performer.profile_images",
          defaultMessage: "Profile Images",
        })}
      </h6>

      {/* Profile Images Slider */}
      {profileImages.length > 0 && (
        <div className="mb-3">
          <ProfileImageSlider
            profileImages={profileImages}
            isEditing={true}
            currentImageIndex={currentImageIndex}
            onImageChange={setCurrentImageIndex}
            onDeleteImage={handleDeleteImage}
            onSetPrimary={handleSetPrimary}
            performerId={parseInt(performer.id!, 10)}
          />
        </div>
      )}

      {/* Add New Image */}
      <Form.Group className="mb-3">
        <Form.Label>
          {intl.formatMessage({
            id: "actions.set_photo",
          })}
        </Form.Label>
        <ImageInput
          isEditing={true}
          onImageChange={(event) => {
            ImageUtils.onImageChange(
              event as React.FormEvent<HTMLInputElement>,
              handleImageUpload
            );
          }}
          onImageURL={handleImageUpload}
        >
          <Button variant="secondary" disabled={isUploading}>
            <Icon icon={faPlus} className="me-2" />
            {isUploading
              ? intl.formatMessage({
                  id: "actions.uploading",
                  defaultMessage: "Uploading...",
                })
              : intl.formatMessage({
                  id: "actions.add_image",
                  defaultMessage: "Add Image",
                })}
          </Button>
        </ImageInput>
      </Form.Group>

      {profileImages.length === 0 && (
        <div className="text-muted">
          {intl.formatMessage({
            id: "performer.no_profile_images",
            defaultMessage: "No profile images. Add your first image above.",
          })}
        </div>
      )}
    </div>
  );
};
