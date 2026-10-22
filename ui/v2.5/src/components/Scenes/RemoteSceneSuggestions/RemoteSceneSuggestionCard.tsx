import React, { useMemo } from "react";
import { Badge, Button, ButtonGroup } from "react-bootstrap";
import { faTag, faUser } from "@fortawesome/free-solid-svg-icons";
import { Link } from "react-router-dom";
import cx from "classnames";
import * as GQL from "src/core/generated-graphql";
import { GridCard } from "src/components/Shared/GridCard/GridCard";
import { StudioOverlay } from "src/components/Shared/GridCard/StudioOverlay";
import { HoverPopover } from "src/components/Shared/HoverPopover";
import { Icon } from "src/components/Shared/Icon";
import { PerformerLink, TagLink } from "src/components/Shared/TagLink";
import { DeathRibbon } from "src/components/Shared/DeathRibbon";
import { useFindColorPresets, useFindStudio } from "src/core/StashService";
import {
  filterSceneGeneralTags,
  sortTagsByColorPreset,
} from "src/utils/tagSorting";
import TextUtils from "src/utils/text";
import {
  dedupeScrapedSceneTags,
  OwnedTagRef,
} from "./tagMatching";
import RemoteSceneSuggestionContextMenu from "./RemoteSceneSuggestionContextMenu";

export type RemoteSceneSuggestionCardProps = {
  scene: GQL.ScrapedSceneDataFragment;
  providerLabel: string;
  priority?: boolean;
  priorityColor?: string | null;
};

/** Fallback highlight for owned tags that have no custom color in the DB. */
const OWNED_TAG_FALLBACK_COLOR = "#137cbd";

type DisplayTag = {
  id: string;
  name: string;
  sort_name?: string | null;
  color?: string | null;
  is_pose_tag: boolean;
  owned: boolean;
};

function getSourceUrl(scene: GQL.ScrapedSceneDataFragment): string {
  return scene.urls?.[0] ?? scene.url ?? "";
}

const RemoteSceneSuggestionCard: React.FC<
  RemoteSceneSuggestionCardProps
