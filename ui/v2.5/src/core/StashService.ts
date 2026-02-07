/**
 * StashService - Application service layer.
 *
 * Provides typed query/mutation functions for all entities.
 * Uses REST API (via rest-client.ts) and TanStack Query hooks (via rest-hooks.ts).
 *
 * Previously this file used Apollo Client for GraphQL operations.
 * All cache manipulation is now handled by TanStack Query's invalidation system.
 */

import { ListFilterModel } from "../models/list-filter/filter";
import * as GQL from "./generated-graphql";
import { apiGet, apiPost, apiPut, apiDelete } from "./rest-client";
import { getQueryClient, queryKeys } from "./query-client";
import { getSSEClient } from "./sse-client";
import { resolveApiPath } from "./createClient";
import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

// ============================================================
// Utility: wrap mutation hook to accept optional default variables
// ============================================================

type AnyMutationTuple = [
  (opts?: { variables?: any }) => Promise<any>,
  any
];

/**
 * Wraps a mutation hook to accept an optional default input.
 * When the mutation function is called without variables,
 * it uses the defaultInput provided at hook creation time.
 * This matches the Apollo pattern where default variables
 * could be passed to useMutation.
 */
function withDefaultInput<T extends AnyMutationTuple>(
  tuple: T,
  defaultInput?: any
): T {
  if (!defaultInput) return tuple;
  const [mutateFn, result] = tuple;
  const wrappedFn = async (opts?: { variables?: any }) => {
    return mutateFn({ variables: opts?.variables ?? defaultInput });
  };
  return [wrappedFn, result] as T;
}

// ============================================================
// SSE State Hook (replaces WebSocket state)
// ============================================================

export function useSSEState(_client?: unknown) {
  const sse = getSSEClient();
  const [state, setState] = useState<"connecting" | "connected" | "error">(
    sse.connected ? "connected" : "connecting"
  );

  useEffect(() => {
    const unsubConnect = sse.on("connected", () => setState("connected"));
    return () => {
      unsubConnect();
    };
  }, [sse]);

  return { state };
}

// Re-export for backward compat (was useWSState)
export const useWSState = useSSEState;

// ============================================================
// REST response envelope types
// ============================================================

interface DataEnvelope<T> {
  data: T;
}

interface ListEnvelope<T> {
  data: T[];
  count: number;
}

// ============================================================
// Helpers for wrapping REST responses into Apollo-like shape
// ============================================================

async function fetchOne<T>(path: string): Promise<T> {
  const resp = await apiGet<DataEnvelope<T>>(path);
  return resp.data;
}

async function fetchList<T>(
  path: string,
  body?: unknown
): Promise<{ items: T[]; count: number }> {
  const resp = await apiPost<ListEnvelope<T>>(path, body);
  return { items: resp.data, count: resp.count };
}

/** Resolve relative API paths in item to absolute backend URLs */
function resolveItemPaths<T>(item: T): T {
  if (!item || typeof item !== "object") return item;
  const obj = item as any;
  if (obj.paths && typeof obj.paths === "object") {
    const resolved: any = {};
    for (const [k, v] of Object.entries(obj.paths)) {
      resolved[k] = typeof v === "string" ? resolveApiPath(v) : v;
    }
    obj.paths = resolved;
  }
  if (typeof obj.image_path === "string") obj.image_path = resolveApiPath(obj.image_path);
  if (typeof obj.front_image_path === "string") obj.front_image_path = resolveApiPath(obj.front_image_path);
  if (Array.isArray(obj.performers)) {
    obj.performers = obj.performers.map((p: any) => ({
      ...p,
      image_path: typeof p.image_path === "string" ? resolveApiPath(p.image_path) : p.image_path,
    }));
  }
  if (Array.isArray(obj.scene_performers)) {
    obj.scene_performers = obj.scene_performers.map((sp: any) => {
      if (sp?.performer && typeof sp.performer === "object" && typeof sp.performer.image_path === "string") {
        return { ...sp, performer: { ...sp.performer, image_path: resolveApiPath(sp.performer.image_path) } };
      }
      return sp;
    });
  }
  if (obj.studio && typeof obj.studio === "object" && typeof obj.studio.image_path === "string") {
    obj.studio = { ...obj.studio, image_path: resolveApiPath(obj.studio.image_path) };
  }
  if (Array.isArray(obj.groups)) {
    obj.groups = obj.groups.map((g: any) => {
      if (g.group && typeof g.group.front_image_path === "string") {
        return { ...g, group: { ...g.group, front_image_path: resolveApiPath(g.group.front_image_path) } };
      }
      return g;
    });
  }
  if (Array.isArray(obj.tags)) {
    obj.tags = obj.tags.map((t: any) => ({
      ...t,
      image_path: typeof t.image_path === "string" ? resolveApiPath(t.image_path) : t.image_path,
    }));
  }
  if (Array.isArray(obj.scene_markers)) {
    obj.scene_markers = obj.scene_markers.map((m: any) => {
      const out: any = { ...m };
      if (typeof m.stream === "string") out.stream = resolveApiPath(m.stream);
      if (typeof m.preview === "string") out.preview = resolveApiPath(m.preview);
      if (typeof m.screenshot === "string") out.screenshot = resolveApiPath(m.screenshot);
      if (m.primary_tag && typeof m.primary_tag.image_path === "string") {
        out.primary_tag = { ...m.primary_tag, image_path: resolveApiPath(m.primary_tag.image_path) };
      }
      if (Array.isArray(m.tags)) {
        out.tags = m.tags.map((t: any) => ({
          ...t,
          image_path: typeof t.image_path === "string" ? resolveApiPath(t.image_path) : t.image_path,
        }));
      }
      return out;
    });
  }
  return obj;
}
function resolveItemsPaths<T>(items: T[]): T[] {
  return items.map(resolveItemPaths);
}

// Wraps REST result into { data: { key: value } } for backward compat
function wrapResult<T>(
  key: string,
  value: T,
  error?: string,
  errors?: { message: string }[]
): {
  data: Record<string, unknown>;
  error?: string;
  errors?: { message: string }[];
} {
  const result: {
    data: Record<string, unknown>;
    error?: string;
    errors?: { message: string }[];
  } = { data: { [key]: resolveItemPaths(value) } as Record<string, T> };
  if (error !== undefined) {
    result.error = error;
  }
  if (errors !== undefined) {
    result.errors = errors;
  }
  return result;
}

function wrapListResult<T>(
  wrapKey: string,
  listKey: string,
  items: T[],
  count: number,
  extras?: Record<string, unknown>
) {
  return {
    data: {
      [wrapKey]: { count, [listKey]: resolveItemsPaths(items), ...extras },
    },
  };
}

// ============================================================
// Query Client helpers
// ============================================================

function invalidateQueries(...keys: (readonly unknown[])[]) {
  const qc = getQueryClient();
  for (const key of keys) {
    qc.invalidateQueries(key as unknown[]);
  }
}

