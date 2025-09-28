import React from "react";
import { Link, useHistory } from "react-router-dom";
import { useIntl } from "react-intl";
import TextUtils from "src/utils/text";
import { ScenePreview } from "../Scenes/SceneCard";
import { SweatDrops } from "../Shared/SweatDrops";
import { HLSBadge } from "../Shared/HLSBadge";
import { BrokenBadge } from "../Shared/BrokenBadge";
import { ProbablyBrokenBadge } from "../Shared/ProbablyBrokenBadge";
import GenderIcon from "../Performers/GenderIcon";
import { PerformerPopover } from "../Performers/PerformerPopover";
import { StudioOverlay } from "../Shared/GridCard/StudioOverlay";
import { Scene } from "./types";
import "./ViewHistoryCard.scss";

interface ViewHistoryCardProps {
  scene: Scene;
  viewDate: string;
  oDate?: string;
  viewCount?: number;
}

export const ViewHistoryCard: React.FC<ViewHistoryCardProps> = ({
  scene,
  viewDate,
  oDate,
  viewCount,
}) => {
  const intl = useIntl();
  const history = useHistory();
  const file = scene.files?.[0];
  const duration = file?.duration || 0;
  const scenePath = `/scenes/${scene.id}`;

  console.log(`ViewHistoryCard for scene ${scene.id}: viewCount = ${viewCount}`);

  const formatViewDate = (date: string) => {
    const viewDateTime = new Date(date);
    const now = new Date();
    const diffMs = now.getTime() - viewDateTime.getTime();
    const diffHours = diffMs / (1000 * 60 * 60);
    const diffDays = diffMs / (1000 * 60 * 60 * 24);

    if (diffHours < 1) {
      const diffMinutes = Math.floor(diffMs / (1000 * 60));
      return `${diffMinutes} minutes ago`;
    } else if (diffHours < 24) {
      const hours = Math.floor(diffHours);
      return `${hours} hour${hours > 1 ? "s" : ""} ago`;
    } else if (diffDays < 7) {
      const days = Math.floor(diffDays);
      return `${days} day${days > 1 ? "s" : ""} ago`;
    } else {
      return viewDateTime.toLocaleDateString();
    }
  };

  const thumbnailUrl = scene.paths?.screenshot || "";
  const previewUrl = scene.paths?.preview || "";
  const vttPath = scene.paths?.vtt || "";

  return (
    <div className="view-history-card card scene-card">
      <Link to={scenePath} className="view-history-thumbnail-container">
        <ScenePreview
          image={thumbnailUrl}
          video={previewUrl}
          vttPath={vttPath}
          isPortrait={false}
          soundActive={false}
          onScrubberClick={(timestamp) => {
            // Navigate to scene at specific timestamp
            history.push(`/scenes/${scene.id}?t=${timestamp}`);
          }}
        />
        {scene.resume_time && scene.resume_time > 0 && duration > 0 && (
          <div title={Math.round((scene.resume_time / duration) * 100) + "%"} className="progress-bar">
            <div style={{ width: `${(scene.resume_time / duration) * 100}%` }} className="progress-indicator" />
          </div>
        )}
        {duration > 0 && (
          <span className="view-history-duration">
            {TextUtils.secondsToTimestamp(duration)}
          </span>
        )}
      </Link>

      <div className="view-history-content">
        <Link to={scenePath} className="view-history-title-link">
          <h3 className="view-history-title">
            {scene.title || TextUtils.fileNameFromPath(file?.path || "")}
            {viewCount && viewCount > 1 && (
              <span className="view-history-view-count-chip">
                {intl.formatMessage({ id: "consecutive_views" }, { count: viewCount })}
              </span>
            )}
            {scene.force_hls && (
              <span className="view-history-status-chip">
                <HLSBadge />
              </span>
            )}
            {scene.is_broken && !scene.is_not_broken && (
              <span className="view-history-status-chip">
                <BrokenBadge />
              </span>
            )}
            {!scene.is_broken && scene.is_probably_broken && !scene.is_not_broken && (
              <span className="view-history-status-chip">
                <ProbablyBrokenBadge />
              </span>
            )}
          </h3>
        </Link>

        {scene.studio && (
          <StudioOverlay studio={scene.studio} />
        )}

        <div className="view-history-bottom-info">
          <div className="view-history-view-info">
            <span className="view-history-view-count">
              {scene.play_count || 0} view{(scene.play_count || 0) !== 1 ? "s" : ""}
            </span>
            <span className="view-history-separator">•</span>
            <span className="view-history-view-date">
              {formatViewDate(viewDate)}
            </span>
          </div>

          {scene.performers && scene.performers.length > 0 && (
            <div className="view-history-performers">
              {scene.performers.slice(0, 3).map((performer) => (
                <PerformerPopover key={performer.id} id={performer.id}>
                  <Link
                    to={`/performers/${performer.id}`}
                    className="view-history-performer-chip"
                  >
                    <GenderIcon gender={performer.gender as any} className="view-history-gender-icon" />
                    {performer.name}
                  </Link>
                </PerformerPopover>
              ))}
              {scene.performers.length > 3 && (
                <span className="view-history-more-performers">
                  +{scene.performers.length - 3} more
                </span>
              )}
            </div>
          )}
        </div>
      </div>

      {oDate && (
        <div className="view-history-o-count-indicator" title={`O-Count: ${new Date(oDate).toLocaleString()}`}>
          <SweatDrops />
        </div>
      )}
    </div>
  );
};