> = ({ scene, providerLabel, priority = false, priorityColor }) => {
  const sourceUrl = useMemo(() => getSourceUrl(scene), [scene]);
  const studioId = scene.studio?.stored_id ?? "";
  const { data: studioData } = useFindStudio(studioId);
  const { data: presetsData } = useFindColorPresets();
  const colorPresets = presetsData?.findColorPresets?.color_presets || [];

  const durationSeconds = scene.duration ?? scene.file?.duration ?? undefined;
  const file = scene.file;

  const performers = scene.performers ?? [];
  const tags = scene.tags ?? [];

  const ownedTagIds = useMemo(
    () =>
      Array.from(
        new Set(
          tags
            .map((t) => t.stored_id)
            .filter((id): id is string => !!id)
        )
      ),
    [tags]
  );

  const ownedPerformerIds = useMemo(
    () =>
      Array.from(
        new Set(
          performers
            .map((p) => p.stored_id)
            .filter((id): id is string => !!id)
        )
      ),
    [performers]
  );

  const { data: ownedTagsData } = GQL.useFindTagsForSelectQuery({
    variables: {
      ids: ownedTagIds,
      filter: { per_page: ownedTagIds.length || 1 },
    },
    skip: ownedTagIds.length === 0,
  });

  const { data: ownedPerformersData } = GQL.useFindPerformersForSelectQuery({
    variables: {
      ids: ownedPerformerIds,
      filter: { per_page: ownedPerformerIds.length || 1 },
    },
    skip: ownedPerformerIds.length === 0,
  });

  const ownedTagsById = useMemo(() => {
    const map = new Map<
      string,
      OwnedTagRef & {
        id: string;
        sort_name?: string | null;
        color?: string | null;
        is_pose_tag: boolean;
      }
    >();
    for (const tag of ownedTagsData?.findTags?.tags ?? []) {
      map.set(tag.id, tag);
    }
    return map;
  }, [ownedTagsData]);

  const dedupedTags = useMemo(
    () => dedupeScrapedSceneTags(tags, ownedTagsById),
    [tags, ownedTagsById]
  );

  const ownedPerformersById = useMemo(() => {
    const map = new Map<
      string,
      GQL.SelectPerformerDataFragment
    >();
    for (const performer of ownedPerformersData?.findPerformers?.performers ??
      []) {
      map.set(performer.id, performer);
    }
    return map;
  }, [ownedPerformersData]);

  const sortedTags = useMemo(() => {
    const displayTags: DisplayTag[] = dedupedTags.map((tag, idx) => {
      if (tag.stored_id) {
        const owned = ownedTagsById.get(tag.stored_id);
        return {
          id: tag.stored_id,
          name: owned?.name ?? tag.name ?? "",
          sort_name: owned?.sort_name,
          color: owned?.color || OWNED_TAG_FALLBACK_COLOR,
          is_pose_tag: owned?.is_pose_tag ?? false,
          owned: true,
        };
      }

      return {
        id: `unmatched-${tag.name ?? idx}`,
        name: tag.name ?? "",
        color: null,
        is_pose_tag: false,
        owned: false,
      };
    });

    return sortTagsByColorPreset(
      filterSceneGeneralTags(displayTags),
      colorPresets
    );
  }, [dedupedTags, ownedTagsById, colorPresets]);

  const matchedStudio = studioData?.findStudio
    ? {
        id: studioData.findStudio.id,
        name: studioData.findStudio.name,
        image_path: studioData.findStudio.image_path,
      }
    : undefined;

  const details = (
    <div className="scene-card__details remote-scene-suggestion-details">
      <div>
        <span className="scene-card__date">
          {TextUtils.getDateDisplayString(scene.date ?? undefined, undefined)}
        </span>
      </div>
    </div>
  );

  const image = (
    <>
      <div className="scene-card-preview">
        <img
          className="scene-card-preview-image"
          loading="lazy"
          src={scene.image ?? ""}
          alt=""
        />
      </div>
      {(durationSeconds ?? 0) >= 1 || file?.width && file?.height ? (
        <div className="scene-specs-overlay">
          {file?.width && file?.height ? (
            <span className="overlay-resolution">
              {TextUtils.resolution(file.width, file.height)}
            </span>
          ) : (
            ""
          )}
          {(durationSeconds ?? 0) >= 1 ? (
            <span className="overlay-duration">
              {TextUtils.secondsToTimestamp(durationSeconds ?? 0)}
            </span>
          ) : (
            ""
          )}
        </div>
      ) : null}
    </>
  );

  const overlays = matchedStudio ? (
    <StudioOverlay studio={matchedStudio} />
  ) : (
    <div className="studio-overlay">
      <a
        href={sourceUrl}
        target="_blank"
        rel="noopener noreferrer"
        onClick={(e) => e.stopPropagation()}
      >
        {providerLabel}
      </a>
    </div>
  );

  const tagPopover =
    sortedTags.length > 0 ? (
      <HoverPopover
        className="tag-count"
        placement="bottom"
        content={sortedTags.map((tag) =>
          tag.owned ? (
            <TagLink
              key={tag.id}
              className="tag-owned"
              tag={{
                id: tag.id,
                name: tag.name,
                sort_name: tag.sort_name,
                color: tag.color,
              }}
              linkType="details"
            />
          ) : (
            <Badge key={tag.id} className="tag-item" variant="secondary">
              {tag.name}
            </Badge>
          )
        )}
      >
        <Button className="minimal">
          <Icon icon={faTag} />
          <span>{sortedTags.length}</span>
        </Button>
      </HoverPopover>
    ) : null;

  const performerPopover =
    performers.length > 0 ? (
      <HoverPopover
        className="performer-count"
        placement="bottom"
        content={performers.map((p, idx) => {
          if (p.stored_id) {
            const owned = ownedPerformersById.get(p.stored_id);
            if (owned) {
              return (
                <div className="performer-tag-container row" key={owned.id}>
                  <Link
                    to={`/performers/${owned.id}`}
                    className="performer-tag col m-auto zoom-2 position-relative"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <img
                      className={`image-thumbnail ${
                        owned.death_date ? "deceased" : ""
                      }`}
                      alt={owned.name ?? ""}
                      src={
                        owned.primary_image_path ?? owned.image_path ?? ""
                      }
                    />
                    {owned.death_date && <DeathRibbon />}
                  </Link>
                  <PerformerLink
                    performer={owned}
                    className="d-block"
                    linkType="scene"
                  />
                </div>
              );
            }

            return (
              <Badge
                key={p.stored_id}
                className="tag-item"
                variant="secondary"
              >
                <Link
                  to={`/performers/${p.stored_id}`}
                  onClick={(e) => e.stopPropagation()}
                >
                  {p.name}
                </Link>
              </Badge>
            );
          }

          return (
            <Badge
              key={p.name ?? idx}
              className="tag-item"
              variant="secondary"
            >
              {p.name}
            </Badge>
          );
        })}
      >
        <Button className="minimal">
          <Icon icon={faUser} />
          <span>{performers.length}</span>
        </Button>
      </HoverPopover>
    ) : null;

  const popovers =
    tagPopover || performerPopover ? (
      <>
        <hr />
        <ButtonGroup className="card-popovers">
          {tagPopover}
          {performerPopover}
        </ButtonGroup>
      </>
    ) : undefined;

  return (
    <RemoteSceneSuggestionContextMenu
      sourceUrl={sourceUrl}
      sourceTitle={scene.title ?? undefined}
    >
      <GridCard
        className={cx("scene-card remote-scene-suggestion-card", {
          "remote-scene-suggestion-card--priority": priority,
        })}
        style={
          priority
            ? ({
                ["--preferred-tag-color" as string]:
                  priorityColor || "#ff69b4",
              } as React.CSSProperties)
            : undefined
        }
        url={sourceUrl}
        title={scene.title ?? ""}
        image={image}
        details={details}
        overlays={overlays}
        popovers={popovers}
        thumbnailSectionClassName="video-section"
        linkClassName="scene-card-link"
      />
    </RemoteSceneSuggestionContextMenu>
  );
};

export default RemoteSceneSuggestionCard;
