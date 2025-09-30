import React, { useState, useEffect, useCallback } from "react";
import { Link } from "react-router-dom";
import { FormattedMessage } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { ScenePreview } from "../Scenes/SceneCard";
import { Icon } from "../Shared/Icon";
import { ConfigurationContext } from "../../hooks/Config";
import GenderIcon from "../Performers/GenderIcon";
import { faPlay, faTimes } from "@fortawesome/free-solid-svg-icons";
import "./NextSceneOverlay.scss";

interface NextSceneOverlayProps {
  nextScene: GQL.SlimSceneDataFragment;
  onPlay: () => void;
  onCancel: () => void;
  onSkip: () => void;
}

export const NextSceneOverlay: React.FC<NextSceneOverlayProps> = ({
  nextScene,
  onPlay,
  onCancel,
  onSkip,
}) => {
  const { configuration } = React.useContext(ConfigurationContext);
  const autoplayTimer = configuration?.interface?.autoplayNextVideoTimer || 10;
  const hasAutoplay = autoplayTimer > 0;
  const [timeLeft, setTimeLeft] = useState(hasAutoplay ? autoplayTimer : 0);
  const [timerCancelled, setTimerCancelled] = useState(false);

  useEffect(() => {
    if (!hasAutoplay || timerCancelled || timeLeft <= 0) {
      if (timeLeft <= 0 && hasAutoplay) {
        onPlay();
      }
      return;
    }

    const timer = setTimeout(() => {
      setTimeLeft(timeLeft - 1);
    }, 1000);

    return () => clearTimeout(timer);
  }, [timeLeft, onPlay, timerCancelled, hasAutoplay]);

  const handleSkip = useCallback(() => {
    onSkip();
  }, [onSkip]);

  const handleCancel = useCallback(() => {
    setTimerCancelled(true);
  }, []);

  const file = nextScene.files.length > 0 ? nextScene.files[0] : undefined;
  const duration = file?.duration ?? 0;
  const scenePath = `/scenes/${nextScene.id}`;

  return (
    <div className="next-scene-overlay">
      <div className="next-scene-overlay-background" onClick={onCancel} />

      <div className="next-scene-overlay-content">
        <button
          className="next-scene-overlay-close"
          onClick={onCancel}
          title="Close"
        >
          <Icon icon={faTimes} />
        </button>

        <div className="next-scene-overlay-main">
          <div className="next-scene-overlay-preview">
            <Link to={scenePath} onClick={handleSkip}>
              <ScenePreview
                image={nextScene.paths.screenshot ?? undefined}
                video={nextScene.paths.preview ?? undefined}
                isPortrait={false}
                soundActive={false}
              />
            </Link>

            {duration > 0 && (
              <span className="next-scene-overlay-duration">
                {Math.floor(duration / 60)}:{Math.floor(duration % 60).toString().padStart(2, '0')}
              </span>
            )}
          </div>

          <div className="next-scene-overlay-info">
            <h2 className="next-scene-overlay-title">
              <Link to={scenePath} onClick={handleSkip}>
                {nextScene.title || "Untitled Scene"}
              </Link>
            </h2>

            <div className="next-scene-overlay-meta">
              {nextScene.date && (
                <span className="next-scene-overlay-date">{nextScene.date}</span>
              )}

              {nextScene.studio && nextScene.date && (
                <span className="next-scene-overlay-separator">•</span>
              )}

              {nextScene.studio && (
                <Link
                  to={`/studios/${nextScene.studio.id}`}
                  className="next-scene-overlay-studio"
                  onClick={handleSkip}
                >
                  {nextScene.studio.image_path ? (
                    <img
                      src={nextScene.studio.image_path}
                      alt={nextScene.studio.name}
                      className="next-scene-overlay-studio-image"
                    />
                  ) : (
                    <span className="next-scene-overlay-studio-text">
                      {nextScene.studio.name}
                    </span>
                  )}
                </Link>
              )}
            </div>

            {nextScene.performers && nextScene.performers.length > 0 && (
              <div className="next-scene-overlay-performers">
                {nextScene.performers.slice(0, 3).map((performer) => (
                  <Link
                    key={performer.id}
                    to={`/performers/${performer.id}`}
                    className="next-scene-overlay-performer-chip"
                    onClick={handleSkip}
                  >
                    <GenderIcon gender={performer.gender as any} className="next-scene-overlay-gender-icon" />
                    {performer.name}
                  </Link>
                ))}
                {nextScene.performers.length > 3 && (
                  <span className="next-scene-overlay-more-performers">
                    +{nextScene.performers.length - 3} more
                  </span>
                )}
              </div>
            )}

            <div className="next-scene-overlay-actions">
              <button
                className="btn btn-primary next-scene-overlay-play-btn"
                onClick={handleSkip}
              >
                <Icon icon={faPlay} />
                <FormattedMessage id="actions.play" defaultMessage="Play" />
              </button>

              {!timerCancelled && autoplayTimer > 0 && (
                <button
                  className="btn btn-secondary next-scene-overlay-cancel-btn"
                  onClick={handleCancel}
                >
                  <FormattedMessage id="actions.cancel" defaultMessage="Cancel" />
                </button>
              )}
            </div>

            <div
              className="next-scene-overlay-countdown"
              style={{ opacity: (!timerCancelled && autoplayTimer > 0) ? 1 : 0 }}
            >
              <FormattedMessage
                id="next_scene_autoplay"
                defaultMessage="Next scene in {seconds}s"
                values={{ seconds: timeLeft }}
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
