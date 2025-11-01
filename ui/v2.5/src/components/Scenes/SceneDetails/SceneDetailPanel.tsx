import React from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link } from "react-router-dom";
import { Button } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import TextUtils from "src/utils/text";
import { TagLink } from "src/components/Shared/TagLink";
import { PerformerCard } from "src/components/Performers/PerformerCard";
import { PerformerPopover } from "src/components/Performers/PerformerPopover";
import { sortPerformers } from "src/core/performers";
import { DirectorLink } from "src/components/Shared/Link";
import { SimilarScenes } from "./SimilarScenes";
import { URLsField } from "src/utils/field";
import { PoseTagsDisplay } from "./PoseTagsDisplay";
import GenderIcon from "src/components/Performers/GenderIcon";
import { useFindColorPresets } from "src/core/StashService";

interface ISceneDetailProps {
  scene: GQL.SceneDataFragment;
}

export const SceneDetailPanel: React.FC<ISceneDetailProps> = (props) => {
  const intl = useIntl();
  const [isDescriptionCollapsed, setIsDescriptionCollapsed] =
    React.useState(true);
  const [shouldShowToggle, setShouldShowToggle] = React.useState(false);
  const textRef = React.useRef<HTMLParagraphElement>(null);

  const { data: presetsData } = useFindColorPresets();
  const colorPresets = presetsData?.findColorPresets?.color_presets || [];

  // Create a map of tags that are already in the scene for quick lookup
  const sceneTagMap = React.useMemo(() => {
    const map = new Map<string, GQL.SlimTagDataFragment>();
    props.scene.tags?.forEach((tag) => map.set(tag.id, tag));
    return map;
  }, [props.scene.tags]);

  // For performer tags, try to find them in scene tags first
  const getTagById = React.useCallback(
    (tagId: string): GQL.SlimTagDataFragment | undefined => {
      // First check scene tags
      return sceneTagMap.get(tagId);
    },
    [sceneTagMap]
  );

  React.useLayoutEffect(() => {
    if (textRef.current && props.scene.details) {
      const tempElement = document.createElement("p");
      tempElement.className = "pre scene-description-text";
      tempElement.style.position = "absolute";
      tempElement.style.visibility = "hidden";
      tempElement.style.width = textRef.current.offsetWidth + "px";
      tempElement.style.whiteSpace = "pre-wrap";
      tempElement.style.wordWrap = "break-word";
      tempElement.style.fontSize = getComputedStyle(textRef.current).fontSize;
      tempElement.style.lineHeight = getComputedStyle(
        textRef.current
      ).lineHeight;
      tempElement.textContent = props.scene.details;

      document.body.appendChild(tempElement);
      const fullHeight = tempElement.offsetHeight;

      const lineHeight =
        parseFloat(getComputedStyle(textRef.current).lineHeight) || 16;
      const threeLinesHeight = lineHeight * 3;

      document.body.removeChild(tempElement);

      setShouldShowToggle(fullHeight > threeLinesHeight);
    }
  }, [props.scene.details]);

  function renderDetails() {
    if (!props.scene.details || props.scene.details === "") return;

    return (
      <>
        <h6 className="font-weight-bold mt-3">
          <FormattedMessage id="details" />:{" "}
        </h6>
        <div className="scene-description">
          <p
            ref={textRef}
            className={`pre scene-description-text ${
              isDescriptionCollapsed ? "scene-description-collapsed" : ""
            }`}
          >
            {props.scene.details}
          </p>
          {shouldShowToggle && (
            <Button
              variant="link"
              size="sm"
              className="scene-description-toggle"
              onClick={() => setIsDescriptionCollapsed(!isDescriptionCollapsed)}
            >
              <FormattedMessage
                id={
                  isDescriptionCollapsed
                    ? "actions.show_more"
                    : "actions.show_less"
                }
              />
            </Button>
          )}
        </div>
      </>
    );
  }

  function renderGeneralTags() {
    // Get all tags from performer_tag_ids to determine which are performer-specific
    const performerSpecificTagIds = new Set<string>();
    if (props.scene.performer_tag_ids) {
      props.scene.performer_tag_ids.forEach((pt: GQL.PerformerTag) => {
        // Only exclude tags that have a specific performer_id (not null)
        if (pt.performer_id && pt.tag_ids) {
          pt.tag_ids.forEach((tagId: string) =>
            performerSpecificTagIds.add(tagId)
          );
        }
      });
    }

    // Filter out performer-specific tags, keep only general tags
    const generalTags = props.scene.tags.filter(
      (tag) => !tag.is_pose_tag && !performerSpecificTagIds.has(tag.id)
    );

    if (generalTags.length === 0) return null;

    // Create a map of colors to presets for quick lookup
    const colorToPreset = new Map<string, GQL.ColorPreset>();
    colorPresets.forEach((preset) => {
      colorToPreset.set(preset.color.toLowerCase(), preset);
    });

    // Sort tags according to requirements:
    // 1. By preset sort order (ascending)
    // 2. If same sort, by preset color (ascending)
    // 3. Tags without color go to the end, sorted alphabetically
    const sortedTags = [...generalTags].sort((a, b) => {
      const aColor = a.color?.toLowerCase();
      const bColor = b.color?.toLowerCase();

      const aPreset = aColor ? colorToPreset.get(aColor) : null;
      const bPreset = bColor ? colorToPreset.get(bColor) : null;

      // Tags without color go to the end
      if (!aPreset && !bPreset) {
        return a.name.localeCompare(b.name);
      }
      if (!aPreset) return 1;
      if (!bPreset) return -1;

      // Compare by sort order
      if (aPreset.sort !== bPreset.sort) {
        return aPreset.sort - bPreset.sort;
      }

      // If sort is same, compare by color
      return aPreset.color.localeCompare(bPreset.color);
    });

    const tags = sortedTags.map((tag) => (
      <TagLink key={tag.id} tag={tag} linkType="details" />
    ));

    return (
      <>
        <div className="mt-3 mb-3">
          <h4>
            <FormattedMessage
              id="countables.tags"
              values={{ count: generalTags.length }}
            />
            <span className="text-muted"> ({generalTags.length})</span>
          </h4>
        </div>
        {tags}
      </>
    );
  }

  function renderPerformerTags() {
    if (!props.scene.performers || props.scene.performers.length === 0) return;

    // Show tags for each performer in the order of performers list (include all performers)
    const performerTags = props.scene.performers
      .map((performer) => {
        const performerTagData = props.scene.performer_tag_ids?.find(
          (pt: GQL.PerformerTag) => pt.performer_id === performer.id
        );

        if (
          !performerTagData ||
          !performerTagData.tag_ids ||
          performerTagData.tag_ids.length === 0
        ) {
          return null; // No tags for this performer
        }

        // Get tag objects using getTagById function
        const tags = performerTagData.tag_ids
          .map((tagId: string) => getTagById(tagId))
          .filter(Boolean) as GQL.SlimTagDataFragment[];

        if (tags.length === 0) {
          return null; // Skip performers whose tags are not loaded yet
        }

        return { performer, tags };
      })
      .filter(
        (
          item
        ): item is {
          performer: GQL.PerformerDataFragment;
          tags: GQL.SlimTagDataFragment[];
        } => item !== null
      );

    if (performerTags.length === 0) return;

    return (
      <>
        {performerTags.map(({ performer, tags }) => (
          <div key={performer.id} className="mt-3 mb-3">
            <h4>
              <FormattedMessage id="countables.tags" values={{ count: 0 }} />
              <PerformerPopover id={performer.id}>
                <span
                  className="badge badge-secondary ml-2"
                  style={{ display: "inline-block" }}
                >
                  {performer.name}
                </span>
              </PerformerPopover>
              <span className="text-muted"> ({tags.length})</span>
            </h4>
            <div>
              {tags.map((tag: GQL.SlimTagDataFragment) => (
                <TagLink
                  key={`${performer.id}-${tag.id}`}
                  // eslint-disable-next-line @typescript-eslint/no-explicit-any
                  tag={tag as any}
                  linkType="details"
                />
              ))}
            </div>
          </div>
        ))}
      </>
    );
  }

  function renderPerformers() {
    // Use scene_performers if available, otherwise fall back to performers
    const performersData = ((props.scene as { scene_performers?: unknown })
      .scene_performers ?? props.scene.performers) as Array<
      | {
          performer?: GQL.PerformerDataFragment;
          small_role?: boolean;
          role_description?: string;
        }
      | GQL.PerformerDataFragment
    >;
    if (performersData.length === 0) return;

    // Separate performers into two groups
    // For scene_performers: check both performer.small_role (from performer card) and scene.small_role (from scene data)
    // For performers: check only performer.small_role
    const mainPerformers = performersData.filter((p) => {
      if ("performer" in p && p.performer) {
        // scene_performers format
        return !(p.small_role || p.performer.small_role);
      } else {
        // performers format
        return !(p as GQL.PerformerDataFragment).small_role;
      }
    });
    const smallRolePerformers = performersData.filter((p) => {
      if ("performer" in p && p.performer) {
        // scene_performers format
        return p.small_role || p.performer.small_role;
      } else {
        // performers format
        return (p as GQL.PerformerDataFragment).small_role;
      }
    });

    // Sort main performers
    const sortedMainPerformers = sortPerformers(
      mainPerformers.map(
        (p) =>
          (p as { performer?: GQL.PerformerDataFragment }).performer ||
          (p as GQL.PerformerDataFragment)
      )
    );

    // Create cards for main performers
    const mainCards = sortedMainPerformers.map((performer) => {
      // Find the corresponding scene_performer data for role_description
      const scenePerformerData = performersData.find(
        (sp) =>
          (
            (sp as { performer?: GQL.PerformerDataFragment }).performer ??
            (sp as GQL.PerformerDataFragment)
          ).id === performer.id
      );
      const roleDescription =
        "role_description" in (scenePerformerData || {})
          ? (scenePerformerData as { role_description?: string })
              .role_description
          : undefined;

      return (
        <PerformerCard
          key={performer.id}
          performer={performer}
          ageFromDate={props.scene.date ?? undefined}
          roleDescription={roleDescription}
        />
      );
    });

    // Create list for performers with small role
    const smallRoleList =
      smallRolePerformers.length > 0 ? (
        <div className="mt-3">
          <h6 className="scene-performers-small-role-header">
            <FormattedMessage
              id="scene_performers.small_role"
              defaultMessage="Also starring:"
            />
          </h6>
          <div className="scene-performers-small-role">
            {smallRolePerformers.map((performer) => {
              const performerData =
                "performer" in performer && performer.performer
                  ? performer.performer
                  : (performer as GQL.PerformerDataFragment);
              const roleDescription =
                "role_description" in performer
                  ? performer.role_description
                  : undefined;
              const currentAge = TextUtils.age(
                performerData.birthdate,
                performerData.death_date
              );
              const productionAge = TextUtils.age(
                performerData.birthdate,
                props.scene.date ?? undefined
              );
              const ageShortString = intl.formatMessage({
                id: "years_old_short",
                defaultMessage: "yo",
              });
              const atProductionString = intl.formatMessage({
                id: "at_production",
                defaultMessage: "at production",
              });

              const currentAgeString =
                currentAge > 0 ? `${currentAge} ${ageShortString}` : "";
              const productionAgeString =
                productionAge > 0
                  ? `${productionAge} ${ageShortString} ${atProductionString}`
                  : "";

              return (
                <Link
                  key={performerData.id}
                  to={`/performers/${performerData.id}`}
                  className="scene-performer-small-role-tag"
                >
                  <div className="performer-info">
                    <div className="performer-header">
                      <GenderIcon
                        gender={performerData.gender}
                        className="gender-icon-small"
                      />
                      <span className="performer-name">
                        {performerData.name}
                      </span>
                      {currentAgeString && (
                        <span className="performer-age-small">
                          ({currentAgeString})
                        </span>
                      )}
                    </div>
                    {productionAgeString && (
                      <div className="performer-age-at-production">
                        {productionAgeString}
                      </div>
                    )}
                    {roleDescription && (
                      <div className="performer-role-description-small">
                        <strong>Role:</strong> {roleDescription}
                      </div>
                    )}
                  </div>
                </Link>
              );
            })}
          </div>
        </div>
      ) : null;

    return (
      <>
        <div className="mt-3 mb-3">
          <h4>
            <FormattedMessage
              id="countables.performers"
              values={{ count: props.scene.performers.length }}
            />
          </h4>
        </div>
        <div className="row justify-content-center scene-performers">
          {mainCards}
        </div>
        {smallRoleList}
      </>
    );
  }

  // filename should use entire row if there is no studio
  const sceneDetailsWidth = props.scene.studio ? "col-9" : "col-12";

  return (
    <>
      <div className="row">
        <div className={`${sceneDetailsWidth} col-12 scene-details`}>
          {props.scene.urls && props.scene.urls.length > 0 && (
            <div className="mb-n3">
              <URLsField id="urls" urls={props.scene.urls} truncate />
            </div>
          )}

          {props.scene.code && (
            <div className="mb-2">
              <h6 className="font-weight-bold d-inline-block mr-1">
                <FormattedMessage id="scene_code" />:
              </h6>
              {props.scene.code}{" "}
            </div>
          )}

          {props.scene.director && (
            <div className="mb-2">
              <h6 className="font-weight-bold d-inline-block mr-1">
                <FormattedMessage id="director" />:{" "}
              </h6>
              <DirectorLink director={props.scene.director} linkType="scene" />
            </div>
          )}
        </div>
      </div>
      <div className="row">
        <div className="col-12">
          {renderDetails()}
          <PoseTagsDisplay scene={props.scene} />
          {renderGeneralTags()}
          {renderPerformerTags()}
          {renderPerformers()}
        </div>
      </div>
      <div className="row">
        <div className="col-12">
          <SimilarScenes scene={props.scene} />
        </div>
      </div>
      <div className="row">
        <div className="col-12">
          <div className="mt-3">
            <div className="mb-2">
              <h6 className="font-weight-bold d-inline-block mr-1">
                <FormattedMessage id="created_at" />:{" "}
              </h6>
              {TextUtils.formatDateTime(intl, props.scene.created_at)}{" "}
            </div>

            <div className="mb-2">
              <h6 className="font-weight-bold d-inline-block mr-1">
                <FormattedMessage id="updated_at" />:{" "}
              </h6>
              {TextUtils.formatDateTime(intl, props.scene.updated_at)}{" "}
            </div>
          </div>
        </div>
      </div>
    </>
  );
};

export default SceneDetailPanel;
