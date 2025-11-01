import React, { useRef, useEffect, useState } from "react";
import { Button } from "react-bootstrap";
import Cropper from "cropperjs";
import { faCrop } from "@fortawesome/free-solid-svg-icons";
import { Icon } from "./Icon";
import { useStudioUpdate } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";

interface IStudioImageCropperProps {
  imageSrc: string;
  studioId: string;
}

export const StudioImageCropper: React.FC<IStudioImageCropperProps> = ({
  imageSrc,
  studioId,
}) => {
  const imageRef = useRef<HTMLImageElement>(null);
  const cropperRef = useRef<Cropper | null>(null);
  const [cropping, setCropping] = useState(false);
  const [cropInfo, setCropInfo] = useState("");
  const [cropperReady, setCropperReady] = useState(false);
  const [updateStudio] = useStudioUpdate();
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

    const cropperOptions: Record<string, unknown> = {
      viewMode: 1,
      initialAspectRatio: 1,
      movable: false,
      rotatable: false,
      scalable: false,
      zoomable: false,
      zoomOnTouch: false,
      zoomOnWheel: false,
      ready() {
        setCropperReady(true);
      },
      crop(e: {
        detail: { x: number; y: number; width: number; height: number };
      }) {
        setCropInfo(
          `X: ${Math.round(e.detail.x)}, Y: ${Math.round(
            e.detail.y
          )}, Width: ${Math.round(e.detail.width)}px, Height: ${Math.round(
            e.detail.height
          )}px`
        );
      },
    };

    cropperRef.current = new Cropper(imageRef.current, cropperOptions);
  };

  const handleCropAccept = async () => {
    if (!cropperRef.current) return;

    setCropping(false);
    setCropperReady(false);
    const cropInfoText = cropInfo;
    setCropInfo("");

    try {
      const croppedCanvas = (
        cropperRef.current as { getCroppedCanvas: () => HTMLCanvasElement }
      ).getCroppedCanvas();
      const imageDataUrl = croppedCanvas.toDataURL();

      const result = await updateStudio({
        variables: {
          input: {
            image: imageDataUrl,
            id: studioId,
          },
        },
      });

      if (result.data?.studioUpdate?.id) {
        if (imageRef.current) {
          const newSrc = imageRef.current.src + "?t=" + Date.now();
          imageRef.current.src = newSrc;
        }
        (cropperRef.current as { destroy: () => void }).destroy();
        cropperRef.current = null;
      } else if (result.errors?.[0]?.message) {
        setCropping(true);
        setCropperReady(true);
        setCropInfo(cropInfoText);
        Toast.error(result.errors[0].message);
      }
    } catch (error) {
      setCropping(true);
      setCropperReady(true);
      setCropInfo(cropInfoText);
      Toast.error(
        error instanceof Error
          ? error.message
          : "Ошибка при обрезке изображения"
      );
    }
  };

  const handleCropCancel = () => {
    setCropping(false);
    setCropperReady(false);
    setCropInfo("");

    if (cropperRef.current) {
      (cropperRef.current as { destroy: () => void }).destroy();
      cropperRef.current = null;
    }
  };

  if (!imageSrc || imageSrc.includes("default=true")) {
    return null;
  }

  return (
    <div className="image-cropper-container">
      <img
        ref={imageRef}
        src={imageSrc}
        className="logo"
        alt="Studio"
        onClick={handleImageClick}
        style={{ transition: "none" }}
      />

      <div className="crop-btn-container">
        <Button
          className="crop-start"
          variant="secondary"
          onClick={handleCropStart}
          style={{ display: cropping ? "none" : "inline-block" }}
        >
          <Icon icon={faCrop} className="mr-1" />
          Crop Image
        </Button>

        <Button
          className="crop-cancel"
          variant="danger"
          onClick={handleCropCancel}
          style={{ display: cropping ? "inline-block" : "none" }}
        >
          Cancel
        </Button>

        <Button
          className="crop-accept"
          variant="success"
          onClick={handleCropAccept}
          style={{
            display: cropping && cropperReady ? "inline-block" : "none",
          }}
        >
          OK
        </Button>

        {cropInfo && <p>{cropInfo}</p>}
      </div>
    </div>
  );
};
