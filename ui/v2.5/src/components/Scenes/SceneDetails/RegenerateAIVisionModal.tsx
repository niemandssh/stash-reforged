import React, { useState } from "react";
import { Alert } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { faEye } from "@fortawesome/free-solid-svg-icons";
import { ModalComponent } from "src/components/Shared/Modal";
import { useSceneRegenerateAIVision } from "src/core/StashService";

interface IRegenerateAIVisionModalProps {
  sceneId: string;
  show: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export const RegenerateAIVisionModal: React.FC<
  IRegenerateAIVisionModalProps
> = ({ sceneId, show, onClose, onSuccess }) => {
  const intl = useIntl();
  const [isProcessing, setIsProcessing] = useState(false);

  const [regenerateAIVision] = useSceneRegenerateAIVision();

  const handleSubmit = async () => {
    setIsProcessing(true);
    try {
      await regenerateAIVision({
        variables: {
          id: sceneId,
        },
      });

      if (onSuccess) {
        onSuccess();
      }
      onClose();
    } catch (error) {
      console.error("Error regenerating AI vision panels:", error);
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <ModalComponent
      show={show}
      icon={faEye}
      header={intl.formatMessage({ id: "dialogs.regenerate_ai_vision.title" })}
      accept={{
        text: intl.formatMessage({ id: "actions.regenerate" }),
        variant: "danger",
        onClick: handleSubmit,
      }}
      cancel={{
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
        onClick: onClose,
      }}
      isRunning={isProcessing}
    >
      <Alert variant="warning">
        <FormattedMessage id="dialogs.regenerate_ai_vision.warning" />
      </Alert>
      <p>
        <FormattedMessage id="dialogs.regenerate_ai_vision.confirm_message" />
      </p>
    </ModalComponent>
  );
};
