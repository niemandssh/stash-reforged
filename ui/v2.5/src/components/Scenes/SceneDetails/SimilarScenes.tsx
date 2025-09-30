
import React, { useState, useMemo, useRef } from "react";
import { FormattedMessage } from "react-intl";
import { useQuery } from "@apollo/client";
import * as GQL from "src/core/generated-graphql";
import { useSimilarityJobMonitor } from "src/hooks/useSimilarityJobMonitor";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { ErrorMessage } from "src/components/Shared/ErrorMessage";
import { Button, ButtonGroup } from "react-bootstrap";
import { GridCard } from "src/components/Shared/GridCard/GridCard";
import { ConfigurationContext } from "src/hooks/Config";
import { objectTitle } from "src/core/files";
import { ScenePreview } from "src/components/Scenes/SceneCard";
import { RatingBanner } from "src/components/Shared/RatingBanner";
import { TruncatedText } from "src/components/Shared/TruncatedText";
import { FileSize } from "src/components/Shared/FileSize";
import TextUtils from "src/utils/text";
import { Icon } from "src/components/Shared/Icon";
import { HoverPopover } from "src/components/Shared/HoverPopover";
import { TagLink } from "src/components/Shared/TagLink";
import { PerformerPopoverButton } from "src/components/Shared/PerformerPopoverButton";
import { GroupTag } from "src/components/Groups/GroupTag";
import { faTag, faFilm, faMapMarkerAlt } from "@fortawesome/free-solid-svg-icons";

interface ISimilarScenesProps {
  scene: GQL.SceneDataFragment;
  limit?: number;
}

interface ISimilarSceneCardProps {
  scene: GQL.SlimSceneDataFragment;
  similarityScore: number;
  similarityScoreData?: GQL.SimilarityScoreData | null;
  getSimilarityColor: (score: number) => string;
  getSimilarityText: (score: number) => string;
  getSimilarityTextColor: (score: number) => string;
  showSimilarityPercent: boolean;
}

