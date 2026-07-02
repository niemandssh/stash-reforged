import React, { useContext, useMemo } from "react";
import * as GQL from "src/core/generated-graphql";
import { ConfigurationContext } from "src/hooks/Config";
import { buildEmbedPlaybackUrl } from "./embedPlayback";
import "./SceneEmbedPlayer.scss";

interface ISceneEmbedPlayerProps {
  scene: GQL.SceneDataFragment;
  embedSrc: string;
  initialTimestamp?: number;
}

export const SceneEmbedPlayer: React.FC<ISceneEmbedPlayerProps> = ({
  scene,
  embedSrc,
  initialTimestamp = 0,
}) => {
  const { configuration } = useContext(ConfigurationContext);
  const uiConfig = configuration?.interface;

  const iframeSrc = useMemo(
    () =>
      buildEmbedPlaybackUrl({
        embedSrc,
        startTime: scene.start_time,
        endTime: scene.end_time,
        resumeTime: scene.resume_time,
        lastPlayedAt: scene.last_played_at,
        trimEnabled: true,
        initialTimestamp,
        alwaysStartFromBeginning: uiConfig?.alwaysStartFromBeginning ?? false,
        rememberPlayPositionFor5Minutes:
          uiConfig?.rememberPlayPositionFor5Minutes ?? false,
      }),
    [
      embedSrc,
      scene.start_time,
      scene.end_time,
      scene.resume_time,
      scene.last_played_at,
      initialTimestamp,
      uiConfig?.alwaysStartFromBeginning,
      uiConfig?.rememberPlayPositionFor5Minutes,
    ]
  );

  return (
    <div className="SceneEmbedPlayer">
      <div className="embed-wrapper">
        <iframe
          key={iframeSrc}
          title={scene.title ?? `Scene ${scene.id}`}
          src={iframeSrc}
          allow="autoplay; fullscreen; encrypted-media; picture-in-picture"
          allowFullScreen
          referrerPolicy="no-referrer-when-downgrade"
        />
      </div>
    </div>
  );
};

export default SceneEmbedPlayer;
