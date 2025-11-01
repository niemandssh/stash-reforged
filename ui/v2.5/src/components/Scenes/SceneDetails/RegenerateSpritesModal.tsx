import React, { useState } from "react";
import { Alert } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { faImages } from "@fortawesome/free-solid-svg-icons";
import { ModalComponent } from "src/components/Shared/Modal";
import { useSceneRegenerateSprites } from "src/core/StashService";

interface IRegenerateSpritesModalProps {
  sceneId: string;
  show: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export const RegenerateSpritesModal: React.FC<IRegenerateSpritesModalProps> = ({
  sceneId,
  show,
  onClose,
  onSuccess,
}) => {
  const intl = useIntl();
  const [isProcessing, setIsProcessing] = useState(false);

  const [regenerateSprites] = useSceneRegenerateSprites();

  const handleSubmit = async () => {
    setIsProcessing(true);
    try {
      await regenerateSprites({
        variables: {
          id: sceneId,
        },
      });

      if (onSuccess) {
        onSuccess();
      }
      onClose();
    } catch (error) {
      console.error("Error regenerating sprites:", error);
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <ModalComponent
      show={show}
      icon={faImages}
      header={intl.formatMessage({ id: "dialogs.regenerate_sprites.title" })}
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
        <FormattedMessage id="dialogs.regenerate_sprites.warning" />
      </Alert>
      <p>
        <FormattedMessage id="dialogs.regenerate_sprites.confirm_message" />
      </p>
    </ModalComponent>
  );
};
