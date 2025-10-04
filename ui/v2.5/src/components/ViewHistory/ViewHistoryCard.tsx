import React from "react";
import { Link, useHistory } from "react-router-dom";
import { useIntl } from "react-intl";
import TextUtils from "src/utils/text";
import { ScenePreview } from "../Scenes/SceneCard";
import { GalleryPreview } from "../Galleries/GalleryCard";
import { SweatDrops } from "../Shared/SweatDrops";
import { HLSBadge } from "../Shared/HLSBadge";
import { BrokenBadge } from "../Shared/BrokenBadge";
import { ProbablyBrokenBadge } from "../Shared/ProbablyBrokenBadge";
import GenderIcon from "../Performers/GenderIcon";
import { PerformerPopover } from "../Performers/PerformerPopover";
import { StudioOverlay } from "../Shared/GridCard/StudioOverlay";
import { Icon } from "../Shared/Icon";
import { faImage } from "@fortawesome/free-solid-svg-icons";
import { Scene, Gallery } from "./types";
import NavUtils from "src/utils/navigation";
import "./ViewHistoryCard.scss";

interface ViewHistoryCardProps {
  scene?: Scene;
  gallery?: Gallery;
  viewDate: string;
  oDate?: string;
  viewCount?: number;
}

export const ViewHistoryCard: React.FC<ViewHistoryCardProps> = ({
  scene,
  gallery,
  viewDate,
  oDate,
  viewCount,
}) => {
  const intl = useIntl();
  const history = useHistory();

  // Determine content type and set variables accordingly
  const isScene = !!scene;
  const content = scene || gallery;
  const contentPath = isScene ? `/scenes/${scene!.id}` : NavUtils.makeGalleryUrl(gallery!);
  const file = isScene ? scene!.files?.[0] : gallery!.files?.[0];

  console.log(`ViewHistoryCard for ${isScene ? 'scene' : 'gallery'} ${content!.id}: viewCount = ${viewCount}`);

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

  const renderPreview = () => {
    if (isScene) {
      const thumbnailUrl = scene!.paths?.screenshot || "";
      const previewUrl = scene!.paths?.preview || "";
      const vttPath = scene!.paths?.vtt || "";
      const duration = scene!.files?.[0]?.duration || 0;

      return (
        <>
          <ScenePreview
            image={thumbnailUrl}
            video={previewUrl}
            vttPath={vttPath}
            isPortrait={false}
            soundActive={false}
            onScrubberClick={(timestamp) => {
              // Navigate to scene at specific timestamp
              history.push(`/scenes/${scene!.id}?t=${timestamp}`);
            }}
          />
          {scene!.resume_time && scene!.resume_time > 0 && duration > 0 && (
            <div title={Math.round((scene!.resume_time / duration) * 100) + "%"} className="progress-bar">
              <div style={{ width: `${(scene!.resume_time / duration) * 100}%` }} className="progress-indicator" />
            </div>
          )}
          {duration > 0 && (
            <span className="view-history-duration">
              {TextUtils.secondsToTimestamp(duration)}
            </span>
          )}
        </>
      );
    } else {
      // For gallery, use GalleryPreview with cover image
      const slimGallery = {
        id: gallery!.id,
        title: gallery!.title,
        code: gallery!.code,
        date: gallery!.date,
        urls: [],
        details: gallery!.details,
        photographer: gallery!.photographer,
        rating100: gallery!.rating100,
        organized: gallery!.organized || false,
        pinned: gallery!.pinned || false,
        o_counter: gallery!.o_counter || 0,
        display_mode: gallery!.display_mode || 0,
        paths: {
          cover: gallery!.paths?.cover || "",
          preview: gallery!.paths?.preview || gallery!.paths?.cover || "",
        },
        image_count: gallery!.image_count || 0,
        files: gallery!.files?.map(file => ({
          id: file.id,
          path: file.path,
          size: file.size,
          mod_time: file.mod_time,
          fingerprints: file.fingerprints.map(fp => ({
            type: fp.type,
            value: fp.value,
          })),
        })) || [],
        chapters: [],
        scenes: [],
        studio: gallery!.studio && gallery!.studio.name ? {
          id: gallery!.studio.id,
          name: gallery!.studio.name,
          image_path: gallery!.studio.image_path,
        } : null,
        tags: [],
        performers: gallery!.performers?.map(performer => ({
          id: performer.id,
          name: performer.name || "",
          gender: performer.gender as any,
          favorite: false,
          image_path: null,
        })) || [],
      };

      return (
        <>
          <GalleryPreview
            gallery={slimGallery}
            onScrubberClick={(index) => {
              // Navigate to gallery at specific image
              history.push(`/galleries/${gallery!.id}?index=${index}`);
            }}
          />
          {gallery!.image_count && gallery!.image_count > 0 && (
            <div className="view-history-image-count">
              <Icon icon={faImage} />
              <span>{gallery!.image_count}</span>
            </div>
          )}
        </>
      );
    }
  };

  const getTitle = () => {
    if (isScene) {
      return scene!.title || TextUtils.fileNameFromPath(file?.path || "");
    } else {
      return gallery!.title || TextUtils.fileNameFromPath(file?.path || "");
    }
  };

  const getViewCount = () => {
    if (isScene) {
      return scene!.play_count || 0;
    } else {
      return gallery!.play_count || 0;
    }
  };

  const getViewCountText = () => {
    const count = getViewCount();
    return `${count} view${count !== 1 ? "s" : ""}`;
  };

  return (
    <div className={`view-history-card card ${isScene ? 'scene-card' : 'gallery-card'}`}>
      <Link to={contentPath} className="view-history-thumbnail-container">
        {renderPreview()}
      </Link>

      <div className="view-history-content">
        <Link to={contentPath} className="view-history-title-link">
          <h3 className="view-history-title">
            {getTitle()}
            {viewCount && viewCount > 1 && (
              <span className="view-history-view-count-chip">
                {intl.formatMessage({ id: "consecutive_views" }, { count: viewCount })}
              </span>
            )}
            {isScene && scene!.force_hls && (
              <span className="view-history-status-chip">
                <HLSBadge />
              </span>
            )}
            {isScene && scene!.is_broken && !scene!.is_not_broken && (
              <span className="view-history-status-chip">
                <BrokenBadge />
              </span>
            )}
            {isScene && !scene!.is_broken && scene!.is_probably_broken && !scene!.is_not_broken && (
              <span className="view-history-status-chip">
                <ProbablyBrokenBadge />
              </span>
            )}
          </h3>
        </Link>

        {content!.studio && content!.studio.name && (
          <StudioOverlay studio={content!.studio as any} />
        )}

        <div className="view-history-bottom-info">
          <div className="view-history-view-info">
            <span className="view-history-view-count">
              {getViewCountText()}
            </span>
            <span className="view-history-separator">•</span>
            <span className="view-history-view-date">
              {formatViewDate(viewDate)}
            </span>
          </div>

          {content!.performers && content!.performers.length > 0 && (
            <div className="view-history-performers">
              {content!.performers.slice(0, 3).map((performer) => (
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
              {content!.performers.length > 3 && (
                <span className="view-history-more-performers">
                  +{content!.performers.length - 3} more
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
