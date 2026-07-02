import * as GQL from "src/core/generated-graphql";
import { getClient, useTagCreate } from "src/core/StashService";
import { useEffect, useState } from "react";
import { Tag, TagSelect, TagSelectProps } from "src/components/Tags/TagSelect";
import { useToast } from "src/hooks/Toast";
import { useIntl } from "react-intl";
import { Badge, Button } from "react-bootstrap";
import { Icon } from "src/components/Shared/Icon";
import { faPlus } from "@fortawesome/free-solid-svg-icons";
import { CollapseButton } from "src/components/Shared/CollapseButton";
import { useTagsHistory } from "./tagsHistory";

export function useTagsEdit(
  srcTags: Tag[] | undefined,
  setFieldValueOrSceneId?: ((ids: string[]) => void) | string,
  sceneIdOrShouldUpdate?: string | boolean,
  shouldUpdateFromSrc = true
) {
  const intl = useIntl();
  const Toast = useToast();
  const [createTag] = useTagCreate();

  let setFieldValue: ((ids: string[]) => void) | undefined;
  let sceneId: string | undefined;
  let finalShouldUpdateFromSrc = shouldUpdateFromSrc;

  if (typeof setFieldValueOrSceneId === "function") {
    setFieldValue = setFieldValueOrSceneId;
    if (typeof sceneIdOrShouldUpdate === "string") {
      sceneId = sceneIdOrShouldUpdate;
    } else if (typeof sceneIdOrShouldUpdate === "boolean") {
      finalShouldUpdateFromSrc = sceneIdOrShouldUpdate;
    }
  } else if (typeof setFieldValueOrSceneId === "string") {
    sceneId = setFieldValueOrSceneId;
    if (typeof sceneIdOrShouldUpdate === "boolean") {
      finalShouldUpdateFromSrc = sceneIdOrShouldUpdate;
    }
  }

  const [tags, setTags] = useState<Tag[]>([]);
  const [newTags, setNewTags] = useState<GQL.ScrapedTag[]>();

  const { addToHistory, undo, redo, clearHistory, canUndo, canRedo } =
    useTagsHistory(sceneId || undefined);

  function onSetTags(items: Tag[]) {
    // Remove duplicates by id
    const uniqueItems = items.filter(
      (item, index, arr) => arr.findIndex((t) => t.id === item.id) === index
    );
    setTags(uniqueItems);
    if (setFieldValue) {
      const tagIds = uniqueItems.map((item) => item.id);
      setFieldValue(tagIds);
    }
    addToHistory(uniqueItems);
  }

  function undoTags() {
    const previousTags = undo();
    if (previousTags) {
      // Remove duplicates by id
      const uniquePreviousTags = previousTags.filter(
        (item, index, arr) => arr.findIndex((t) => t.id === item.id) === index
      );
      setTags(uniquePreviousTags);
      if (setFieldValue) {
        const tagIds = uniquePreviousTags.map((item) => item.id);
        setFieldValue(tagIds);
      }
    }
    return previousTags;
  }

  function redoTags() {
    const nextTags = redo();
    if (nextTags) {
      // Remove duplicates by id
      const uniqueNextTags = nextTags.filter(
        (item, index, arr) => arr.findIndex((t) => t.id === item.id) === index
      );
      setTags(uniqueNextTags);
      if (setFieldValue) {
        const tagIds = uniqueNextTags.map((item) => item.id);
        setFieldValue(tagIds);
      }
    }
    return nextTags;
  }

  useEffect(() => {
    // Update from srcTags if provided, regardless of user interaction
    // But only merge if user hasn't interacted, to avoid conflicts
    if (srcTags) {
      if (finalShouldUpdateFromSrc) {
        // User hasn't interacted - full sync
        setTags((currentTags) => {
          const currentTagIds = new Set(currentTags.map((t) => t.id));
          const srcTagIds = new Set(srcTags.map((t) => t.id));

          // Check if we actually need to update
          const hasChanges =
            srcTags.some((tag) => !currentTagIds.has(tag.id)) ||
            currentTags.some((tag) => !srcTagIds.has(tag.id));

          if (!hasChanges) {
            return currentTags; // No changes, return same reference to prevent re-renders
          }

          const updatedTags = currentTags.filter((tag) =>
            srcTagIds.has(tag.id)
          );

          const tagsToAdd = srcTags.filter((tag) => !currentTagIds.has(tag.id));
          // Convert GQL.Tag to Tag type for TagSelect
          const convertedNewTags = tagsToAdd.map((tag) => ({
            id: tag.id,
            name: tag.name,
            sort_name: tag.sort_name,
            aliases: tag.aliases,
            image_path: tag.image_path,
            is_pose_tag: tag.is_pose_tag,
            color: tag.color,
          }));
          updatedTags.push(...convertedNewTags);

          // Remove duplicates
          const uniqueUpdatedTags = updatedTags.filter(
            (tag, index, arr) => arr.findIndex((t) => t.id === tag.id) === index
          );

          return uniqueUpdatedTags;
        });
      } else {
        // User has interacted - only add missing tags from srcTags, don't remove existing
        setTags((currentTags) => {
          const currentTagIds = new Set(currentTags.map((t) => t.id));

          const tagsToAdd = srcTags.filter((tag) => !currentTagIds.has(tag.id));
          if (tagsToAdd.length > 0) {
            // Convert GQL.Tag to Tag type for TagSelect
            const convertedNewTags = tagsToAdd.map((tag) => ({
              id: tag.id,
              name: tag.name,
              sort_name: tag.sort_name,
              aliases: tag.aliases,
              image_path: tag.image_path,
              is_pose_tag: tag.is_pose_tag,
              color: tag.color,
            }));

            const combinedTags = [...currentTags, ...convertedNewTags];
            // Remove duplicates
            const uniqueCombinedTags = combinedTags.filter(
              (tag, index, arr) =>
                arr.findIndex((t) => t.id === tag.id) === index
            );
            return uniqueCombinedTags;
          }

          return currentTags;
        });
      }
    }
  }, [srcTags, finalShouldUpdateFromSrc]);

  async function createNewTag(toCreate: GQL.ScrapedTag) {
    const tagInput: GQL.TagCreateInput = { name: toCreate.name ?? "" };
    try {
      const result = await createTag({
        variables: {
          input: tagInput,
        },
      });

      if (!result.data?.tagCreate) {
        Toast.error(new Error("Failed to create tag"));
        return;
      }

      // add the new tag to the new tags value
      onSetTags(
        tags.concat([
          {
            id: result.data.tagCreate.id,
            name: result.data.tagCreate.name,
            sort_name: result.data.tagCreate.sort_name,
            aliases: result.data.tagCreate.aliases,
            image_path: result.data.tagCreate.image_path,
            is_pose_tag: result.data.tagCreate.is_pose_tag,
            color: result.data.tagCreate.color,
          },
        ])
      );

      // remove the tag from the list
      const newTagsClone = newTags!.concat();
      const pIndex = newTagsClone.indexOf(toCreate);
      newTagsClone.splice(pIndex, 1);

      setNewTags(newTagsClone);

      Toast.success(
        intl.formatMessage(
          { id: "toast.created_entity" },
          {
            entity: intl.formatMessage({ id: "tag" }).toLocaleLowerCase(),
            entity_name: toCreate.name,
          }
        )
      );
    } catch (e) {
      Toast.error(e);
    }
  }

  function scrapedTagToTag(
    scraped: Pick<GQL.ScrapedTag, "name" | "stored_id">
  ): Tag {
    return {
      id: scraped.stored_id!,
      name: scraped.name ?? "",
      aliases: [],
      is_pose_tag: false,
      color: undefined,
    };
  }

  function tagDataToTag(tag: GQL.TagDataFragment): Tag {
    return {
      id: tag.id,
      name: tag.name,
      sort_name: tag.sort_name,
      aliases: tag.aliases,
      image_path: tag.image_path,
      is_pose_tag: tag.is_pose_tag,
      color: tag.color,
    };
  }

  async function resolveTagsByIDs(
    ids: string[],
    existing: Map<string, Tag>
  ): Promise<Map<string, Tag>> {
    const resolved = new Map(existing);
    const uniqueMissing = [...new Set(ids.filter((id) => !resolved.has(id)))];

    if (uniqueMissing.length === 0) {
      return resolved;
    }

    const client = getClient();
    await Promise.all(
      uniqueMissing.map(async (id) => {
        try {
          let result = await client.query<GQL.FindTagQuery>({
            query: GQL.FindTagDocument,
            variables: { id },
            fetchPolicy: "cache-first",
          });
          if (!result.data?.findTag) {
            result = await client.query<GQL.FindTagQuery>({
              query: GQL.FindTagDocument,
              variables: { id },
              fetchPolicy: "network-only",
            });
          }
          if (result.data?.findTag) {
            resolved.set(id, tagDataToTag(result.data.findTag));
          }
        } catch {
          // Tag may not exist (e.g. stale stored_id from scraper)
        }
      })
    );

    return resolved;
  }

  async function updateTagsStateFromScraper(
    scrapedTags?: Pick<GQL.ScrapedTag, "name" | "stored_id">[],
    supplementalTags?: Tag[]
  ) {
    if (!scrapedTags) {
      return;
    }

    const idTags = scrapedTags.filter(
      (t) => t.stored_id !== undefined && t.stored_id !== null
    );
    const newNewTags = scrapedTags.filter((t) => !t.stored_id);
    setNewTags(newNewTags);

    if (idTags.length === 0) {
      onSetTags([]);
      return;
    }

    const tagMap = new Map<string, Tag>();
    for (const tag of tags) {
      tagMap.set(tag.id, tag);
    }
    for (const tag of supplementalTags ?? []) {
      tagMap.set(tag.id, tag);
    }

    const ids = idTags.map((t) => t.stored_id!);
    const resolvedMap = await resolveTagsByIDs(ids, tagMap);

    const seen = new Set<string>();
    const resolvedTags = idTags
      .map((scrapedTag) => {
        const id = scrapedTag.stored_id!;
        return resolvedMap.get(id) ?? scrapedTagToTag(scrapedTag);
      })
      .filter((tag) => {
        if (seen.has(tag.id)) {
          return false;
        }
        seen.add(tag.id);
        return true;
      });

    onSetTags(resolvedTags);
  }

  function renderNewTags() {
    if (!newTags || newTags.length === 0) {
      return;
    }

    const ret = (
      <>
        {newTags.map((t) => (
          <Badge
            className="tag-item"
            variant="secondary"
            key={t.name}
            onClick={() => createNewTag(t)}
          >
            {t.name}
            <Button className="minimal ml-2">
              <Icon className="fa-fw" icon={faPlus} />
            </Button>
          </Badge>
        ))}
      </>
    );

    const minCollapseLength = 10;

    if (newTags.length >= minCollapseLength) {
      return (
        <CollapseButton text={`Missing (${newTags.length})`}>
          {ret}
        </CollapseButton>
      );
    }

    return ret;
  }

  function tagsControl(props?: TagSelectProps) {
    return (
      <>
        <TagSelect
          key={`tag-select-${sceneId || "unknown"}`}
          isMulti
          onSelect={onSetTags}
          values={tags}
          instanceId={sceneId || "new-scene"}
          {...props}
        />
        {renderNewTags()}
      </>
    );
  }

  return {
    tags,
    onSetTags,
    tagsControl,
    updateTagsStateFromScraper,
    undoTags,
    redoTags,
    clearHistory,
    canUndo,
    canRedo,
  };
}
