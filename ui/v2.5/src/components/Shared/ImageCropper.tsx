import React, { useRef, useEffect, useState } from "react";
import { Button } from "react-bootstrap";
import Cropper from "cropperjs";
import { faCrop } from "@fortawesome/free-solid-svg-icons";
import { Icon } from "./Icon";
import { usePerformerUpdate } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";

interface IImageCropperProps {
  imageSrc: string;
  performerId: string;
}

export const ImageCropper: React.FC<IImageCropperProps> = ({
  imageSrc,
  performerId,
}) => {
  const imageRef = useRef<HTMLImageElement>(null);
  const cropperRef = useRef<Cropper | null>(null);
  const [cropping, setCropping] = useState(false);
  const [cropInfo, setCropInfo] = useState("");
  const [cropperReady, setCropperReady] = useState(false);
  const [updatePerformer] = usePerformerUpdate();
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
    const cropInfoText = cropInfo;
    setCropInfo("");

    try {
      const croppedCanvas = (cropperRef.current as { getCroppedCanvas: () => HTMLCanvasElement }).getCroppedCanvas();
      const imageDataUrl = croppedCanvas.toDataURL();

      const result = await updatePerformer({
        variables: {
          input: {
            image: imageDataUrl,
            id: performerId,
          },
        },
      });

      if (result.data?.performerUpdate?.id) {
        // Перезагружаем изображение
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
      Toast.error(error instanceof Error ? error.message : "Ошибка при обрезке изображения");
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

  return (
    <div className="image-cropper-container">
      <div className="detail-header-image" style={{ flexDirection: "column" }}>
        <img
          ref={imageRef}
          src={imageSrc}
          className="performer"
          alt="Performer"
          onClick={handleImageClick}
          style={{ transition: "none" }}
        />
      </div>

      <div id="crop-btn-container">
        <Button
          id="crop-start"
          variant="secondary"
          onClick={handleCropStart}
          style={{ display: cropping ? "none" : "inline-block" }}
        >
          <Icon icon={faCrop} className="mr-1" />
          Crop Image
        </Button>

        <Button
          id="crop-cancel"
          variant="danger"
          onClick={handleCropCancel}
          style={{ display: cropping ? "inline-block" : "none" }}
        >
          Cancel
        </Button>

        <Button
          id="crop-accept"
          variant="success"
          onClick={handleCropAccept}
          style={{ display: cropping && cropperReady ? "inline-block" : "none" }}
        >
          OK
        </Button>

        {cropInfo && <p>{cropInfo}</p>}
      </div>
    </div>
  );
};
