import * as GQL from "src/core/generated-graphql";

/**
 * Selects the next scene to autoplay from similar scenes
 * @param similarScenes Array of similar scenes with their similarity scores
 * @param currentSceneId ID of the current scene to exclude
 * @param viewedSceneIds Array of scene IDs that have already been viewed in this session
 * @returns The selected next scene or null if no suitable scene found
 */
export function selectNextScene(
  similarScenes: Array<{
    __typename?: 'SimilarScene';
    similarity_score: number;
    scene: GQL.SlimSceneDataFragment;
  }> | undefined,
  currentSceneId: string,
  viewedSceneIds: string[] = []
): GQL.SlimSceneDataFragment | null {
  if (!similarScenes || similarScenes.length === 0) {
    return null;
  }

  // Filter out the current scene and already viewed scenes
  const filteredScenes = similarScenes.filter(
    (similarScene: { scene: GQL.SlimSceneDataFragment }) =>
      similarScene.scene.id !== currentSceneId &&
      !viewedSceneIds.includes(similarScene.scene.id)
  );

  if (filteredScenes.length === 0) {
    return null;
  }

  // Sort by similarity score in descending order
  const sortedScenes = filteredScenes.sort(
    (a: { similarity_score: number }, b: { similarity_score: number }) => b.similarity_score - a.similarity_score
  );

  // Take batches of 5 scenes until we find one that's not viewed
  // Start with top 5, then next 5, etc.
  let batchStart = 0;
  const batchSize = 5;

  while (batchStart < sortedScenes.length) {
    const batchEnd = Math.min(batchStart + batchSize, sortedScenes.length);
    const currentBatch = sortedScenes.slice(batchStart, batchEnd);

    if (currentBatch.length > 0) {
      // Randomly select one from the current batch
      const randomIndex = Math.floor(Math.random() * currentBatch.length);
      return currentBatch[randomIndex].scene;
    }

    batchStart += batchSize;
  }

  return null;
}

/**
 * Hook to get the next scene for autoplay
 * @param sceneId Current scene ID
 * @param similarScenes Similar scenes data
 * @param viewedSceneIds Array of scene IDs that have already been viewed in this session
 * @returns The next scene to autoplay or null
 */
export function useNextScene(
  sceneId: string,
  similarScenes: Array<{
    __typename?: 'SimilarScene';
    similarity_score: number;
    scene: GQL.SlimSceneDataFragment;
  }> | undefined,
  viewedSceneIds: string[] = []
): GQL.SlimSceneDataFragment | null {
  return selectNextScene(similarScenes, sceneId, viewedSceneIds);
}
