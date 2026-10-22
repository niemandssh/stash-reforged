import { uniq, uniqBy } from "lodash-es";
import * as GQL from "src/core/generated-graphql";
import {
  booleanMergeConflict,
  hasRealImagePath,
  imageMergeConflict,
  ratingMergeConflict,
  scalarMergeConflict,
} from "src/components/Shared/mergeConflictUtils";
import { genderToString } from "src/utils/gender";
import { circumcisedToString } from "src/utils/circumcised";

function idToStoredID(o: { id: string; name: string }) {
  return { stored_id: o.id, name: o.name };
}

export function aggregatePerformersForMerge(
  sources: GQL.PerformerDataFragment[]
): GQL.ScrapedPerformer {
  return {
    name: sources.find((s) => s.name)?.name ?? "",
    disambiguation: sources.find((s) => s.disambiguation)?.disambiguation,
    aliases: uniq(
      sources.flatMap((s) => [s.name, ...s.alias_list].filter(Boolean))
    ).join(", "),
    urls: uniq(sources.flatMap((s) => s.urls ?? [])),
    gender: sources.find((s) => s.gender)?.gender ?? undefined,
    birthdate: sources.find((s) => s.birthdate)?.birthdate,
    death_date: sources.find((s) => s.death_date)?.death_date,
    ethnicity: sources.find((s) => s.ethnicity)?.ethnicity,
    country: sources.find((s) => s.country)?.country,
    eye_color: sources.find((s) => s.eye_color)?.eye_color,
    hair_color: sources.find((s) => s.hair_color)?.hair_color,
    height: sources.find((s) => s.height_cm)?.height_cm?.toString(),
    weight: sources.find((s) => s.weight)?.weight?.toString(),
    measurements: sources.find((s) => s.measurements)?.measurements,
    fake_tits: sources.find((s) => s.fake_tits)?.fake_tits,
    penis_length: sources.find((s) => s.penis_length)?.penis_length?.toString(),
    circumcised: sources.find((s) => s.circumcised)?.circumcised ?? undefined,
    career_length: sources.find((s) => s.career_length)?.career_length,
    tattoos: sources.find((s) => s.tattoos)?.tattoos,
    piercings: sources.find((s) => s.piercings)?.piercings,
    details: sources.find((s) => s.details)?.details,
    tags: uniqBy(
      sources.flatMap((s) => s.tags.map(idToStoredID)),
      (t) => t.stored_id
    ),
    images: (() => {
      const path = sources.find((s) => hasRealImagePath(s.image_path))
        ?.image_path;
      return path ? [path] : undefined;
    })(),
  };
}

export function performerMergeHasConflicts(
  dest: GQL.PerformerDataFragment,
  sources: GQL.PerformerDataFragment[]
): boolean {
  const scraped = aggregatePerformersForMerge(sources);

  if (scalarMergeConflict(dest.disambiguation, scraped.disambiguation)) return true;
  if (scalarMergeConflict(dest.details, scraped.details)) return true;
  if (scalarMergeConflict(dest.ethnicity, scraped.ethnicity)) return true;
  if (scalarMergeConflict(dest.country, scraped.country)) return true;
  if (scalarMergeConflict(dest.eye_color, scraped.eye_color)) return true;
  if (scalarMergeConflict(dest.hair_color, scraped.hair_color)) return true;
  if (scalarMergeConflict(dest.measurements, scraped.measurements)) return true;
  if (scalarMergeConflict(dest.fake_tits, scraped.fake_tits)) return true;
  if (scalarMergeConflict(dest.career_length, scraped.career_length)) return true;
  if (scalarMergeConflict(dest.tattoos, scraped.tattoos)) return true;
  if (scalarMergeConflict(dest.piercings, scraped.piercings)) return true;
  if (scalarMergeConflict(dest.birthdate, scraped.birthdate)) return true;
  if (scalarMergeConflict(dest.death_date, scraped.death_date)) return true;

  const destGender = genderToString(dest.gender);
  const mergedGender = genderToString(scraped.gender as GQL.GenderEnum | undefined);
  if (scalarMergeConflict(destGender, mergedGender)) return true;

  const destCircumcised = circumcisedToString(dest.circumcised);
  const mergedCircumcised = circumcisedToString(
    scraped.circumcised as GQL.CircumisedEnum | undefined
  );
  if (scalarMergeConflict(destCircumcised, mergedCircumcised)) return true;

  if (
    scalarMergeConflict(
      dest.height_cm?.toString(),
      scraped.height
    )
  ) {
    return true;
  }
  if (scalarMergeConflict(dest.weight?.toString(), scraped.weight)) return true;
  if (
    scalarMergeConflict(dest.penis_length?.toString(), scraped.penis_length)
  ) {
    return true;
  }

  if (ratingMergeConflict(dest.rating100, sources.find((s) => s.rating100)?.rating100)) {
    return true;
  }
  if (booleanMergeConflict(dest.favorite, sources.every((s) => s.favorite))) {
    return true;
  }
  if (
    imageMergeConflict(
      dest.image_path,
      sources.find((s) => hasRealImagePath(s.image_path))?.image_path
    )
  ) {
    return true;
  }

  return false;
}

export function performerToUpdateInput(
  dest: GQL.PerformerDataFragment
): GQL.PerformerUpdateInput {
  return {
    id: dest.id,
    name: dest.name,
    disambiguation: dest.disambiguation,
    alias_list: dest.alias_list,
    urls: dest.urls,
    gender: dest.gender,
    birthdate: dest.birthdate,
    death_date: dest.death_date,
    ethnicity: dest.ethnicity,
    country: dest.country,
    eye_color: dest.eye_color,
    hair_color: dest.hair_color,
    height_cm: dest.height_cm,
    weight: dest.weight,
    measurements: dest.measurements,
    fake_tits: dest.fake_tits,
    penis_length: dest.penis_length,
    circumcised: dest.circumcised,
    career_length: dest.career_length,
    tattoos: dest.tattoos,
    piercings: dest.piercings,
    details: dest.details,
    rating100: dest.rating100,
    favorite: dest.favorite,
    tag_ids: dest.tags.map((t) => t.id),
    primary_tag_id: dest.primary_tag?.id,
    stash_ids: dest.stash_ids.map((s) => ({
      stash_id: s.stash_id,
      endpoint: s.endpoint,
    })),
    ignore_auto_tag: dest.ignore_auto_tag,
    small_role: dest.small_role,
  };
}
