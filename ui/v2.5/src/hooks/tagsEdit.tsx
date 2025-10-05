import * as GQL from "src/core/generated-graphql";
import { useTagCreate } from "src/core/StashService";
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

  if (typeof setFieldValueOrSceneId === 'function') {
    setFieldValue = setFieldValueOrSceneId;
    if (typeof sceneIdOrShouldUpdate === 'string') {
      sceneId = sceneIdOrShouldUpdate;
    } else if (typeof sceneIdOrShouldUpdate === 'boolean') {
      finalShouldUpdateFromSrc = sceneIdOrShouldUpdate;
    }
  } else if (typeof setFieldValueOrSceneId === 'string') {
    sceneId = setFieldValueOrSceneId;
    if (typeof sceneIdOrShouldUpdate === 'boolean') {
      finalShouldUpdateFromSrc = sceneIdOrShouldUpdate;
    }
  }

  const [tags, setTags] = useState<Tag[]>([]);
  const [newTags, setNewTags] = useState<GQL.ScrapedTag[]>();

  const {
    addToHistory,
    undo,
    redo,
    clearHistory,
    canUndo,
    canRedo
  } = useTagsHistory(sceneId || undefined);

  function onSetTags(items: Tag[]) {
    setTags(items);
    if (setFieldValue) {
      setFieldValue(items.map((item) => item.id));
    }
    addToHistory(items);
  }

  function undoTags() {
    const previousTags = undo();
    if (previousTags) {
      setTags(previousTags);
      if (setFieldValue) {
        setFieldValue(previousTags.map((item) => item.id));
      }
    }
    return previousTags;
  }

  function redoTags() {
    const nextTags = redo();
    if (nextTags) {
      setTags(nextTags);
      if (setFieldValue) {
        setFieldValue(nextTags.map((item) => item.id));
      }
    }
    return nextTags;
  }

  useEffect(() => {
    if (finalShouldUpdateFromSrc !== false && srcTags) {
      console.log('🏷️ useTagsEdit: updating tags from srcTags, count:', srcTags?.length);

      setTags(currentTags => {
        const currentTagIds = new Set(currentTags.map(t => t.id));
        const srcTagIds = new Set(srcTags.map(t => t.id));

        const updatedTags = currentTags.filter(tag => srcTagIds.has(tag.id));

        const newTags = srcTags.filter(tag => !currentTagIds.has(tag.id));
        // Convert GQL.Tag to Tag type for TagSelect
        const convertedNewTags = newTags.map(tag => ({
          id: tag.id,
          name: tag.name,
          sort_name: tag.sort_name,
          aliases: tag.aliases,
          image_path: tag.image_path,
          is_pose_tag: tag.is_pose_tag,
          color: tag.color
        }));
        updatedTags.push(...convertedNewTags);

        return updatedTags;
      });
    } else {
      console.log('🚫 useTagsEdit: NOT updating tags from srcTags (user has modified)');
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
            name: toCreate.name ?? "",
            aliases: [],
            is_pose_tag: false,
            color: undefined, // New tags don't have color initially
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

  function updateTagsStateFromScraper(
    scrapedTags?: Pick<GQL.ScrapedTag, "name" | "stored_id">[]
  ) {
    if (!scrapedTags) {
      return;
    }

    // map tags to their ids and filter out those not found
    const idTags = scrapedTags.filter(
      (t) => t.stored_id !== undefined && t.stored_id !== null
    );
    const newNewTags = scrapedTags.filter((t) => !t.stored_id);
    onSetTags(
      idTags.map((p) => {
        return {
          id: p.stored_id!,
          name: p.name ?? "",
          aliases: [],
          is_pose_tag: false,
          color: undefined, // Scraped tags don't have color info
        };
      })
    );

    setNewTags(newNewTags);
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
        <TagSelect isMulti onSelect={onSetTags} values={tags} {...props} />
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