const SimilarSceneCard: React.FC<ISimilarSceneCardProps> = ({
  scene,
  similarityScore,
  similarityScoreData,
  getSimilarityColor,
  getSimilarityText,
  getSimilarityTextColor,
  showSimilarityPercent
}) => {
  const { configuration } = React.useContext(ConfigurationContext);
  const chipRef = useRef<HTMLDivElement>(null);
  
  const sceneLink = `/scenes/${scene.id}`;
  
  const file = useMemo(
    () => (scene.files.length > 0 ? scene.files[0] : undefined),
    [scene]
  );

  function zoomIndex() {
    return "";
  }

  function filelessClass() {
    if (!scene.files.length) {
      return "fileless";
    }
    return "";
  }

  function isPortrait() {
    const width = file?.width ? file.width : 0;
    const height = file?.height ? file.height : 0;
    return height > width;
  }

  function maybeRenderSceneSpecsOverlay() {
    return (
      <div className="scene-specs-overlay">
        {file?.size !== undefined ? (
          <span className="overlay-filesize extra-scene-info">
            <FileSize size={file.size} />
          </span>
        ) : (
          ""
        )}
        {file?.width && file?.height ? (
          <span className="overlay-resolution">
            {" "}
            {TextUtils.resolution(file?.width, file?.height)}
          </span>
        ) : (
          ""
        )}
        {(file?.duration ?? 0) >= 1 ? (
          <span className="overlay-duration">
            {TextUtils.secondsToTimestamp(file?.duration ?? 0)}
          </span>
        ) : (
          ""
        )}
      </div>
    );
  }

  function maybeRenderInteractiveSpeedOverlay() {
    return (
      <div className="scene-interactive-speed-overlay">
        {scene.interactive_speed ?? ""}
      </div>
    );
  }

  function maybeRenderTagPopoverButton() {
    if (scene.tags.length <= 0) return;

    const popoverContent = scene.tags.map((tag) => (
      <TagLink key={tag.id} tag={tag} linkType="details" />
    ));

    return (
      <HoverPopover
        className="tag-count"
        placement="bottom"
        content={popoverContent}
      >
        <Button className="minimal">
          <Icon icon={faTag} />
          <span>{scene.tags.length}</span>
        </Button>
      </HoverPopover>
    );
  }

  function maybeRenderPerformerPopoverButton() {
    if (scene.performers.length <= 0) return;

    return (
      <PerformerPopoverButton
        performers={scene.performers}
        linkType="scene"
      />
    );
  }

  function maybeRenderGroupPopoverButton() {
    if (scene.groups.length <= 0) return;

    const popoverContent = scene.groups.map((sceneGroup) => (
      <GroupTag key={sceneGroup.group.id} group={sceneGroup.group} />
    ));

    return (
      <HoverPopover
        placement="bottom"
        content={popoverContent}
        className="group-count tag-tooltip"
      >
        <Button className="minimal">
          <Icon icon={faFilm} />
          <span>{scene.groups.length}</span>
        </Button>
      </HoverPopover>
    );
  }

  function maybeRenderSceneMarkerPopoverButton() {
    if (scene.scene_markers.length <= 0) return;

    const popoverContent = scene.scene_markers.map((marker) => (
      <div key={marker.id} className="d-flex justify-content-between">
        <span>{marker.title}</span>
        <span className="scene-marker-duration">
          {TextUtils.secondsToTimestamp(marker.seconds)}
        </span>
      </div>
    ));

    return (
      <HoverPopover
        placement="bottom"
        content={popoverContent}
        className="marker-count tag-tooltip"
      >
        <Button className="minimal">
          <Icon icon={faMapMarkerAlt} />
          <span>{scene.scene_markers.length}</span>
        </Button>
      </HoverPopover>
    );
  }

  function maybeRenderPopoverButtonGroup() {
    if (
      scene.tags.length > 0 ||
      scene.performers.length > 0 ||
      scene.groups.length > 0 ||
      scene.scene_markers.length > 0
    ) {
      return (
        <React.Fragment key="popover-button-group">
          <hr />
          <ButtonGroup className="card-popovers">
            {maybeRenderTagPopoverButton()}
            {maybeRenderPerformerPopoverButton()}
            {maybeRenderGroupPopoverButton()}
            {maybeRenderSceneMarkerPopoverButton()}
          </ButtonGroup>
        </React.Fragment>
      );
    }
  }

  return (
    <GridCard
      className={`scene-card ${zoomIndex()} ${filelessClass()}`}
      url={sceneLink}
      title={objectTitle(scene)}
      linkClassName="scene-card-link"
      thumbnailSectionClassName="video-section"
      resumeTime={scene.resume_time ?? undefined}
      duration={file?.duration ?? undefined}
      image={
        <>
          <ScenePreview
            image={scene.paths.screenshot ?? undefined}
            video={scene.paths.preview ?? undefined}
            isPortrait={isPortrait()}
            soundActive={configuration?.interface?.soundOnPreview ?? false}
            vttPath={scene.paths.vtt ?? undefined}
          />
          <RatingBanner rating={scene.rating100} />
          {maybeRenderSceneSpecsOverlay()}
          {maybeRenderInteractiveSpeedOverlay()}
        </>
      }
      overlays={
        showSimilarityPercent ? (
          <HoverPopover
            placement="top"
            offset={[10, 10]}
            popoverClassName="similarity-popover-content"
            target={chipRef}
            content={
              similarityScoreData ? (() => {
                const totalContribution = (similarityScoreData.enhanced_tags || 0) +
                  (similarityScoreData.normal_tags || 0) +
                  (similarityScoreData.reduced_tags || 0) +
                  (similarityScoreData.tags || 0) +
                  (similarityScoreData.performers || 0) +
                  (similarityScoreData.groups || 0) +
                  (similarityScoreData.studio || 0);

                return (
                  <div className="p-2" style={{ fontSize: '0.9rem', lineHeight: '1.5' }}>
                    <div className="mb-2"><strong>Similarity Breakdown:</strong></div>
                    {totalContribution > 0 && (
                      <>
                        {similarityScoreData.enhanced_tags !== undefined && similarityScoreData.enhanced_tags > 0 && (
                          <div>Enhanced Tags: <strong>{(similarityScoreData.enhanced_tags * 100).toFixed(1)}%</strong></div>
                        )}
                        {similarityScoreData.normal_tags !== undefined && similarityScoreData.normal_tags > 0 && (
                          <div>Normal Tags: <strong>{(similarityScoreData.normal_tags * 100).toFixed(1)}%</strong></div>
                        )}
                        {similarityScoreData.reduced_tags !== undefined && similarityScoreData.reduced_tags < 0 && (
                          <div>Reduced Tags Penalty: <strong>{(similarityScoreData.reduced_tags * 100).toFixed(1)}%</strong></div>
                        )}
                        {similarityScoreData.tags !== undefined && similarityScoreData.tags > 0 && (
                          <div>Tags: <strong>{(similarityScoreData.tags * 100).toFixed(1)}%</strong></div>
                        )}
                        {similarityScoreData.performers !== undefined && similarityScoreData.performers > 0 && (
                          <div>Performers: <strong>{(similarityScoreData.performers * 100).toFixed(1)}%</strong></div>
                        )}
                        {similarityScoreData.groups !== undefined && similarityScoreData.groups > 0 && (
                          <div>Groups: <strong>{(similarityScoreData.groups * 100).toFixed(1)}%</strong></div>
                        )}
                        {similarityScoreData.studio !== undefined && similarityScoreData.studio > 0 && (
                          <div>Studio: <strong>{(similarityScoreData.studio * 100).toFixed(1)}%</strong></div>
                        )}
                      </>
                    )}
                    {similarityScoreData.penalty !== undefined && similarityScoreData.penalty < 1 && (
                      <div>Penalty: <strong>{(similarityScoreData.penalty * 100).toFixed(0)}%</strong></div>
                    )}
                  </div>
                );
              })() : (
                <div>Similarity score: {getSimilarityText(similarityScore)}</div>
              )
            }
          >
            <div
              ref={chipRef}
              className="similarity-badge px-2 py-1 rounded fw-bold"
              style={{
                position: 'absolute',
                bottom: '12px',
                left: '8px',
                backgroundColor: getSimilarityColor(similarityScore),
                color: getSimilarityTextColor(similarityScore),
                fontSize: '0.8rem',
                zIndex: 10,
                cursor: 'pointer'
              }}
            >
              {getSimilarityText(similarityScore)}
            </div>
          </HoverPopover>
        ) : null
      }
      details={
        <div className="scene-card__details">
          <span className="scene-card__date">{scene.date}</span>
          <span className="file-path extra-scene-info">
            {scene.paths.screenshot}
          </span>
          <TruncatedText
            className="scene-card__description"
            text={scene.details}
            lineCount={3}
          />
        </div>
      }
      popovers={maybeRenderPopoverButtonGroup()}
    />
  );
};