// Re-export isLoading for backward compatibility
export function isLoading(networkStatus: number) {
  return networkStatus === 1 || networkStatus === 3 || networkStatus === 4;
}

// ============================================================
// Object queries - Scenes
// ============================================================

export const useFindScene = (id: string) => {
  const skip = id === "new" || id === "";
  return GQL.useFindSceneQuery({ variables: { id }, skip });
};

export const useSceneStreams = (id: string) =>
  GQL.useSceneStreamsQuery({ variables: { id } });

export const useFindScenes = (filter?: ListFilterModel) =>
  GQL.useFindScenesQuery({
    skip: filter === undefined,
    variables: {
      filter: filter?.makeFindFilter(),
      scene_filter: filter?.makeFilter(),
    },
  });

export const queryFindScenes = async (filter: ListFilterModel) => {
  const result = await fetchList("/scenes/query", {
    filter: filter.makeFindFilter(),
    scene_filter: filter.makeFilter(),
  });
  return wrapListResult("findScenes", "scenes", result.items, result.count);
};

export const queryFindScenesByID = async (sceneIDs: number[]) => {
  const result = await fetchList("/scenes/query", { scene_ids: sceneIDs });
  return wrapListResult("findScenes", "scenes", result.items, result.count);
};

export const queryFindScenesForSelect = async (filter: ListFilterModel) => {
  const result = await fetchList("/scenes/query", {
    filter: filter.makeFindFilter(),
    scene_filter: filter.makeFilter(),
  });
  return wrapListResult("findScenes", "scenes", result.items, result.count);
};

export const queryFindScenesByIDForSelect = async (sceneIDs: string[]) => {
  const result = await fetchList("/scenes/query", { ids: sceneIDs });
  return wrapListResult("findScenes", "scenes", result.items, result.count);
};

export const querySceneByPathRegex = async (filter: GQL.FindFilterType) => {
  const result = await fetchList("/scenes/query", { filter });
  return wrapListResult("findScenesByPathRegex", "scenes", result.items, result.count);
};

// ============================================================
// Object queries - Images
// ============================================================

export const useFindImage = (id: string) =>
  GQL.useFindImageQuery({ variables: { id } });

export const useFindImages = (filter?: ListFilterModel) =>
  GQL.useFindImagesQuery({
    skip: filter === undefined,
    variables: {
      filter: filter?.makeFindFilter(),
      image_filter: filter?.makeFilter(),
    },
  });

export const queryFindImages = async (filter: ListFilterModel) => {
  const result = await fetchList("/images/query", {
    filter: filter.makeFindFilter(),
    image_filter: filter.makeFilter(),
  });
  return wrapListResult("findImages", "images", result.items, result.count);
};

// ============================================================
// Object queries - Groups
// ============================================================

export const useFindGroup = (id: string) => {
  const skip = id === "new" || id === "";
  return GQL.useFindGroupQuery({ variables: { id }, skip });
};

export const useFindGroups = (filter?: ListFilterModel) =>
  GQL.useFindGroupsQuery({
    skip: filter === undefined,
    variables: {
      filter: filter?.makeFindFilter(),
      group_filter: filter?.makeFilter(),
    },
  });

export const queryFindGroups = async (filter: ListFilterModel) => {
  const result = await fetchList("/groups/query", {
    filter: filter.makeFindFilter(),
    group_filter: filter.makeFilter(),
  });
  return wrapListResult("findGroups", "groups", result.items, result.count);
};

export const queryFindGroupsByIDForSelect = async (groupIDs: string[]) => {
  const result = await fetchList("/groups/query", { ids: groupIDs });
  return wrapListResult("findGroups", "groups", result.items, result.count);
};

export const queryFindGroupsForSelect = async (filter: ListFilterModel) => {
  const result = await fetchList("/groups/query", {
    filter: filter.makeFindFilter(),
    group_filter: filter.makeFilter(),
  });
  return wrapListResult("findGroups", "groups", result.items, result.count);
};

// ============================================================
// Object queries - Scene Markers
// ============================================================

export const useFindSceneMarkers = (filter?: ListFilterModel) =>
  GQL.useFindSceneMarkersQuery({
    skip: filter === undefined,
    variables: {
      filter: filter?.makeFindFilter(),
      scene_marker_filter: filter?.makeFilter(),
    },
  });

export const queryFindSceneMarkers = async (filter: ListFilterModel) => {
  const result = await fetchList("/scene-markers/query", {
    filter: filter.makeFindFilter(),
    scene_marker_filter: filter.makeFilter(),
  });
  return wrapListResult("findSceneMarkers", "scene_markers", result.items, result.count);
};

export const useMarkerStrings = () => GQL.useMarkerStringsQuery();

// ============================================================
// Object queries - Galleries
// ============================================================

export const useFindGallery = (id: string) => {
  const skip = id === "new" || id === "";
  return GQL.useFindGalleryQuery({ variables: { id }, skip });
};

export const useFindGalleryImageID = (id: string, index: number) =>
  GQL.useFindGalleryImageIdQuery({ variables: { id, index } });

export const useFindGalleries = (filter?: ListFilterModel) =>
  GQL.useFindGalleriesQuery({
    skip: filter === undefined,
    variables: {
      filter: filter?.makeFindFilter(),
      gallery_filter: filter?.makeFilter(),
    },
  });

export const queryFindGalleries = async (filter: ListFilterModel) => {
  const result = await fetchList("/galleries/query", {
    filter: filter.makeFindFilter(),
    gallery_filter: filter.makeFilter(),
  });
  return wrapListResult("findGalleries", "galleries", result.items, result.count);
};

export const queryFindGalleriesForSelect = async (filter: ListFilterModel) => {
  const result = await fetchList("/galleries/query", {
    filter: filter.makeFindFilter(),
    gallery_filter: filter.makeFilter(),
  });
  return wrapListResult("findGalleries", "galleries", result.items, result.count);
};

export const queryFindGalleriesByIDForSelect = async (galleryIDs: string[]) => {
  const result = await fetchList("/galleries/query", { ids: galleryIDs });
  return wrapListResult("findGalleries", "galleries", result.items, result.count);
};

// ============================================================
// Object queries - Games
// ============================================================

export const useFindGame = (id: string) => {
  const skip = id === "new" || id === "";
  return GQL.useFindGameQuery({ variables: { id }, skip });
};

export const useFindGames = (filter?: ListFilterModel) =>
  GQL.useFindGamesQuery({
    skip: filter === undefined,
    variables: {
      filter: filter?.makeFindFilter(),
      game_filter: filter?.makeFilter(),
    },
  });

export const queryFindGames = async (filter: ListFilterModel) => {
  const result = await fetchList("/games/query", {
    filter: filter.makeFindFilter(),
    game_filter: filter.makeFilter(),
  });
  return wrapListResult("findGames", "games", result.items, result.count);
};

