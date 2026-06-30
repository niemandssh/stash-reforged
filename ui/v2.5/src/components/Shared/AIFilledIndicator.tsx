import React from "react";
import { OverlayTrigger, Tooltip } from "react-bootstrap";
import { useIntl } from "react-intl";
import { faRobot } from "@fortawesome/free-solid-svg-icons";
import { Icon } from "./Icon";

interface IAIFilledIndicatorProps {
  className?: string;
}

export const AIFilledIndicator: React.FC<IAIFilledIndicatorProps> = ({
  className = "",
}) => {
  const intl = useIntl();

  return (
    <OverlayTrigger
      overlay={
        <Tooltip id="ai-filled-indicator-tooltip">
          {intl.formatMessage({ id: "is_ai_filled" })}
        </Tooltip>
      }
      placement="top"
    >
      <span
        className={`ai-filled-indicator ${className}`}
        style={{
          color: "#7ec8ff",
          fontSize: "1.1rem",
          cursor: "default",
        }}
      >
        <Icon icon={faRobot} />
      </span>
    </OverlayTrigger>
  );
};

export default AIFilledIndicator;
