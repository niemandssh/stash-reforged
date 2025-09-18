import React, { useRef, useEffect, useState } from "react";
import { Button } from "react-bootstrap";
import Cropper from "cropperjs";
import { faCrop } from "@fortawesome/free-solid-svg-icons";
import { Icon } from "src/components/Shared/Icon";
import { usePerformerProfileImageUpdate } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";

interface IProfileImageCropperProps {
  imageSrc: string;
  profileImageId: number;
  performerId: string;
  onCroppingChange?: (cropping: boolean) => void;
}

export const ProfileImageCropper: React.FC<IProfileImageCropperProps> = ({
  imageSrc,
  profileImageId,
  performerId,
  onCroppingChange,
}) => {
  const imageRef = useRef<HTMLImageElement>(null);
  const cropperRef = useRef<Cropper | null>(null);
  const [cropping, setCropping] = useState(false);
  const [cropInfo, setCropInfo] = useState("");
  const [cropperReady, setCropperReady] = useState(false);
  const [updateProfileImage] = usePerformerProfileImageUpdate();
  const Toast = useToast();

  useEffect(() => {
    return () => {
      if (cropperRef.current) {
        (cropperRef.current as { destroy: () => void }).destroy();
      }
    };
  }, []);

  const handleImageClick = (evt: React.MouseEvent) => {
    if (cropping) {
      evt.preventDefault();
      evt.stopPropagation();
    }
  };

  const handleCropStart = () => {
    if (!imageRef.current) return;

    setCropping(true);
    setCropperReady(false);
    onCroppingChange?.(true);

    const cropperOptions: Record<string, unknown> = {
      viewMode: 1,
      initialAspectRatio: 2 / 3,
      movable: false,
      rotatable: false,
      scalable: false,
      zoomable: false,
      zoomOnTouch: false,
      zoomOnWheel: false,
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

  const handleCropAccept = async () => {
    if (!cropperRef.current) return;

    setCropping(false);
    setCropperReady(false);
    onCroppingChange?.(false);
    const cropInfoText = cropInfo;
    setCropInfo("");

    try {
      const croppedCanvas = (cropperRef.current as { getCroppedCanvas: () => HTMLCanvasElement }).getCroppedCanvas();
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

  if (!imageSrc || imageSrc.includes('default=true')) {
    return null;
  }

  return (
    <div className="image-cropper-container" style={{ pointerEvents: 'auto' }}>
      <div className="detail-header-image" style={{ flexDirection: "column", pointerEvents: 'auto' }}>
        <img
          ref={imageRef}
          src={imageSrc}
          className="performer"
          alt="Performer Profile Image"
          onClick={handleImageClick}
          style={{ transition: "none" }}
        />
      </div>

      <div className="crop-btn-container">
        <Button
          className="crop-start"
          variant="secondary"
          onClick={(e) => {
            e.stopPropagation();
            handleCropStart();
          }}
          style={{ display: cropping ? "none" : "inline-block" }}
        >
          <Icon icon={faCrop} className="mr-1" />
          Crop Image
        </Button>

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
    </div>
  );
};