// ============================================================
// Object queries - Performers
// ============================================================

export const useFindPerformer = (id: string) => {
  const skip = id === "new" || id === "";
  return GQL.useFindPerformerQuery({ variables: { id }, skip });
};

export const queryFindPerformer = async (id: string) => {
  const performer = await fetchOne(`/performers/${id}`);
  return wrapResult("findPerformer", performer);
};

export const useFindPerformers = (filter?: ListFilterModel) =>
  GQL.useFindPerformersQuery({
    skip: filter === undefined,
    variables: {
      filter: filter?.makeFindFilter(),
      performer_filter: filter?.makeFilter(),
    },
  });

export const queryFindPerformers = async (filter: ListFilterModel) => {
  const result = await fetchList("/performers/query", {
    filter: filter.makeFindFilter(),
    performer_filter: filter.makeFilter(),
  });
  return wrapListResult("findPerformers", "performers", result.items, result.count);
};

export const queryFindPerformersByIDForSelect = async (performerIDs: string[]) => {
  const result = await fetchList("/performers/query", { ids: performerIDs });
  return wrapListResult("findPerformers", "performers", result.items, result.count);
};

export const queryFindPerformersForSelect = async (filter: ListFilterModel) => {
  const result = await fetchList("/performers/query", {
    filter: filter.makeFindFilter(),
    performer_filter: filter.makeFilter(),
  });
  return wrapListResult("findPerformers", "performers", result.items, result.count);
};

// ============================================================
// Object queries - Studios
// ============================================================

export const useFindStudio = (id: string) => {
  const skip = id === "new" || id === "";
  return GQL.useFindStudioQuery({ variables: { id }, skip });
};

export const queryFindStudio = async (id: string) => {
  const studio = await fetchOne(`/studios/${id}`);
  return wrapResult("findStudio", studio);
};

export const useFindStudios = (filter?: ListFilterModel) =>
  GQL.useFindStudiosQuery({
    skip: filter === undefined,
    variables: {
      filter: filter?.makeFindFilter(),
      studio_filter: filter?.makeFilter(),
    },
  });

export const queryFindStudios = async (filter: ListFilterModel) => {
  const result = await fetchList("/studios/query", {
    filter: filter.makeFindFilter(),
    studio_filter: filter.makeFilter(),
  });
  return wrapListResult("findStudios", "studios", result.items, result.count);
};

export const queryFindStudiosByIDForSelect = async (studioIDs: string[]) => {
  const result = await fetchList("/studios/query", { ids: studioIDs });
  return wrapListResult("findStudios", "studios", result.items, result.count);
};

export const queryFindStudiosForSelect = async (filter: ListFilterModel) => {
  const result = await fetchList("/studios/query", {
    filter: filter.makeFindFilter(),
    studio_filter: filter.makeFilter(),
  });
  return wrapListResult("findStudios", "studios", result.items, result.count);
};

// ============================================================
// Object queries - Tags
// ============================================================

export const useFindTag = (id: string) => {
  const skip = id === "new" || id === "";
  return GQL.useFindTagQuery({ variables: { id }, skip });
};

export const useFindTags = (filter?: ListFilterModel) =>
  GQL.useFindTagsQuery({
    skip: filter === undefined,
    variables: {
      filter: filter?.makeFindFilter(),
      tag_filter: filter?.makeFilter(),
    },
  });

export const queryFindTags = async (filter: ListFilterModel) => {
  const result = await fetchList("/tags/query", {
    filter: filter.makeFindFilter(),
    tag_filter: filter.makeFilter(),
  });
  return wrapListResult("findTags", "tags", result.items, result.count);
};

export const queryFindTagsByIDForSelect = async (tagIDs: string[]) => {
  const result = await fetchList("/tags/query", { ids: tagIDs });
  return wrapListResult("findTags", "tags", result.items, result.count);
};

export const queryFindTagsForSelect = async (filter: ListFilterModel) => {
  const result = await fetchList("/tags/query", {
    filter: filter.makeFindFilter(),
    tag_filter: filter.makeFilter(),
  });
  return wrapListResult("findTags", "tags", result.items, result.count);
};

// ============================================================
// Color Presets & Saved Filters
// ============================================================

export const useFindColorPresets = () => GQL.useFindColorPresetsQuery();

export const useColorPresetCreate = () => GQL.useColorPresetCreateMutation();
export const useColorPresetUpdate = () => GQL.useColorPresetUpdateMutation();
export const useColorPresetDestroy = () => GQL.useColorPresetDestroyMutation();

export const useFindTagColors = () => GQL.useFindTagColorsQuery();

export const useFindSavedFilter = (id: string) =>
  GQL.useFindSavedFilterQuery({ variables: { id } });

export const useFindSavedFilters = (mode?: GQL.FilterMode) =>
  GQL.useFindSavedFiltersQuery({ variables: { mode } });

export const useFindDefaultFilter = (
  mode: GQL.FilterMode,
  skip?: boolean
) =>
  GQL.useFindDefaultFilterQuery({
    variables: { mode },
    skip,
  });

// ============================================================
// Scene Mutations
// ============================================================

export const useSceneCreate = () => GQL.useSceneCreateMutation();

export const mutateCreateScene = async (input: GQL.SceneCreateInput) => {
  const result = await apiPost<DataEnvelope<GQL.Scene>>("/scenes", input);
  return wrapResult("sceneCreate", result.data);
};

export const useSceneUpdate = () =>
  GQL.useSceneUpdateMutation();

export const useBulkSceneUpdate = (defaultInput?: any) =>
  withDefaultInput(GQL.useBulkSceneUpdateMutation(), defaultInput);

export const useScenesUpdate = (defaultInput?: any) =>
  withDefaultInput(GQL.useScenesUpdateMutation(), defaultInput);

export const useSceneDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useSceneDestroyMutation(), defaultInput);

export const useScenesDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useScenesDestroyMutation(), defaultInput);

export const useSceneAddO = () =>
  GQL.useSceneAddOMutation();

export const useSceneDeleteO = () =>
  GQL.useSceneDeleteOMutation();

export const useSceneResetO = (defaultInput?: any) =>
  withDefaultInput(GQL.useSceneResetOMutation(), defaultInput ? { id: defaultInput } : undefined);

export const useSceneResetActivity = (...args: any[]) =>
  withDefaultInput(GQL.useSceneResetActivityMutation(), args.length ? { id: args[0], reset_resume: args[1], reset_duration: args[2] } : undefined);

export const useSceneIncrementOmg = (defaultId?: any) =>
  withDefaultInput(GQL.useSceneIncrementOmgMutation(), defaultId ? { id: defaultId } : undefined);

export const useSceneDecrementOmg = () =>
  GQL.useSceneDecrementOmgMutation();

export const useSceneResetOmg = (defaultId?: any) =>
  withDefaultInput(GQL.useSceneResetOmgMutation(), defaultId ? { id: defaultId } : undefined);

