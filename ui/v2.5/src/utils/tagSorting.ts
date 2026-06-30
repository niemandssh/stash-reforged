import * as GQL from "src/core/generated-graphql";

export type TagWithColor = Pick<
  GQL.SlimTagDataFragment,
  "id" | "name" | "color" | "is_pose_tag"
>;

export function getPerformerSpecificTagIds(
  performerTagIds?: Array<
    Pick<GQL.PerformerTag, "performer_id" | "tag_ids"> | null
  > | null
): Set<string> {
  const ids = new Set<string>();
  performerTagIds?.forEach((pt) => {
    if (pt?.performer_id && pt.tag_ids) {
      pt.tag_ids.forEach((tagId) => ids.add(tagId));
    }
  });
  return ids;
}

export function filterSceneGeneralTags<T extends TagWithColor>(
  tags: T[],
  performerTagIds?: Array<
    Pick<GQL.PerformerTag, "performer_id" | "tag_ids"> | null
  > | null
): T[] {
  const performerSpecificTagIds = getPerformerSpecificTagIds(performerTagIds);
  return tags.filter(
    (tag) => !tag.is_pose_tag && !performerSpecificTagIds.has(tag.id)
  );
}

/**
 * Sorts tags by color preset order
 * 1. By preset sort order (ascending)
 * 2. If same sort, by preset color (ascending)
 * 3. Tags without color go to the end, sorted alphabetically
 */
export function sortTagsByColorPreset<T extends TagWithColor>(
  tags: T[],
  colorPresets: GQL.ColorPreset[],
  direction: GQL.SortDirectionEnum = GQL.SortDirectionEnum.Asc
): T[] {
  const colorToPreset = new Map<string, GQL.ColorPreset>();
  colorPresets.forEach((preset) => {
    colorToPreset.set(preset.color.toLowerCase(), preset);
  });

  const sortedTags = [...tags].sort((a, b) => {
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

    const colorComparison = aPreset.color.localeCompare(bPreset.color);

    return direction === GQL.SortDirectionEnum.Desc
      ? -colorComparison
      : colorComparison;
  });

  return sortedTags;
}

/**
 * Gets the color preset for a tag
 */
export function getTagColorPreset(
  tag: TagWithColor,
  colorPresets: GQL.ColorPreset[]
): GQL.ColorPreset | null {
  if (!tag.color) return null;

  return (
    colorPresets.find(
      (preset) => preset.color.toLowerCase() === tag.color?.toLowerCase()
    ) || null
  );
}
