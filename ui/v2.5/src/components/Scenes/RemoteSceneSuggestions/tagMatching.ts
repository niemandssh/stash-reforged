import * as GQL from "src/core/generated-graphql";

export type ConfiguredTagRef = Pick<
  GQL.SelectTagDataFragment,
  "id" | "name" | "aliases" | "color"
>;

function normalizeTagName(name: string): string {
  return name.trim().toLowerCase();
}

function buildTagNameSet(tagRefs: ConfiguredTagRef[]): Set<string> {
  const names = new Set<string>();
  for (const tag of tagRefs) {
    if (tag.name) {
      names.add(normalizeTagName(tag.name));
    }
    for (const alias of tag.aliases ?? []) {
      if (alias) {
        names.add(normalizeTagName(alias));
      }
    }
  }
  return names;
}

export type ScrapedSceneTag = NonNullable<
  GQL.ScrapedSceneDataFragment["tags"]
>[number];

export type OwnedTagRef = {
  name: string;
  aliases?: string[] | null;
};

/** Merges scraped tags that refer to the same library tag or alias name. */
export function dedupeScrapedSceneTags(
  tags: GQL.ScrapedSceneDataFragment["tags"] | undefined,
  ownedTagsById?: Map<string, OwnedTagRef>
): ScrapedSceneTag[] {
  const result: ScrapedSceneTag[] = [];
  const coveredNames = new Set<string>();
  const seenStoredIds = new Set<string>();

  function markCovered(name: string, aliases?: string[] | null) {
    coveredNames.add(normalizeTagName(name));
    for (const alias of aliases ?? []) {
      if (alias) {
        coveredNames.add(normalizeTagName(alias));
      }
    }
  }

  for (const tag of tags ?? []) {
    if (!tag.stored_id) {
      continue;
    }

    if (seenStoredIds.has(tag.stored_id)) {
      continue;
    }

    seenStoredIds.add(tag.stored_id);
    result.push(tag);

    const owned = ownedTagsById?.get(tag.stored_id);
    if (owned?.name) {
      markCovered(owned.name, owned.aliases);
    } else if (tag.name) {
      markCovered(tag.name);
    }
  }

  for (const tag of tags ?? []) {
    if (tag.stored_id) {
      continue;
    }

    const scrapedName = tag.name ? normalizeTagName(tag.name) : "";
    if (!scrapedName || coveredNames.has(scrapedName)) {
      continue;
    }

    coveredNames.add(scrapedName);
    result.push(tag);
  }

  return result;
}

export function sceneMatchesConfiguredTags(
  scene: GQL.ScrapedSceneDataFragment,
  tagRefs: ConfiguredTagRef[]
): boolean {
  if (tagRefs.length === 0) {
    return false;
  }

  const idSet = new Set(tagRefs.map((tag) => tag.id));
  const nameSet = buildTagNameSet(tagRefs);

  for (const tag of scene.tags ?? []) {
    if (tag.stored_id && idSet.has(tag.stored_id)) {
      return true;
    }

    const scrapedName = tag.name ? normalizeTagName(tag.name) : "";
    if (scrapedName && nameSet.has(scrapedName)) {
      return true;
    }
  }

  const title = scene.title ? normalizeTagName(scene.title) : "";
  if (title) {
    for (const name of nameSet) {
      if (name && title.includes(name)) {
        return true;
      }
    }
  }

  return false;
}

export type ConfiguredPerformerRef = Pick<
  GQL.PerformerDataFragment,
  "id" | "name" | "disambiguation"
>;

export function sceneMatchesConfiguredPerformer(
  scene: GQL.ScrapedSceneDataFragment,
  performer: ConfiguredPerformerRef
): boolean {
  const performerNames = new Set<string>();
  if (performer.name?.trim()) {
    performerNames.add(normalizeTagName(performer.name));
  }

  for (const scenePerformer of scene.performers ?? []) {
    if (scenePerformer.stored_id && scenePerformer.stored_id === performer.id) {
      return true;
    }

    const scrapedName = scenePerformer.name
      ? normalizeTagName(scenePerformer.name)
      : "";
    if (scrapedName && performerNames.has(scrapedName)) {
      return true;
    }
  }

  const title = scene.title ? normalizeTagName(scene.title) : "";
  if (title) {
    for (const name of performerNames) {
      if (name && title.includes(name)) {
        return true;
      }
    }
  }

  return false;
}

export type PreferredTagColorSource = {
  id: string;
  name?: string | null;
  aliases?: string[] | null;
  color?: string | null;
};

const DEFAULT_PREFERRED_TAG_COLOR = "#ff69b4";

/**
 * Returns the highlight color for the first preferred tag (by settings order)
 * that matches the scene. Later preferred tags are ignored when an earlier one
 * already matches.
 */
export function getPreferredTagHighlightColor(
  preferredTagIds: string[],
  tagsById: Map<string, PreferredTagColorSource>,
  sceneTags: Array<{
    id?: string | null;
    stored_id?: string | null;
    name?: string | null;
  }>,
  options?: {
    sceneTitle?: string | null;
    matchTitle?: boolean;
    fallbackColor?: string;
  }
): string | null {
  if (preferredTagIds.length === 0) {
    return null;
  }

  const sceneTagIds = new Set<string>();
  const sceneTagNames = new Set<string>();
  for (const tag of sceneTags) {
    if (tag.id) {
      sceneTagIds.add(tag.id);
    }
    if (tag.stored_id) {
      sceneTagIds.add(tag.stored_id);
    }
    if (tag.name) {
      sceneTagNames.add(normalizeTagName(tag.name));
    }
  }

  const title = options?.matchTitle
    ? normalizeTagName(options.sceneTitle ?? "")
    : "";
  const fallback = options?.fallbackColor ?? DEFAULT_PREFERRED_TAG_COLOR;

  for (const preferredId of preferredTagIds) {
    const preferred = tagsById.get(preferredId);
    const names: string[] = [];
    if (preferred?.name) {
      names.push(normalizeTagName(preferred.name));
    }
    for (const alias of preferred?.aliases ?? []) {
      if (alias) {
        names.push(normalizeTagName(alias));
      }
    }

    const matchesId = sceneTagIds.has(preferredId);
    const matchesName = names.some((name) => sceneTagNames.has(name));
    const matchesTitle =
      !!title && names.some((name) => name.length > 0 && title.includes(name));

    if (matchesId || matchesName || matchesTitle) {
      const color = preferred?.color?.trim();
      return color || fallback;
    }
  }

  return null;
}
