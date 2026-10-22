import { uniqBy } from "lodash-es";
import * as GQL from "src/core/generated-graphql";
import {
  booleanMergeConflict,
  hasRealImagePath,
  imageMergeConflict,
  ratingMergeConflict,
  scalarMergeConflict,
} from "src/components/Shared/mergeConflictUtils";

function idToStoredID(o: { id: string; name: string }) {
  return { stored_id: o.id, name: o.name };
}

export function aggregateStudiosForMerge(
  sources: GQL.StudioDataFragment[]
) {
  const parentStudio = sources.find((s) => s.parent_studio)?.parent_studio;
  return {
    name: sources.find((s) => s.name)?.name ?? "",
    url: sources.find((s) => s.url)?.url,
    details: sources.find((s) => s.details)?.details,
    tags: uniqBy(
      sources.flatMap((s) => s.tags.map(idToStoredID)),
      (t) => t.stored_id
    ),
    image:
      sources.find((s) => hasRealImagePath(s.image_path))?.image_path ??
      undefined,
    parent: parentStudio
      ? { stored_id: parentStudio.id, name: parentStudio.name }
      : undefined,
  };
}

export function studioMergeHasConflicts(
  dest: GQL.StudioDataFragment,
  sources: GQL.StudioDataFragment[]
): boolean {
  const scraped = aggregateStudiosForMerge(sources);

  if (scalarMergeConflict(dest.url, scraped.url)) return true;
  if (scalarMergeConflict(dest.details, scraped.details)) return true;
  if (ratingMergeConflict(dest.rating100, sources.find((s) => s.rating100)?.rating100)) {
    return true;
  }
  if (booleanMergeConflict(dest.favorite, sources.every((s) => s.favorite))) {
    return true;
  }
  if (imageMergeConflict(dest.image_path, scraped.image)) return true;

  if (dest.parent_studio && scraped.parent) {
    if (dest.parent_studio.id !== scraped.parent.stored_id) {
      return true;
    }
  }

  return false;
}

export function studioToUpdateInput(
  dest: GQL.StudioDataFragment
): GQL.StudioUpdateInput {
  return {
    id: dest.id,
    name: dest.name,
    url: dest.url,
    details: dest.details,
    aliases: dest.aliases,
    rating100: dest.rating100,
    favorite: dest.favorite,
    tag_ids: dest.tags.map((t) => t.id),
    parent_id: dest.parent_studio?.id,
    stash_ids: dest.stash_ids.map((s) => ({
      stash_id: s.stash_id,
      endpoint: s.endpoint,
    })),
    ignore_auto_tag: dest.ignore_auto_tag,
  };
}
