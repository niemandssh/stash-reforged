import { useState } from "react";
import { useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { ObjectListScrapeResult } from "./scrapeResult";
import { sortStoredIdObjects } from "src/utils/data";
import { Tag } from "src/components/Tags/TagSelect";
import { useCreateScrapedTag } from "./createObjects";
import { ScrapedTagsRow } from "./ScrapedObjectsRow";

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

export function useScrapedTags(
  existingTags: Tag[],
  scrapedTags?: GQL.Maybe<GQL.ScrapedTag[]>
) {
  const intl = useIntl();
  const [tags, setTags] = useState<ObjectListScrapeResult<GQL.ScrapedTag>>(
    new ObjectListScrapeResult<GQL.ScrapedTag>(
      sortStoredIdObjects(
        existingTags.map((t) => ({
          stored_id: t.id,
          name: t.name,
        }))
      ),
      sortStoredIdObjects(scrapedTags ?? undefined)
    )
  );

  const [newTags, setNewTags] = useState<GQL.ScrapedTag[]>(
    scrapedTags?.filter((t) => !t.stored_id) ?? []
  );

  const [createdTags, setCreatedTags] = useState<Tag[]>([]);

  const createNewTags = useCreateScrapedTag({
    scrapeResult: tags,
    setScrapeResult: setTags,
    newObjects: newTags,
    setNewObjects: setNewTags,
    onTagCreated: (tag) => {
      setCreatedTags((prev) => [
        ...prev.filter((t) => t.id !== tag.id),
        tagDataToTag(tag),
      ]);
    },
  });

  const scrapedTagsRow = (
    <ScrapedTagsRow
      title={intl.formatMessage({ id: "tags" })}
      result={tags}
      onChange={(value) => setTags(value)}
      newObjects={newTags}
      onCreateNew={(tag) => createNewTags([tag])}
      onCreateAll={() => createNewTags(newTags)}
    />
  );

  return {
    tags,
    newTags,
    createdTags,
    scrapedTagsRow,
  };
}
