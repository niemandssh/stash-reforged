import * as GQL from "src/core/generated-graphql";
import { getClient } from "src/core/StashService";
import {
  addToRandomSceneBuffer,
  clearRandomSceneBuffer,
  getRandomSceneBuffer,
  randomSceneBufferKeyForRating,
  RANDOM_SCENE_BUFFER_KEY_UNCLASSIFIED,
  RANDOM_SCENE_BUFFER_KEY_UNORGANISED,
  shouldClearRandomSceneBuffer,
} from "src/utils/randomSceneBuffer";

const MAX_RANDOM_SCENE_ATTEMPTS = 25;

/** Archived scenes are excluded from navbar Review / Random / Random Best. */
const nonArchivedSceneFilter: GQL.SceneFilterType = {
  is_archived: false,
};

const withNonArchivedFilter = (
  filter: GQL.SceneFilterType
): GQL.SceneFilterType => ({
  ...filter,
  ...nonArchivedSceneFilter,
});

const fetchRandomSceneId = async (
  baseFilter: GQL.SceneFilterType,
  poolCount: number,
  excludeIds: Set<string>
): Promise<string | null> => {
  const client = getClient();

  for (let attempt = 0; attempt < MAX_RANDOM_SCENE_ATTEMPTS; attempt++) {
    const randomPage = Math.floor(Math.random() * poolCount) + 1;

    const result = await client.query<GQL.FindScenesQuery>({
      query: GQL.FindScenesDocument,
      variables: {
        filter: {
          per_page: 1,
          page: randomPage,
          sort: "random",
        },
        scene_filter: baseFilter,
      },
    });

    const scenes = result.data?.findScenes?.scenes || [];
    if (scenes.length === 0) {
      return null;
    }

    const sceneId = scenes[0]?.id || null;
    if (sceneId && !excludeIds.has(sceneId)) {
      return sceneId;
    }
  }

  return null;
};

const getRandomSceneWithFilter = async (
  bufferKey: string,
  baseFilter: GQL.SceneFilterType
): Promise<string | null> => {
  const sceneFilter = withNonArchivedFilter(baseFilter);
  try {
    const countResult = await getClient().query<GQL.FindScenesQuery>({
      query: GQL.FindScenesDocument,
      variables: {
        filter: {
          per_page: 0,
        },
        scene_filter: sceneFilter,
      },
    });

    const totalCount = countResult.data?.findScenes?.count || 0;
    if (totalCount === 0) {
      return null;
    }

    let buffer = getRandomSceneBuffer(bufferKey);
    if (shouldClearRandomSceneBuffer(buffer, totalCount)) {
      clearRandomSceneBuffer(bufferKey);
      buffer = [];
    }

    let excludeIds = new Set(buffer);
    let sceneId = await fetchRandomSceneId(
      sceneFilter,
      totalCount,
      excludeIds
    );

    if (!sceneId && excludeIds.size > 0) {
      clearRandomSceneBuffer(bufferKey);
      excludeIds = new Set();
      sceneId = await fetchRandomSceneId(sceneFilter, totalCount, excludeIds);
    }

    if (sceneId) {
      addToRandomSceneBuffer(bufferKey, sceneId);
    }

    return sceneId;
  } catch (error) {
    console.error("Error getting random scene:", error);
    return null;
  }
};

/** Returns a random scene that is not organised (packed). */
export const getRandomUnorganisedScene = async (): Promise<string | null> => {
  return getRandomSceneWithFilter(RANDOM_SCENE_BUFFER_KEY_UNORGANISED, {
    organized: false,
  });
};

/** Returns a random scene that is not classified yet. */
export const getRandomUnclassifiedScene = async (): Promise<string | null> => {
  return getRandomSceneWithFilter(RANDOM_SCENE_BUFFER_KEY_UNCLASSIFIED, {
    classified: false,
  });
};

export const getRandomScene = async (
  ratingThreshold: number = 55
): Promise<string | null> => {
  return getRandomSceneWithFilter(
    randomSceneBufferKeyForRating(ratingThreshold),
    {
      rating100: {
        modifier: GQL.CriterionModifier.GreaterThan,
        value: ratingThreshold - 1,
      },
    }
  );
};

export const getRandomBestScene = async (
  ratingThreshold: number = 90
): Promise<string | null> => {
  return getRandomScene(ratingThreshold);
};

export const REVIEW_SCENE_QUERY_PARAM = "review";

export function makeReviewSceneLink(sceneId: string): string {
  return `/scenes/${sceneId}?${REVIEW_SCENE_QUERY_PARAM}=true`;
}
