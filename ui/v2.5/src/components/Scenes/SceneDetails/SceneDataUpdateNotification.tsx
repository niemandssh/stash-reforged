import React from "react";
import { Button, Alert } from "react-bootstrap";
import { FormattedMessage } from "react-intl";
import { Icon } from "src/components/Shared/Icon";
import { faSync } from "@fortawesome/free-solid-svg-icons";

interface ISceneDataUpdateNotificationProps {
  onRefresh: () => void;
  visible: boolean;
}

export const SceneDataUpdateNotification: React.FC<
  ISceneDataUpdateNotificationProps
> = ({ onRefresh, visible }) => {
  if (!visible) {
    return null;
  }

  return (
    <Alert
      variant="warning"
      className="mb-2 scene-data-update-notification"
      style={{
        backgroundColor: "rgb(255 207 56)",
        color: "rgb(133, 100, 4)",
        padding: "4px 4px",
        margin: "0px 0px 4px",
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        border: "none",
        fontSize: "12px",
        borderRadius: "4px",
      }}
    >
      <div style={{ display: "flex", alignItems: "center" }}>
        <Icon icon={faSync} className="me-2" />
        <FormattedMessage
          id="scene_data_updated_notification"
          defaultMessage="Данные сцены обновлены, нажмите здесь для обновления"
        />
      </div>
      <Button
        variant="outline-warning"
        size="sm"
        onClick={onRefresh}
        style={{
          borderColor: "rgb(133, 100, 4)",
          color: "rgb(133, 100, 4)",
          backgroundColor: "transparent",
          fontSize: "12px",
          padding: "2px 8px",
        }}
      >
        <FormattedMessage id="actions.refresh" defaultMessage="Обновить" />
      </Button>
    </Alert>
  );
};

export default SceneDataUpdateNotification;