export const useSceneAddOmg = (defaultId?: any) =>
  withDefaultInput(GQL.useSceneAddOmgMutation(), defaultId ? { id: defaultId } : undefined);

export const useSceneDeleteOmg = (defaultId?: any) =>
  withDefaultInput(GQL.useSceneDeleteOmgMutation(), defaultId ? { id: defaultId } : undefined);

export const useSceneGenerateScreenshot = () =>
  GQL.useSceneGenerateScreenshotMutation();

export const useOpenInExternalPlayer = () =>
  GQL.useOpenInExternalPlayerMutation();

export const useScanVideoFileThreats = () =>
  GQL.useScanVideoFileThreatsMutation();

export const useScanAllScenesForThreats = () =>
  GQL.useScanAllScenesForThreatsMutation();

export const useSceneSaveActivity = () =>
  GQL.useSceneSaveActivityMutation();

export const useSceneAddPlay = () =>
  GQL.useSceneAddPlayMutation();

export const useSceneDeletePlay = () =>
  GQL.useSceneDeletePlayMutation();

export const useSceneResetPlayCount = () =>
  GQL.useSceneResetPlayCountMutation();

// Aliases that bind scene id for convenience (backward compat)
export const useSceneIncrementO = (id?: string) => {
  const [mutate, result] = GQL.useSceneAddOMutation();
  const fn = (opts?: { variables?: Record<string, unknown> }) =>
    mutate({ ...opts, variables: { id: id ?? "", ...opts?.variables } } as any);
  return [fn, result] as const;
};

export const useSceneDecrementO = (id?: string) => {
  const [mutate, result] = GQL.useSceneDeleteOMutation();
  const fn = (opts?: { variables?: Record<string, unknown> }) =>
    mutate({ ...opts, variables: { id: id ?? "", ...opts?.variables } } as any);
  return [fn, result] as const;
};

export const useSceneIncrementPlayCount = () =>
  GQL.useSceneAddPlayMutation();

export const useSceneDecrementPlayCount = () =>
  GQL.useSceneDeletePlayMutation();

export const useSceneSaveFilteredScreenshot = () =>
  GQL.useSceneSaveFilteredScreenshotMutation();

export const useSceneReduceResolution = (defaultInput?: any) =>
  withDefaultInput(GQL.useSceneReduceResolutionMutation(), defaultInput);

export const useSceneRegenerateSprites = (defaultInput?: any) =>
  withDefaultInput(GQL.useSceneRegenerateSpritesMutation(), defaultInput);

export const useSceneTrimVideo = (defaultInput?: any) =>
  withDefaultInput(GQL.useSceneTrimMutation(), defaultInput);

export const mutateSceneSetPrimaryFile = async (sceneId: string, fileId: string) => {
  const result = await apiPost(`/scenes/${sceneId}/set-primary-file`, { file_id: fileId });
  return wrapResult("sceneSetPrimaryFile", result);
};

export const mutateSceneAssignFile = async (sceneId: string, fileId: string) => {
  const result = await apiPost(`/scenes/${sceneId}/assign-file`, { file_id: fileId });
  return wrapResult("sceneAssignFile", result);
};

// ============================================================
// Image Mutations
// ============================================================

export const useImageUpdate = () =>
  GQL.useImageUpdateMutation();

export const useBulkImageUpdate = (defaultInput?: any) =>
  withDefaultInput(GQL.useBulkImageUpdateMutation(), defaultInput);

export const useImagesDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useImagesDestroyMutation(), defaultInput);

export const useImageAddO = () =>
  GQL.useImageIncrementOMutation();

export const useImageDeleteO = () =>
  GQL.useImageDecrementOMutation();

export const useImageResetO = () =>
  GQL.useImageResetOMutation();

export const useImageAddOmg = () =>
  GQL.useImageIncrementOmgMutation();

export const useImageDeleteOmg = () =>
  GQL.useImageDecrementOmgMutation();

export const useImageResetOmg = () =>
  GQL.useImageResetOmgMutation();

// Aliases that bind image id for convenience (backward compat)
export const useImageIncrementO = (id?: string) => {
  const [mutate, result] = GQL.useImageIncrementOMutation();
  const fn = (opts?: { variables?: Record<string, unknown> }) =>
    mutate({ ...opts, variables: { id: id ?? "", ...opts?.variables } } as any);
  return [fn, result] as const;
};

export const useImageIncrementOmg = (id?: string) => {
  const [mutate, result] = GQL.useImageIncrementOmgMutation();
  const fn = (opts?: { variables?: Record<string, unknown> }) =>
    mutate({ ...opts, variables: { id: id ?? "", ...opts?.variables } } as any);
  return [fn, result] as const;
};

export const mutateImageResetO = async (id: string) => {
  const result = await apiPost(`/images/${id}/o/reset`);
  return wrapResult("imageResetO", result);
};

export const mutateImageSetPrimaryFile = async (imageId: string, fileId: string) => {
  const result = await apiPost(`/images/${imageId}/set-primary-file`, { file_id: fileId });
  return wrapResult("imageSetPrimaryFile", result);
};

export const mutateImageIncrementO = async (id: string) => {
  const result = await apiPost(`/images/${id}/o/increment`);
  return wrapResult("imageIncrementO", result);
};

export const mutateImageDecrementO = async (id: string) => {
  const result = await apiPost(`/images/${id}/o/decrement`);
  return wrapResult("imageDecrementO", result);
};

// ============================================================
// Group Mutations
// ============================================================

export const useGroupCreate = () =>
  GQL.useGroupCreateMutation();

export const useGroupUpdate = () =>
  GQL.useGroupUpdateMutation();

export const useBulkGroupUpdate = (defaultInput?: any) =>
  withDefaultInput(GQL.useBulkGroupUpdateMutation(), defaultInput);

export const useGroupDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useGroupDestroyMutation(), defaultInput);

export const useGroupsDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useGroupsDestroyMutation(), defaultInput);

export const useReorderSubGroups = () =>
  GQL.useReorderSubGroupsMutation();

export const useAddGroupSubGroups = () => {
  const [addSubGroups] = GQL.useAddGroupSubGroupsMutation();
  return [addSubGroups] as const;
};

export const useRemoveGroupSubGroups = () => {
  const [removeSubGroups] = GQL.useRemoveGroupSubGroupsMutation();
  return [removeSubGroups] as const;
};

// Convenience aliases for sub-group operations (backward compat)
export const useAddSubGroups = () => {
  const [addSubGroups] = GQL.useAddGroupSubGroupsMutation();
  return async (groupId: string, subGroups: { group_id: string; description?: string }[]) =>
    addSubGroups({ variables: { input: { containing_group_id: groupId, sub_groups: subGroups } } } as any);
};

