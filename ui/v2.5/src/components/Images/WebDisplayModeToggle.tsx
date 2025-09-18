import React from "react";
import { WebDisplayMode } from "src/models/list-filter/types";
import { useIntl } from "react-intl";
import { Icon } from "../Shared/Icon";
import { faExpand, faCompress } from "@fortawesome/free-solid-svg-icons";

interface IWebDisplayModeToggleProps {
  currentMode: WebDisplayMode;
  onModeChange: (mode: WebDisplayMode) => void;
}

export const WebDisplayModeToggle: React.FC<IWebDisplayModeToggleProps> = ({
  currentMode,
  onModeChange,
}) => {
  const intl = useIntl();
  
  const handleModeChange = (mode: WebDisplayMode) => {
    onModeChange(mode);
  };

  return (
    <div className="web-display-mode-toggle">
      <button
        className={`web-display-mode-btn ${currentMode === WebDisplayMode.FullSize ? "active" : ""}`}
        onClick={() => handleModeChange(WebDisplayMode.FullSize)}
        title={intl.formatMessage({ id: "web_display_mode.full_size" })}
      >
        <Icon icon={faExpand} />
      </button>
      <button
        className={`web-display-mode-btn ${currentMode === WebDisplayMode.FitToScreen ? "active" : ""}`}
        onClick={() => handleModeChange(WebDisplayMode.FitToScreen)}
        title={intl.formatMessage({ id: "web_display_mode.fit_to_screen" })}
      >
        <Icon icon={faCompress} />
      </button>
    </div>
  );
};