export const SimilarScenes: React.FC<ISimilarScenesProps> = ({ 
  scene, 
  limit = 10 
}) => {
  const [displayLimit, setDisplayLimit] = useState(limit);
  const { configuration } = React.useContext(ConfigurationContext);

  // Use GraphQL query to fetch similar scenes
  const { data, loading, error, refetch } = useQuery<GQL.FindSimilarScenesQuery, GQL.FindSimilarScenesQueryVariables>(
    GQL.FindSimilarScenesDocument,
    {
      variables: { id: scene.id, limit: 100 }, // Fetch more than needed
      skip: !scene.id,
    }
  );

  // Monitor similarity job completion for this scene
  useSimilarityJobMonitor({
    onSimilarityJobComplete: (completedSceneId) => {
      if (completedSceneId === scene.id) {
        console.log(`Similarity job completed for scene ${scene.id}, similar scenes will be refreshed`);
      }
    },
    refetch: refetch
  });

  const allSimilarScenes = data?.findScene?.similar_scenes || [];
  const displayedScenes = allSimilarScenes.slice(0, displayLimit);
  const hasMore = allSimilarScenes.length > displayLimit;

  // Function to get color based on similarity score
  const getSimilarityColor = (score: number) => {
    const percentage = Math.round(Math.min(score, 1.0) * 100);
    if (percentage >= 85) return '#28a745'; // Green
    if (percentage >= 60) return '#ffc107'; // Yellow
    if (percentage >= 40) return '#dc3545'; // Red
    return '#6c757d'; // Gray
  };

  // Function to get text color based on similarity score
  const getSimilarityTextColor = (score: number) => {
    const percentage = Math.round(Math.min(score, 1.0) * 100);
    if (percentage >= 60 && percentage < 85) return '#000000'; // Black for yellow
    return '#ffffff'; // White for other colors
  };

  // Function to get similarity percentage text
  const getSimilarityText = (score: number) => {
    return `${Math.round(score * 100)}%`;
  };

  if (loading) {
    return (
      <div className="similar-scenes mt-3">
        <h4>
          <FormattedMessage id="scene_similar_scenes" defaultMessage="Similar Scenes" />
        </h4>
        <LoadingIndicator />
      </div>
    );
  }

  if (error) {
    return (
      <div className="similar-scenes mt-3">
        <h4>
          <FormattedMessage id="scene_similar_scenes" defaultMessage="Similar Scenes" />
        </h4>
        <ErrorMessage error={error} />
      </div>
    );
  }

  if (allSimilarScenes.length === 0) {
    return (
      <div className="similar-scenes mt-3">
        <h4>
          <FormattedMessage id="scene_similar_scenes" defaultMessage="Similar Scenes" />
        </h4>
        <p className="text-muted">
          <FormattedMessage 
            id="scene_no_similar_scenes" 
            defaultMessage="No similar scenes found" 
          />
        </p>
      </div>
    );
  }

  const handleShowMore = () => {
    setDisplayLimit(prev => prev + limit);
  };

  return (
    <div className="similar-scenes mt-3">
      <style>{`
        .similar-scenes .similarity-badge {
          position: absolute;
          bottom: 8px;
          left: 8px;
          z-index: 10;
        }
      `}</style>
      <div className="mt-3 mb-3">
        <h4>
          <FormattedMessage id="scene_similar_scenes" defaultMessage="Similar Scenes" />
        </h4>
      </div>

      <div className="row">
        {displayedScenes.map((similarScene) => (
          <div key={similarScene.scene.id} className="col-lg-6 col-md-6 col-sm-12 mb-2 px-1">
            <SimilarSceneCard
              scene={similarScene.scene}
              similarityScore={similarScene.similarity_score}
              similarityScoreData={similarScene.similarity_score_data}
              getSimilarityColor={getSimilarityColor}
              getSimilarityText={getSimilarityText}
              getSimilarityTextColor={getSimilarityTextColor}
              showSimilarityPercent={configuration?.interface?.showSimilarityPercent !== undefined ? configuration.interface.showSimilarityPercent : true}
            />
          </div>
        ))}
      </div>

      {hasMore && (
        <div className="text-center mt-3">
          <Button
            variant="outline-light"
            onClick={handleShowMore}
          >
            <FormattedMessage 
              id="actions.show_more" 
              defaultMessage="Show More" 
            />
            <span className="ml-1">
              ({allSimilarScenes.length - displayLimit} more)
            </span>
          </Button>
        </div>
      )}
    </div>
  );
}