export const useRemoveSubGroups = () => {
  const [removeSubGroups] = GQL.useRemoveGroupSubGroupsMutation();
  return async (groupId: string, subGroupIds: string[]) =>
    removeSubGroups({ variables: { input: { containing_group_id: groupId, sub_group_ids: subGroupIds } } } as any);
};

export const useReorderSubGroupsMutation = () =>
  GQL.useReorderSubGroupsMutation();

// ============================================================
// Scene Marker Mutations
// ============================================================

export const useSceneMarkerCreate = () =>
  GQL.useSceneMarkerCreateMutation();

export const useSceneMarkerUpdate = () =>
  GQL.useSceneMarkerUpdateMutation();

export const useSceneMarkerDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useSceneMarkerDestroyMutation(), defaultInput);

export const useSceneMarkersDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useSceneMarkersDestroyMutation(), defaultInput);

// ============================================================
// Gallery Mutations
// ============================================================

export const useGalleryCreate = () =>
  GQL.useGalleryCreateMutation();

export const useGalleryAddO = () =>
  GQL.useGalleryAddOMutation();

export const useGalleryDeleteO = () =>
  GQL.useGalleryDeleteOMutation();

export const useGalleryResetO = (defaultId?: any) =>
  withDefaultInput(GQL.useGalleryResetOMutation(), defaultId ? { id: defaultId } : undefined);

export const useGalleryAddOmg = () =>
  GQL.useGalleryIncrementOmgMutation();

export const useGalleryDeleteOmg = () =>
  GQL.useGalleryDecrementOmgMutation();

export const useGalleryResetOmg = () =>
  GQL.useGalleryResetOmgMutation();

export const useGalleryUpdate = () =>
  GQL.useGalleryUpdateMutation();

export const useBulkGalleryUpdate = (defaultInput?: any) =>
  withDefaultInput(GQL.useBulkGalleryUpdateMutation(), defaultInput);

export const useGalleryDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useGalleryDestroyMutation(), defaultInput);

export const useGalleryChapterCreate = () =>
  GQL.useGalleryChapterCreateMutation();

export const useGalleryChapterUpdate = () =>
  GQL.useGalleryChapterUpdateMutation();

export const useGalleryChapterDestroy = () =>
  GQL.useGalleryChapterDestroyMutation();

export const useGalleryIncrementPlayCount = () =>
  GQL.useGalleryIncrementPlayMutation();

export const useGalleryDecrementPlayCount = (id: string) =>
  GQL.useGalleryDeletePlayMutation();

export const useGalleryResetPlayCount = (id: string) =>
  GQL.useGalleryResetPlayCountMutation();

// Aliases that bind gallery id for convenience (backward compat)
export const useGalleryIncrementO = (id?: string) => {
  const [mutate, result] = GQL.useGalleryAddOMutation();
  const fn = (opts?: { variables?: Record<string, unknown> }) =>
    mutate({ ...opts, variables: { id: id ?? "", ...opts?.variables } } as any);
  return [fn, result] as const;
};

export const useGalleryDecrementO = (id?: string) => {
  const [mutate, result] = GQL.useGalleryDeleteOMutation();
  const fn = (opts?: { variables?: Record<string, unknown> }) =>
    mutate({ ...opts, variables: { id: id ?? "", ...opts?.variables } } as any);
  return [fn, result] as const;
};

export const useGalleryIncrementOmg = (id?: string) => {
  const [mutate, result] = GQL.useGalleryIncrementOmgMutation();
  const fn = (opts?: { variables?: Record<string, unknown> }) =>
    mutate({ ...opts, variables: { id: id ?? "", ...opts?.variables } } as any);
  return [fn, result] as const;
};

export const mutateResetGalleryCover = async (input: { gallery_id: string }) => {
  const result = await apiPost(`/galleries/${input.gallery_id}/reset-cover`);
  return wrapResult("resetGalleryCover", result);
};

export const mutateAddGalleryImages = async (input: { gallery_id: string; image_ids: string[] }) => {
  const result = await apiPost(`/galleries/${input.gallery_id}/images`, { image_ids: input.image_ids });
  return wrapResult("addGalleryImages", result);
};

export const mutateRemoveGalleryImages = async (input: { gallery_id: string; image_ids: string[] }) => {
  const result = await apiDelete(`/galleries/${input.gallery_id}/images`, { image_ids: input.image_ids });
  return wrapResult("removeGalleryImages", result);
};

export const mutateSetGalleryCover = async (input: { gallery_id: string; cover_image_id: string }) => {
  const result = await apiPost(`/galleries/${input.gallery_id}/set-cover`, { cover_image_id: input.cover_image_id });
  return wrapResult("setGalleryCover", result);
};

export const mutateGallerySetPrimaryFile = async (galleryId: string, fileId: string) => {
  const result = await apiPost(`/galleries/${galleryId}/set-primary-file`, { file_id: fileId });
  return wrapResult("gallerySetPrimaryFile", result);
};

// ============================================================
// Performer Mutations
// ============================================================

export const usePerformerCreate = () =>
  GQL.usePerformerCreateMutation();

export const usePerformerUpdate = () =>
  GQL.usePerformerUpdateMutation();

export const useBulkPerformerUpdate = (defaultInput?: any) =>
  withDefaultInput(GQL.useBulkPerformerUpdateMutation(), defaultInput);

export const usePerformerDestroy = () =>
  GQL.usePerformerDestroyMutation();

export const usePerformersDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.usePerformersDestroyMutation(), defaultInput);

export const usePerformerProfileImageCreate = () =>
  GQL.usePerformerProfileImageCreateMutation();

export const usePerformerProfileImageUpdate = () =>
  GQL.usePerformerProfileImageUpdateMutation();

export const usePerformerProfileImageDestroy = () =>
  GQL.usePerformerProfileImageDestroyMutation();

// ============================================================
// Studio Mutations
// ============================================================

export const useStudioCreate = () =>
  GQL.useStudioCreateMutation();

export const useStudioUpdate = () =>
  GQL.useStudioUpdateMutation();

export const useStudioDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useStudioDestroyMutation(), defaultInput);

export const useStudiosDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useStudiosDestroyMutation(), defaultInput);

// ============================================================
// Tag Mutations
// ============================================================

export const useTagCreate = () =>
  GQL.useTagCreateMutation();

export const useTagUpdate = () =>
  GQL.useTagUpdateMutation();

export const useBulkTagUpdate = (defaultInput?: any) =>
  withDefaultInput(GQL.useBulkTagUpdateMutation(), defaultInput);

export const useTagDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useTagDestroyMutation(), defaultInput);

export const useTagsDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useTagsDestroyMutation(), defaultInput);

export const useTagsMerge = () =>
  GQL.useTagsMergeMutation();

// ============================================================
// Saved Filter Mutations
// ============================================================

