import React, { useState, useContext } from "react";
import { Form, Alert, Row, Col } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { ModalComponent } from "src/components/Shared/Modal";
import { DurationInput } from "src/components/Shared/DurationInput";
import { useToast } from "src/hooks/Toast";
import { ConfigurationContext } from "src/hooks/Config";
import * as GQL from "src/core/generated-graphql";
import { useSceneTrimVideo } from "src/core/StashService";
import { faCut } from "@fortawesome/free-solid-svg-icons";

interface ITrimVideoModalProps {
  scene: GQL.SceneDataFragment;
  onClose: () => void;
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function formatTime(seconds: number): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);
  
  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  }
  return `${minutes}:${secs.toString().padStart(2, '0')}`;
}

export const TrimVideoModal: React.FC<ITrimVideoModalProps> = ({
  scene,
  onClose,
}) => {
  const intl = useIntl();
  const Toast = useToast();
  const { configuration } = useContext(ConfigurationContext);
  const [trimVideo] = useSceneTrimVideo();

  const [selectedFileId, setSelectedFileId] = useState<string>(
    scene.files.length > 0 ? scene.files[0].id : ""
  );
  const [startTime, setStartTime] = useState<number | null>(
    scene.start_time || null
  );
  const [endTime, setEndTime] = useState<number | null>(
    scene.end_time || null
  );
  const [isProcessing, setIsProcessing] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const selectedFile = scene.files.find((f) => f.id === selectedFileId);
  const originalDuration = selectedFile?.duration || 0;

  // Check if start_time or end_time are set
  const hasTrimTimes = scene.start_time !== null || scene.end_time !== null;

  const validateTimes = (start: number | null, end: number | null): string | null => {
    // At least one time must be set
    if (start === null && end === null) {
      return intl.formatMessage({ id: "dialogs.trim_video.select_required" });
    }

    // Validate start time if set
    if (start !== null) {
      if (start < 0) {
        return "Start time cannot be negative";
      }
      if (start >= originalDuration) {
        return `Start time cannot be greater than or equal to video duration (${formatTime(originalDuration)})`;
      }
    }

    // Validate end time if set
    if (end !== null) {
      if (end <= 0) {
        return "End time must be greater than 0";
      }
      if (end > originalDuration) {
        return `End time cannot be greater than video duration (${formatTime(originalDuration)})`;
      }
    }

    // If both are set, validate relationship
    if (start !== null && end !== null) {
      if (end <= start) {
        return "End time must be greater than start time";
      }
    }

    return null;
  };

  const handleSubmit = () => {
    const validationError = validateTimes(startTime, endTime);
    if (validationError) {
      Toast.error(validationError);
      return;
    }
    setShowConfirm(true);
  };

  const handleConfirm = async () => {
    const validationError = validateTimes(startTime, endTime);
    if (validationError) {
      Toast.error(validationError);
      return;
    }

    // At least one time must be set
    if (startTime === null && endTime === null) {
      Toast.error(intl.formatMessage({ id: "dialogs.trim_video.select_required" }));
      return;
    }

    setIsProcessing(true);
    try {
      const result = await trimVideo({
        variables: {
          input: {
            scene_id: scene.id,
            file_id: selectedFileId,
            start_time: startTime || 0,
            end_time: endTime || 0,
          },
        },
      });

      if (result.data?.sceneTrimVideo) {
        Toast.success(
          intl.formatMessage(
            { id: "actions.trim_video_started" },
            { jobId: result.data.sceneTrimVideo }
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
    if (startTime === null || endTime === null) return null;
    
    const trimDuration = endTime - startTime;
    const removedFromEnd = originalDuration - endTime;

    return (
      <ModalComponent
        show
        icon={faCut}
        header={intl.formatMessage({ id: "dialogs.trim_video.confirm_title" })}
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
          <FormattedMessage id="dialogs.trim_video.warning" />
        </Alert>
        <p>
          <FormattedMessage
            id="dialogs.trim_video.confirm_message"
            values={{
              startTime: formatTime(startTime),
              endTime: formatTime(endTime),
              trimDuration: formatTime(trimDuration),
              remainingDuration: formatTime(removedFromEnd),
            }}
          />
        </p>
        <p>
          <strong>
            <FormattedMessage id="dialogs.trim_video.temp_path_label" />
          </strong>
          <br />
          <a 
            href={`file://${configuration?.general?.generatedPath ? 
              configuration.general.generatedPath.substring(0, configuration.general.generatedPath.lastIndexOf('/')) + '/temp' : 
              './temp'}`} 
            target="_blank" 
            rel="noopener noreferrer"
            style={{ fontWeight: 'bold', textDecoration: 'underline' }}
          >
            {configuration?.general?.generatedPath ? 
              configuration.general.generatedPath.substring(0, configuration.general.generatedPath.lastIndexOf('/')) + '/temp' : 
              './temp'}
          </a>
        </p>
      </ModalComponent>
    );
  }

  return (
    <ModalComponent
      show
      icon={faCut}
      header={intl.formatMessage({ id: "dialogs.trim_video.title" })}
      accept={{
        onClick: handleSubmit,
        text: intl.formatMessage({ id: "actions.trim_video" }),
      }}
      cancel={{
        onClick: onClose,
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
      }}
      isRunning={isProcessing}
    >
      <Form>
        {hasTrimTimes && (
          <Alert variant="warning">
            <strong>Warning:</strong> <FormattedMessage id="dialogs.trim_video.warning_text" />
          </Alert>
        )}

        {!hasTrimTimes && (
          <Alert variant="warning">
            <FormattedMessage id="dialogs.trim_video.disabled_message" />
          </Alert>
        )}


        {scene.files.length > 1 && (
          <Form.Group controlId="file-select" as={Row}>
            <Form.Label column sm={3}>
              <FormattedMessage id="file" />
            </Form.Label>
            <Col sm={9}>
              <Form.Control
                as="select"
                value={selectedFileId}
                onChange={(e) => setSelectedFileId(e.target.value)}
                className="input-control"
                disabled={!hasTrimTimes}
              >
                {scene.files.map((file) => (
                  <option key={file.id} value={file.id}>
                    {file.path ? file.path.split('/').pop() : 'Unknown file'} ({file.width}x{file.height} -{" "}
                    {formatFileSize(file.size)})
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
            {selectedFile.path ? selectedFile.path.split('/').pop() : 'Unknown file'}
            <br />
            <strong>
              <FormattedMessage id="dialogs.trim_video.original_duration" />:{" "}
            </strong>
            {formatTime(originalDuration)}
            <br />
            <strong>
              <FormattedMessage id="size" />:{" "}
            </strong>
            {formatFileSize(selectedFile.size)}
          </Alert>
        )}

        <Form.Group controlId="start-time" as={Row}>
          <Form.Label column sm={3}>
            <FormattedMessage id="dialogs.trim_video.start_time" />
          </Form.Label>
          <Col sm={9}>
            <DurationInput
              value={startTime}
              setValue={setStartTime}
              disabled={!hasTrimTimes}
            />
          </Col>
        </Form.Group>

        <Form.Group controlId="end-time" as={Row}>
          <Form.Label column sm={3}>
            <FormattedMessage id="dialogs.trim_video.end_time" />
          </Form.Label>
          <Col sm={9}>
            <DurationInput
              value={endTime}
              setValue={setEndTime}
              disabled={!hasTrimTimes}
            />
          </Col>
        </Form.Group>

        {startTime !== null && endTime !== null && (
          <Alert variant="info">
            <strong>
              <FormattedMessage id="dialogs.trim_video.trim_duration" />:{" "}
            </strong>
            {formatTime(endTime - startTime)}
            <br />
            <strong>
              <FormattedMessage id="dialogs.trim_video.original_duration" />:{" "}
            </strong>
            {formatTime(originalDuration)}
          </Alert>
        )}

      </Form>
    </ModalComponent>
  );
};
