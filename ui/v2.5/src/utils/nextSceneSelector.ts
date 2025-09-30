import * as GQL from "src/core/generated-graphql";

/**
 * Selects the next scene to autoplay from similar scenes
 * @param similarScenes Array of similar scenes with their similarity scores
 * @param currentSceneId ID of the current scene to exclude
 * @returns The selected next scene or null if no suitable scene found
 */
export function selectNextScene(
  similarScenes: GQL.FindSimilarScenesQuery["findScene"]["similar_scenes"],
  currentSceneId: string
): GQL.SlimSceneDataFragment | null {
  if (!similarScenes || similarScenes.length === 0) {
    return null;
  }

  // Filter out the current scene (though it shouldn't be in similar scenes anyway)
  const filteredScenes = similarScenes.filter(
    (similarScene) => similarScene.scene.id !== currentSceneId
  );

  if (filteredScenes.length === 0) {
    return null;
  }

  // Sort by similarity score in descending order
  const sortedScenes = filteredScenes.sort(
    (a, b) => b.similarity_score - a.similarity_score
  );

  // Take top 5 scenes
  const topScenes = sortedScenes.slice(0, 5);

  if (topScenes.length === 0) {
    return null;
  }

  // Randomly select one from the top scenes
  const randomIndex = Math.floor(Math.random() * topScenes.length);
  return topScenes[randomIndex].scene;
}

/**
 * Hook to get the next scene for autoplay
 * @param sceneId Current scene ID
 * @param similarScenes Similar scenes data
 * @returns The next scene to autoplay or null
 */
export function useNextScene(
  sceneId: string,
  similarScenes: GQL.FindSimilarScenesQuery["findScene"]["similar_scenes"] | undefined
): GQL.SlimSceneDataFragment | null {
  return selectNextScene(similarScenes || [], sceneId);
}
