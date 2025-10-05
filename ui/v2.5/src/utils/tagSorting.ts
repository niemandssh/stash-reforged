import * as GQL from "src/core/generated-graphql";

/**
 * Sorts tags by color preset order
 * 1. By preset sort order (ascending)
 * 2. If same sort, by preset color (ascending)
 * 3. Tags without color go to the end, sorted alphabetically
 */
export function sortTagsByColorPreset(
  tags: GQL.TagDataFragment[],
  colorPresets: GQL.ColorPreset[],
  direction: GQL.SortDirectionEnum = GQL.SortDirectionEnum.Asc
): GQL.TagDataFragment[] {
  const colorToPreset = new Map<string, GQL.ColorPreset>();
  colorPresets.forEach(preset => {
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

    return direction === GQL.SortDirectionEnum.Desc ? -colorComparison : colorComparison;
  });

  return sortedTags;
}

/**
 * Gets the color preset for a tag
 */
export function getTagColorPreset(
  tag: GQL.TagDataFragment,
  colorPresets: GQL.ColorPreset[]
): GQL.ColorPreset | null {
  if (!tag.color) return null;

  return colorPresets.find(preset =>
    preset.color.toLowerCase() === tag.color?.toLowerCase()
  ) || null;
}