export const useSaveFilter = () => {
  const [saveFilterMutation] = GQL.useSaveFilterMutation();
  const qc = useQueryClient();

  const saveFilter = async (input: GQL.SaveFilterInput) => {
    const result = await saveFilterMutation({ variables: { input } });
    qc.invalidateQueries(queryKeys.filters.all);
    return result;
  };

  return saveFilter;
};

export const useSetDefaultFilter = () => {
  const [setDefaultFilter] = GQL.useSetDefaultFilterMutation();
  const qc = useQueryClient();

  return async (input: GQL.SetDefaultFilterInput) => {
    const result = await setDefaultFilter({ variables: { input } });
    qc.invalidateQueries(queryKeys.filters.all);
    return result;
  };
};

export const useDestroySavedFilter = () =>
  GQL.useDestroySavedFilterMutation();

export const useSavedFilterDestroy = (defaultInput?: any) =>
  withDefaultInput(GQL.useDestroySavedFilterMutation(), defaultInput);

// ============================================================
// Scrapers
// ============================================================

export const useListSceneScrapers = () => GQL.useListSceneScrapersQuery();

export const queryScrapeScene = async (
  scraperID: string,
  scene: GQL.SceneUpdateInput
) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scrapers/scene", {
    source: { scraper_id: scraperID },
    input: scene,
  });
  return wrapResult("scrapeSingleScene", result.data);
};

export const queryScrapeSceneQuery = async (
  scraperID: string,
  query: string
) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scrapers/scene", {
    source: { scraper_id: scraperID },
    input: { query },
  });
  return wrapResult("scrapeSingleScene", result.data);
};

export const queryScrapeSceneQueryFragment = async (
  source: GQL.ScraperSourceInput,
  input: GQL.ScrapeSingleSceneInput
) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scrapers/scene", {
    source,
    input,
  });
  return wrapResult("scrapeSingleScene", result.data);
};

export const queryScrapeSceneURL = async (url: string) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scrapers/url", { url });
  return wrapResult("scrapeURL", result.data);
};

export const queryScrapeGallery = async (
  scraperID: string,
  gallery: GQL.GalleryUpdateInput
) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scrapers/gallery", {
    source: { scraper_id: scraperID },
    input: gallery,
  });
  return wrapResult("scrapeSingleGallery", result.data);
};

export const queryScrapeGalleryURL = async (url: string) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scrapers/url", { url });
  return wrapResult("scrapeURL", result.data);
};

export const queryScrapeGroupURL = async (url: string) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scrapers/url", { url });
  return wrapResult("scrapeURL", result.data);
};

export const useListPerformerScrapers = () =>
  GQL.useListPerformerScrapersQuery();

export const queryScrapePerformer = async (
  scraperID: string,
  performer: GQL.ScrapedPerformerInput
) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scrapers/performer", {
    source: { scraper_id: scraperID },
    input: performer,
  });
  return wrapResult("scrapeSinglePerformer", result.data);
};

export const queryScrapePerformerURL = async (url: string) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scrapers/url", { url });
  return wrapResult("scrapeURL", result.data);
};

export const queryScrapeImageURL = async (url: string) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scrapers/url", { url });
  return wrapResult("scrapeURL", result.data);
};

export const queryScrapeImage = async (scraperId: string, imageId: string) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scrapers/image", {
    source: { scraper_id: scraperId },
    input: { image_id: imageId },
  });
  const data = result.data;
  return wrapResult("scrapeSingleImage", Array.isArray(data) ? data : [data]);
};

