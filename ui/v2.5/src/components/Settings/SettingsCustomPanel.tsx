import React from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { SettingSection } from "./SettingSection";
import { NumberSetting } from "./Inputs";
import { useSettings } from "./context";

export const SettingsCustomPanel: React.FC = () => {
  const intl = useIntl();
  const { interface: iface, saveInterface } = useSettings();

  return (
    <div className="settings-custom-panel">
      <SettingSection headingID="config.custom.rating_thresholds" headingDefault="Rating Thresholds">
        <NumberSetting
          id="random-rating-threshold"
          headingID="config.custom.random_rating_threshold"
          subHeadingID="config.custom.random_rating_threshold_desc"
          value={iface.randomRatingThreshold ?? 55}
          onChange={(v) => saveInterface({ randomRatingThreshold: v })}
          min={0}
          max={100}
        />
        <NumberSetting
          id="random-best-rating-threshold"
          headingID="config.custom.random_best_rating_threshold"
          subHeadingID="config.custom.random_best_rating_threshold_desc"
          value={iface.randomBestRatingThreshold ?? 90}
          onChange={(v) => saveInterface({ randomBestRatingThreshold: v })}
          min={0}
          max={100}
        />
      </SettingSection>
    </div>
  );
};
