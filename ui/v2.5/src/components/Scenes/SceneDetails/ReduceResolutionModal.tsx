import React, { useState } from "react";
import { Form, Alert, Row, Col } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { ModalComponent } from "src/components/Shared/Modal";
import { useToast } from "src/hooks/Toast";
import * as GQL from "src/core/generated-graphql";
import { useSceneReduceResolution } from "src/core/StashService";
import { faCompressAlt } from "@fortawesome/free-solid-svg-icons";

interface IReduceResolutionModalProps {
  scene: GQL.SceneDataFragment;
  onClose: () => void;
}

interface IResolutionOption {
  width: number;
  height: number;
  label: string;
}

function getResolutionOptions(
  currentWidth: number,
  currentHeight: number
): IResolutionOption[] {
  const aspectRatio = currentWidth / currentHeight;
  const options: IResolutionOption[] = [];

  // Common resolutions in descending order
  const standardResolutions = [
    { width: 3840, height: 2160, label: "4K (3840x2160)" },
    { width: 2560, height: 1440, label: "2K (2560x1440)" },
    { width: 1920, height: 1080, label: "1080p (1920x1080)" },
    { width: 1280, height: 720, label: "720p (1280x720)" },
    { width: 854, height: 480, label: "480p (854x480)" },
    { width: 640, height: 360, label: "360p (640x360)" },
  ];

  // Filter resolutions that are smaller than current and maintain aspect ratio
  for (const res of standardResolutions) {
    if (res.width < currentWidth && res.height < currentHeight) {
      // Check if aspect ratio is close (within 5%)
      const resAspectRatio = res.width / res.height;
      if (Math.abs(aspectRatio - resAspectRatio) / aspectRatio < 0.05) {
        options.push(res);
      }
    }
  }

  // If no standard resolutions fit, calculate proportional resolutions
  if (options.length === 0) {
    const scales = [0.75, 0.5, 0.25];
    for (const scale of scales) {
      const width = Math.floor(currentWidth * scale);
      const height = Math.floor(currentHeight * scale);
      // Lower minimum requirements for smaller videos
      if (width >= 320 && height >= 240) {
        options.push({
          width,
          height,
          label: `${width}x${height} (${Math.round(scale * 100)}%)`,
        });
      }
    }
  }

  return options;
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

export const ReduceResolutionModal: React.FC<IReduceResolutionModalProps> = ({
  scene,
  onClose,
}) => {
  const intl = useIntl();
  const Toast = useToast();
  const [reduceResolution] = useSceneReduceResolution();

  const [selectedFileId, setSelectedFileId] = useState<string>(
    scene.files.length > 0 ? scene.files[0].id : ""
  );
  const [selectedResolution, setSelectedResolution] = useState<string>("");
  const [isProcessing, setIsProcessing] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const selectedFile = scene.files.find((f) => f.id === selectedFileId);
  const resolutionOptions = selectedFile
    ? getResolutionOptions(selectedFile.width || 0, selectedFile.height || 0)
    : [];

  const handleSubmit = () => {
    if (!selectedFileId || !selectedResolution) {
      Toast.error(
        intl.formatMessage({ id: "dialogs.reduce_resolution.select_required" })
      );
      return;
    }
    setShowConfirm(true);
  };

  const handleConfirm = async () => {
    if (!selectedFileId || !selectedResolution) return;

    const [width, height] = selectedResolution.split("x").map(Number);

    setIsProcessing(true);
    try {
      const result = await reduceResolution({
        variables: {
          input: {
            scene_id: scene.id,
            file_id: selectedFileId,
            target_width: width,
            target_height: height,
          },
        },
      });

      if (result.data?.sceneReduceResolution) {
        Toast.success(
          intl.formatMessage(
            { id: "actions.reduce_resolution_started" },
            { jobId: result.data.sceneReduceResolution }
          )
        );
        onClose();
      }
    } catch (e) {
      Toast.error(e);
      setShowConfirm(false);
    } finally {
      setIsProcessing(false);
    }
  };

  if (showConfirm) {
    const [width, height] = selectedResolution.split("x").map(Number);
    return (
      <ModalComponent
        show
        icon={faCompressAlt}
        header={intl.formatMessage({
          id: "dialogs.reduce_resolution.confirm_title",
        })}
        accept={{
          variant: "danger",
          onClick: handleConfirm,
          text: intl.formatMessage({ id: "actions.confirm" }),
        }}
        cancel={{
          onClick: () => setShowConfirm(false),
          text: intl.formatMessage({ id: "actions.cancel" }),
          variant: "secondary",
        }}
        isRunning={isProcessing}
      >
        <Alert variant="warning">
          <FormattedMessage id="dialogs.reduce_resolution.warning" />
        </Alert>
        <p>
          <FormattedMessage
            id="dialogs.reduce_resolution.confirm_message"
            values={{
              currentResolution: `${selectedFile?.width}x${selectedFile?.height}`,
              targetResolution: `${width}x${height}`,
            }}
          />
        </p>
      </ModalComponent>
    );
  }

  return (
    <ModalComponent
      show
      icon={faCompressAlt}
      header={intl.formatMessage({ id: "dialogs.reduce_resolution.title" })}
      accept={{
        onClick: handleSubmit,
        text: intl.formatMessage({ id: "actions.reduce_resolution" }),
        disabled: !selectedFileId || !selectedResolution,
      }}
      cancel={{
        onClick: onClose,
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
      }}
      isRunning={isProcessing}
    >
      <Form>
        {scene.files.length > 1 && (
          <Form.Group controlId="file-select" as={Row}>
            <Form.Label column sm={3}>
              <FormattedMessage id="file" />
            </Form.Label>
            <Col sm={9}>
              <Form.Control
                as="select"
                value={selectedFileId}
                onChange={(e) => {
                  setSelectedFileId(e.target.value);
                  setSelectedResolution("");
                }}
                className="input-control"
              >
                {scene.files.map((file) => (
                  <option key={file.id} value={file.id}>
                    {file.path ? file.path.split("/").pop() : "Unknown file"} (
                    {file.width}x{file.height} - {formatFileSize(file.size)})
                  </option>
                ))}
              </Form.Control>
            </Col>
          </Form.Group>
        )}

        {scene.files.length === 1 && selectedFile && (
          <Alert variant="info">
            <strong>
              <FormattedMessage id="file" />:{" "}
            </strong>
            {selectedFile.path
              ? selectedFile.path.split("/").pop()
              : "Unknown file"}
            <br />
            <strong>
              <FormattedMessage id="resolution" />:{" "}
            </strong>
            {selectedFile.width}x{selectedFile.height}
            <br />
            <strong>
              <FormattedMessage id="size" />:{" "}
            </strong>
            {formatFileSize(selectedFile.size)}
          </Alert>
        )}

        <Form.Group controlId="resolution-select" as={Row}>
          <Form.Label column sm={3}>
            <FormattedMessage id="dialogs.reduce_resolution.target_resolution" />
          </Form.Label>
          <Col sm={9}>
            {resolutionOptions.length > 0 ? (
              <Form.Control
                as="select"
                value={selectedResolution}
                onChange={(e) => setSelectedResolution(e.target.value)}
                className="input-control"
              >
                <option value="">
                  {intl.formatMessage({
                    id: "dialogs.reduce_resolution.select_resolution",
                  })}
                </option>
                {resolutionOptions.map((opt) => (
                  <option
                    key={`${opt.width}x${opt.height}`}
                    value={`${opt.width}x${opt.height}`}
                  >
                    {opt.label}
                  </option>
                ))}
              </Form.Control>
            ) : (
              <Alert variant="warning">
                <FormattedMessage id="dialogs.reduce_resolution.no_options" />
              </Alert>
            )}
          </Col>
        </Form.Group>

        {selectedResolution && (
          <Alert variant="info">
            <FormattedMessage id="dialogs.reduce_resolution.info" />
          </Alert>
        )}
      </Form>
    </ModalComponent>
  );
};