export const useScrapePerformerList = (scraperId: string, query: string) => {
  const [data, setData] = useState<{ scrapeSinglePerformer: unknown[] } | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!query || !scraperId) {
      setData(null);
      return;
    }

    let cancelled = false;
    setLoading(true);
    apiPost<DataEnvelope<unknown[]>>("/scrapers/performer", {
      source: { scraper_id: scraperId },
      input: { query },
    })
      .then((result) => {
        if (!cancelled) {
          setData({ scrapeSinglePerformer: result.data ?? [] });
        }
      })
      .catch(() => {
        if (!cancelled) setData(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [scraperId, query]);

  return { data, loading };
};

export const useListGroupScrapers = () => GQL.useListGroupScrapersQuery();

export const useListGalleryScrapers = () => GQL.useListGalleryScrapersQuery();

export const useListImageScrapers = () => GQL.useListImageScrapersQuery();

export const mutateSubmitStashBoxSceneDraft = async (
  input: GQL.StashBoxDraftSubmissionInput
) => {
  const result = await apiPost<DataEnvelope<unknown>>("/stash-box/scene-draft", input);
  return wrapResult("submitStashBoxSceneDraft", result.data);
};

export const mutateSubmitStashBoxPerformerDraft = async (
  input: GQL.StashBoxDraftSubmissionInput
) => {
  const result = await apiPost<DataEnvelope<unknown>>("/stash-box/performer-draft", input);
  return wrapResult("submitStashBoxPerformerDraft", result.data);
};

export const stashBoxSceneBatchQuery = async (
  sceneIds: string[],
  stashBoxEndpoint: string,
  stashBoxIndex?: number
) => {
  const body: Record<string, unknown> = {
    stash_box_endpoint: stashBoxEndpoint,
    scene_ids: sceneIds,
  };
  if (stashBoxIndex !== undefined) {
    body.stash_box_index = stashBoxIndex;
  }
  const result = await apiPost<DataEnvelope<unknown>>("/stash-box/scene-batch", body);
  return wrapResult("scrapeMultiScenes", result.data);
};

export const stashBoxPerformerQuery = async (
  performerInput: string | { q?: string; performer_ids?: string[] },
  stashBoxEndpoint: string,
  stashBoxIndex?: number
) => {
  const body: Record<string, unknown> = {
    stash_box_endpoint: stashBoxEndpoint,
  };
  if (typeof performerInput === "string") {
    body.q = performerInput;
  } else {
    if (performerInput.q !== undefined) {
      body.q = performerInput.q;
    }
    if (performerInput.performer_ids !== undefined) {
      body.performer_ids = performerInput.performer_ids;
    }
  }
  if (stashBoxIndex !== undefined) {
    body.stash_box_index = stashBoxIndex;
  }
  const result = await apiPost<DataEnvelope<unknown>>("/stash-box/performer", body);
  // The API can return either scrapeMultiPerformers or scrapeSinglePerformer
  // Check the response structure to determine which key to use
  const responseData = result.data as Record<string, unknown>;
  if (Array.isArray(responseData.scrapeMultiPerformers)) {
    return wrapResult("scrapeMultiPerformers", responseData.scrapeMultiPerformers);
  } else if (Array.isArray(responseData.scrapeSinglePerformer)) {
    return wrapResult("scrapeSinglePerformer", responseData.scrapeSinglePerformer);
  }
  // Default to scrapeSinglePerformer for backward compatibility
  return wrapResult("scrapeSinglePerformer", Array.isArray(result.data) ? result.data : [result.data]);
};

export const stashBoxStudioQuery = async (
  studioInput: string | { q?: string },
  stashBoxEndpoint: string,
  stashBoxIndex?: number
) => {
  const body: Record<string, unknown> = {
    stash_box_endpoint: stashBoxEndpoint,
  };
  if (typeof studioInput === "string") {
    body.q = studioInput;
  } else if (studioInput.q !== undefined) {
    body.q = studioInput.q;
  }
  if (stashBoxIndex !== undefined) {
    body.stash_box_index = stashBoxIndex;
  }
  const result = await apiPost<DataEnvelope<unknown>>("/stash-box/studio", body);
  return wrapResult("scrapeSingleStudio", Array.isArray(result.data) ? result.data : [result.data]);
};

// ============================================================
// Configuration
// ============================================================

export const useConfiguration = () => GQL.useConfigurationQuery();

export const usePlugins = () => GQL.usePluginsQuery();

export const usePluginTasks = () => GQL.usePluginTasksQuery();

export const useStats = () => GQL.useStatsQuery();

export const useOCountStats = () => GQL.useOCountStatsQuery();

export const useVersion = () => GQL.useVersionQuery();

export const useLatestVersion = () =>
  GQL.useLatestVersionQuery({
    notifyOnNetworkStatusChange: true,
    errorPolicy: "ignore",
  });

export const useDLNAStatus = () =>
  GQL.useDlnaStatusQuery({
    fetchPolicy: "no-cache",
  });

export const useJobQueue = () =>
  GQL.useJobQueueQuery({
    fetchPolicy: "no-cache",
  });

export const useLogs = () =>
  GQL.useLogsQuery({
    fetchPolicy: "no-cache",
  });

export const queryLogs = async () => {
  const logs = await fetchOne("/logs");
  return wrapResult("logs", logs);
};

export const useSystemStatus = () => GQL.useSystemStatusQuery();

export const refetchSystemStatus = () => {
  invalidateQueries(queryKeys.systemStatus);
};

export const useJobsSubscribe = () => GQL.useJobsSubscribeSubscription();

export const useLoggingSubscribe = () => GQL.useLoggingSubscribeSubscription();

// ============================================================
// Scraper/Plugin Management
// ============================================================

export const mutateReloadScrapers = async () => {
  await apiPost("/scrapers/reload");
  invalidateQueries(queryKeys.scrapers.all, queryKeys.packages.installed("scraper"));
  return { data: { reloadScrapers: true } };
};

export const mutateReloadPlugins = async () => {
  await apiPost("/plugins/reload");
  invalidateQueries(queryKeys.plugins.all, queryKeys.packages.installed("plugin"));
  return { data: { reloadPlugins: true } };
};

type BoolMap = { [key: string]: boolean };

export const mutateSetPluginsEnabled = async (enabledMap: BoolMap) => {
  await apiPut("/plugins/enabled", enabledMap);
  invalidateQueries(queryKeys.plugins.all);
  return { data: { setPluginsEnabled: true } };
};

// ============================================================
// Config Mutations
// ============================================================

export const useConfigureGeneral = () =>
  GQL.useConfigureGeneralMutation();

export const useConfigureInterface = () =>
  GQL.useConfigureInterfaceMutation();

export const useGenerateAPIKey = () =>
  GQL.useGenerateApiKeyMutation();

export const useConfigureDefaults = () =>
  GQL.useConfigureDefaultsMutation();

export const useConfigureUI = () =>
  GQL.useConfigureUiMutation();

export const useConfigureUISetting = () =>
  GQL.useConfigureUiSettingMutation();

export const useConfigureScraping = () =>
  GQL.useConfigureScrapingMutation();

export const useConfigureDLNA = () =>
  GQL.useConfigureDlnaMutation();

export const useConfigurePlugin = () =>
  GQL.useConfigurePluginMutation();

export const useEnableDLNA = () => GQL.useEnableDlnaMutation();
export const useDisableDLNA = () => GQL.useDisableDlnaMutation();
export const useAddTempDLNAIP = () => GQL.useAddTempDlnaipMutation();
export const useRemoveTempDLNAIP = () => GQL.useRemoveTempDlnaipMutation();

// ============================================================
// Jobs
// ============================================================

export const mutateStopJob = async (jobID: string) => {
  await apiPost(`/jobs/${jobID}/stop`);
  invalidateQueries(queryKeys.jobs.all);
  return { data: { stopJob: true } };
};

// ============================================================
// Setup / Migrate
// ============================================================

export const mutateSetup = async (input: GQL.SetupInput) => {
  const result = await apiPost<DataEnvelope<unknown>>("/system/setup", input);
  invalidateQueries(queryKeys.config, queryKeys.systemStatus);
  return wrapResult("setup", result.data);
};

export const mutateMigrate = async (input: GQL.MigrateInput) => {
  await apiPost("/system/migrate", input);
  return { data: { migrate: true } };
};

export function postMigrate() {
  invalidateQueries(queryKeys.config, queryKeys.systemStatus);
}

// ============================================================
// Packages
// ============================================================

export const useInstalledScraperPackages = <T extends boolean>(
  loadUpgrades: T
) => {
  if (loadUpgrades) {
    return GQL.useInstalledScraperPackagesStatusQuery();
  }
  return GQL.useInstalledScraperPackagesQuery();
};

export const queryAvailableScraperPackages = async (source: string) => {
  const result = await fetchOne(`/packages/available?type=scraper&source=${encodeURIComponent(source)}`);
  return wrapResult("availablePackages", result);
};

export const mutateInstallScraperPackages = async (
  packages: GQL.PackageSpecInput[]
) => {
  await apiPost("/packages/install", { type: "scraper", packages });
  invalidateQueries(queryKeys.packages.installed("scraper"), queryKeys.scrapers.all);
  return { data: { installPackages: true } };
};

export const mutateUpdateScraperPackages = async (packages: GQL.PackageSpecInput[]) => {
  await apiPost("/packages/update", { type: "scraper", packages });
  invalidateQueries(queryKeys.packages.installed("scraper"), queryKeys.scrapers.all);
  return { data: { updatePackages: true } };
};

export const mutateUninstallScraperPackages = async (
  packages: GQL.PackageSpecInput[]
) => {
  await apiPost("/packages/uninstall", { type: "scraper", packages });
  invalidateQueries(queryKeys.packages.installed("scraper"), queryKeys.scrapers.all);
  return { data: { uninstallPackages: true } };
};

export const useInstalledPluginPackages = <T extends boolean>(
  loadUpgrades: T
) => {
  if (loadUpgrades) {
    return GQL.useInstalledPluginPackagesStatusQuery();
  }
  return GQL.useInstalledPluginPackagesQuery();
};

export const queryAvailablePluginPackages = async (source: string) => {
  const result = await fetchOne(`/packages/available?type=plugin&source=${encodeURIComponent(source)}`);
  return wrapResult("availablePackages", result);
};

export const mutateInstallPluginPackages = async (packages: GQL.PackageSpecInput[]) => {
  await apiPost("/packages/install", { type: "plugin", packages });
  invalidateQueries(queryKeys.packages.installed("plugin"), queryKeys.plugins.all);
  return { data: { installPackages: true } };
};

export const mutateUpdatePluginPackages = async (packages: GQL.PackageSpecInput[]) => {
  await apiPost("/packages/update", { type: "plugin", packages });
  invalidateQueries(queryKeys.packages.installed("plugin"), queryKeys.plugins.all);
  return { data: { updatePackages: true } };
};

export const mutateUninstallPluginPackages = async (
  packages: GQL.PackageSpecInput[]
) => {
  await apiPost("/packages/uninstall", { type: "plugin", packages });
  invalidateQueries(queryKeys.packages.installed("plugin"), queryKeys.plugins.all);
  return { data: { uninstallPackages: true } };
};

// ============================================================
// Tasks (metadata operations)
// ============================================================

export const mutateMetadataScan = async (input: GQL.ScanMetadataInput) => {
  const result = await apiPost("/metadata/scan", input);
  return { data: { metadataScan: result } };
};

export const mutateMetadataIdentify = async (input: GQL.IdentifyMetadataInput) => {
  const result = await apiPost("/metadata/identify", input);
  return { data: { metadataIdentify: result } };
};

export const mutateMetadataAutoTag = async (input: GQL.AutoTagMetadataInput) => {
  const result = await apiPost("/metadata/auto-tag", input);
  return { data: { metadataAutoTag: result } };
};

export const mutateMetadataGenerate = async (input: GQL.GenerateMetadataInput) => {
  const result = await apiPost("/metadata/generate", input);
  return { data: { metadataGenerate: result } };
};

export const mutateMetadataClean = async (input: GQL.CleanMetadataInput) => {
  const result = await apiPost("/metadata/clean", input);
  return { data: { metadataClean: result } };
};

export const mutateCleanGenerated = async (input: GQL.CleanGeneratedInput) => {
  const result = await apiPost("/metadata/clean-generated", input);
  return { data: { metadataCleanGenerated: result } };
};

export const mutateRunPluginTask = async (
  pluginId: string,
  taskName: string,
  args?: GQL.Scalars["Map"]["input"]
) => {
  const result = await apiPost(`/plugins/${pluginId}/run`, {
    task_name: taskName,
    args,
  });
  return { data: { runPluginTask: result } };
};

export const mutateMetadataExport = async () => {
  const result = await apiPost("/metadata/export");
  return { data: { metadataExport: result } };
};

export const mutateExportObjects = async (input: GQL.ExportObjectsInput) => {
  const result = await apiPost("/metadata/export-objects", input);
  return { data: { exportObjects: result } };
};

export const mutateMetadataImport = async () => {
  const result = await apiPost("/metadata/import");
  return { data: { metadataImport: result } };
};

export const mutateImportObjects = async (input: GQL.ImportObjectsInput) => {
  const result = await apiPost("/metadata/import-objects", input);
  return { data: { importObjects: result } };
};

// ============================================================
// Database operations
// ============================================================

export const mutateBackupDatabase = async (input: GQL.BackupDatabaseInput) => {
  const result = await apiPost("/database/backup", input);
  return { data: { backupDatabase: result } };
};

export const mutateAnonymiseDatabase = async (input: GQL.AnonymiseDatabaseInput) => {
  const result = await apiPost("/database/anonymise", input);
  return { data: { anonymiseDatabase: result } };
};

export const mutateOptimiseDatabase = async () => {
  await apiPost("/database/optimise");
  return { data: { optimiseDatabase: true } };
};

export const mutateMigrateHashNaming = async () => {
  await apiPost("/database/migrate-hash-naming");
  return { data: { migrateHashNaming: true } };
};

export const mutateMigrateSceneScreenshots = async (
  input: GQL.MigrateSceneScreenshotsInput
) => {
  await apiPost("/database/migrate-screenshots", input);
  return { data: { migrateSceneScreenshots: true } };
};

export const mutateMigrateBlobs = async (input: GQL.MigrateBlobsInput) => {
  await apiPost("/database/migrate-blobs", input);
  return { data: { migrateBlobs: true } };
};

export const mutateRecalculateSceneSimilarities = async (sceneID?: string) => {
  await apiPost(`/scenes/${sceneID}/recalculate-similarity`);
  invalidateQueries(queryKeys.scenes.all);
  return { data: { recalculateSceneSimilarities: true } };
};

// ============================================================
// Game Mutations
// ============================================================

export const useGameCreate = () => GQL.useGameCreateMutation();
export const useGameUpdate = () => GQL.useGameUpdateMutation();
export const useGameDestroy = () => GQL.useGameDestroyMutation();
export const useGameAddO = () => GQL.useGameIncrementOMutation();
export const useGameDeleteO = () => GQL.useGameDecrementOMutation();
export const useGameResetO = () => GQL.useGameResetOMutation();
export const useGameAddOmg = () => GQL.useGameAddOmgMutation();
export const useGameDeleteOmg = () => GQL.useGameDeleteOmgMutation();
export const useGameResetOmg = () => GQL.useGameResetOmgMutation();
export const useGameAddView = () => GQL.useGameAddViewMutation();
export const useGameDeleteView = () => GQL.useGameDeleteViewMutation();
export const useGameResetViews = () => GQL.useGameResetViewsMutation();
export const useGameIncrementView = () => GQL.useGameIncrementViewMutation();

// ============================================================
// File Operations
// ============================================================

export const mutateDeleteFiles = async (fileIds: string[]) => {
  const result = await apiDelete("/files/delete", { ids: fileIds });
  return wrapResult("deleteFiles", result);
};

// ============================================================
// Misc
// ============================================================

export const useDirectory = (path?: string) =>
  GQL.useDirectoryQuery({ variables: { path } });

export const queryParseSceneFilenames = async (
  filter: GQL.FindFilterType,
  config: GQL.SceneParserInput
) => {
  const result = await apiPost<DataEnvelope<unknown>>("/scenes/parse-filenames", {
    filter,
    config,
  });
  return wrapResult("parseSceneFilenames", result.data);
};

// Backward compat exports
export const scraperMutationImpactedQueries = queryKeys.scrapers.all;
export const pluginMutationImpactedQueries = queryKeys.plugins.all;
export const performerMutationImpactedQueries: unknown[] = [];
export const studioMutationImpactedQueries: unknown[] = [];

// Re-export useFindJobQuery from GQL for backward compat
export const useFindJobQuery = GQL.useFindJobQuery;

// Re-export getClient as no-op for backward compat
export const getClient = () => null;
export const getWSClient = () => getSSEClient();

// Re-export evictQueries as no-op for backward compatibility
export function evictQueries(_cache: unknown, _queries: unknown[]) {
  // No-op - TanStack Query handles cache invalidation automatically
}
