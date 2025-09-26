import React from "react";
import { SettingSection } from "./SettingSection";
import { NumberSetting, BooleanSetting, StringSetting } from "./Inputs";
import { useSettings } from "./context";

export const SettingsCustomPanel: React.FC = () => {
  const { interface: iface, saveInterface } = useSettings();

  return (
    <div className="settings-custom-panel">
      <SettingSection headingID="config.custom.rating_thresholds">
        <NumberSetting
          id="random-rating-threshold"
          headingID="config.custom.random_rating_threshold"
          subHeadingID="config.custom.random_rating_threshold_desc"
          value={iface.randomRatingThreshold ?? 55}
          onChange={(v) => saveInterface({ randomRatingThreshold: v })}
        />
        <NumberSetting
          id="random-best-rating-threshold"
          headingID="config.custom.random_best_rating_threshold"
          subHeadingID="config.custom.random_best_rating_threshold_desc"
          value={iface.randomBestRatingThreshold ?? 90}
          onChange={(v) => saveInterface({ randomBestRatingThreshold: v })}
        />
      </SettingSection>
      
      <SettingSection headingID="config.custom.similar_scenes">
        <BooleanSetting
          id="show-similarity-percent"
          headingID="config.custom.show_similarity_percent"
          subHeadingID="config.custom.show_similarity_percent_desc"
          checked={iface.showSimilarityPercent ?? true}
          onChange={(v) => saveInterface({ showSimilarityPercent: v })}
        />
      </SettingSection>

      <SettingSection headingID="config.custom.external_player">
        <StringSetting
          id="external-video-player"
          headingID="config.custom.external_video_player"
          subHeadingID="config.custom.external_video_player_desc"
          value={iface.externalVideoPlayer ?? ""}
          onChange={(v) => saveInterface({ externalVideoPlayer: v })}
        />
      </SettingSection>

      <SettingSection headingID="config.custom.redirect">
        <BooleanSetting
          id="redirect-home-to-scenes"
          headingID="config.custom.redirect_home_to_scenes"
          subHeadingID="config.custom.redirect_home_to_scenes_desc"
          checked={iface.redirectHomeToScenes ?? false}
          onChange={(v) => saveInterface({ redirectHomeToScenes: v })}
        />
      </SettingSection>
      </div>
  );
};
