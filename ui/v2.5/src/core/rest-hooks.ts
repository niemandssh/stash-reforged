/**
 * REST-based React hooks providing Apollo Client-compatible API.
 * Uses TanStack Query (React Query) v4 under the hood.
 *
 * These hooks are designed to be drop-in replacements for the
 * Apollo-generated hooks that were previously in generated-graphql.ts.
 */

import {
  useQuery,
  useMutation,
  useQueryClient,
  QueryKey,
  UseQueryOptions,
} from "@tanstack/react-query";
import { useEffect, useState, useCallback, useRef, useMemo } from "react";
import { apiGet, apiPost, apiPut, apiDelete } from "./rest-client";
import { resolveApiPath } from "./createClient";
import {
  getSSEClient,
  SSEJobStatusUpdate,
  SSELogEntry,
} from "./sse-client";
import { queryKeys, getQueryClient } from "./query-client";
import type * as T from "./types";

// ============================================================
// Local type definitions for types not in generated types.ts
// ============================================================

export type FindDefaultFilterQuery = { findDefaultFilter?: T.SavedFilter | null };
export type FindDefaultFilterQueryVariables = { mode: T.FilterMode };

export type FindViewHistoryQuery = { findViewHistory?: { count: number; items: unknown[] } };
export type FindViewHistoryQueryVariables = { filter?: T.FindFilterType; historyFilter?: unknown };

export type SetDefaultFilterMutation = { setDefaultFilter: boolean };
export type SetDefaultFilterMutationVariables = { input: T.SetDefaultFilterInput };

// ============================================================
// Apollo-compatible types and adapters
// ============================================================

/** Minimal Apollo QueryHookOptions compatible interface */
export interface QueryHookOptions<TData = unknown, TVars = unknown> {
  variables?: TVars;
  skip?: boolean;
  fetchPolicy?: string;
  notifyOnNetworkStatusChange?: boolean;
  errorPolicy?: string;
  pollInterval?: number;
}

/** Minimal Apollo LazyQueryHookOptions compatible interface */
export type LazyQueryHookOptions<
  TData = unknown,
  TVars = unknown,
> = QueryHookOptions<TData, TVars>;

/** Minimal Apollo MutationHookOptions compatible interface */
export interface MutationHookOptions<TData = unknown, TVars = unknown> {
  variables?: TVars;
  refetchQueries?: unknown[];
  update?: (cache: unknown, result: unknown, options?: unknown) => void;
  onCompleted?: (data: TData) => void;
  onError?: (error: Error) => void;
}

/** Minimal Apollo SubscriptionHookOptions */
export interface SubscriptionHookOptions<TData = unknown, TVars = unknown> {
  variables?: TVars;
  skip?: boolean;
}

/** Apollo-compatible query result */
export interface QueryResult<TData = unknown> {
  data: TData | undefined;
  loading: boolean;
  error: Error | undefined;
  refetch: (...args: unknown[]) => Promise<unknown>;
  networkStatus: number;
  called: boolean;
  previousData?: TData;
}

/** Apollo-compatible mutation result tuple */
/** Apollo-compatible FetchResult returned by mutation functions */
export type FetchResult<TData = Record<string, unknown>> = {
  data?: TData | null;
  errors?: readonly { message: string }[];
};

export type MutationTuple<TData, TVars> = [
  (options?: { variables?: TVars }) => Promise<FetchResult<TData>>,
  {
    data: TData | undefined;
    loading: boolean;
    error: Error | undefined;
    called: boolean;
    reset: () => void;
  },
];

/** Apollo-compatible lazy query result tuple */
export type LazyQueryTuple<TData, TVars> = [
  (options?: { variables?: TVars }) => void,
  {
    data: TData | undefined;
    loading: boolean;
    error: Error | undefined;
    called: boolean;
  },
];

// Apollo NetworkStatus enum values
const NetworkStatus = {
  loading: 1,
  setVariables: 2,
  fetchMore: 3,
  refetch: 4,
  poll: 6,
  ready: 7,
  error: 8,
} as const;

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
// Generic hook builders
// ============================================================

function useRestQuery<TData>(
  queryKey: QueryKey,
  queryFn: () => Promise<TData>,
  options?: {
    enabled?: boolean;
    cacheTime?: number;
    staleTime?: number;
    refetchOnWindowFocus?: boolean;
  }
): QueryResult<TData> {
  const result = useQuery<TData, Error>(queryKey, queryFn, {
    enabled: options?.enabled,
    cacheTime: options?.cacheTime,
    staleTime: options?.staleTime,
    refetchOnWindowFocus: options?.refetchOnWindowFocus,
  });

  // Memoize the return value to keep a stable object reference
  // and prevent infinite re-render loops in consumers that put
  // the result into useEffect dependency arrays.
  return useMemo(
    () => ({
      data: result.data,
      loading: result.isLoading,
      error: result.error ?? undefined,
      refetch: result.refetch as (...args: unknown[]) => Promise<unknown>,
      networkStatus: result.isLoading
        ? NetworkStatus.loading
        : result.isSuccess
          ? NetworkStatus.ready
          : NetworkStatus.error,
      called: true,
      previousData: result.isPreviousData ? result.data : undefined,
    }),
    [result.data, result.isLoading, result.error, result.refetch, result.isSuccess, result.isPreviousData]
  );
}

function useRestLazyQuery<TData, TVars>(
  buildQueryKey: (vars?: TVars) => QueryKey,
  buildQueryFn: (vars?: TVars) => () => Promise<TData>
): LazyQueryTuple<TData, TVars> {
  const [vars, setVars] = useState<TVars | undefined>(undefined);
  const [called, setCalled] = useState(false);

  const result = useQuery<TData, Error>(
    buildQueryKey(vars),
    buildQueryFn(vars),
    { enabled: called && vars !== undefined }
  );

  const execute = useCallback(
    (options?: { variables?: TVars }) => {
      setVars(options?.variables);
      setCalled(true);
    },
    []
  );

  return [
    execute,
    {
      data: result.data,
      loading: result.isLoading && called,
      error: result.error ?? undefined,
      called,
    },
  ];
}

function useRestMutation<TData, TVars>(
  mutationFn: (variables: TVars) => Promise<TData>,
  options?: {
    invalidateKeys?: QueryKey[];
  }
): MutationTuple<TData, TVars> {
  const queryClient = useQueryClient();

  const mutation = useMutation<TData, Error, TVars>(mutationFn, {
    onSuccess: () => {
      if (options?.invalidateKeys) {
        for (const key of options.invalidateKeys) {
          queryClient.invalidateQueries(key);
        }
      }
    },
  });

  const mutateFn = async (opts?: { variables?: TVars }): Promise<FetchResult<TData>> => {
    if (!opts?.variables) return { data: undefined };
    const data = await mutation.mutateAsync(opts.variables);
    return { data };
  };

  return [
    mutateFn,
    {
      data: mutation.data,
      loading: mutation.isLoading,
      error: mutation.error ?? undefined,
      called: mutation.isLoading || mutation.isSuccess || mutation.isError,
      reset: mutation.reset,
    },
  ];
}

// ============================================================
// REST data fetchers (unwrap envelope)
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

/**
 * Resolve all relative API paths in a scene/image/gallery/performer object
 * to absolute backend URLs. This ensures <img src> and similar attributes
 * point to the correct backend host in dev mode.
 */
function resolveItemPaths<T>(item: T): T {
  if (!item || typeof item !== "object") return item;
  const obj = item as any;

  // Resolve paths object (scene.paths, image.paths, gallery.paths)
  if (obj.paths && typeof obj.paths === "object") {
    const resolved: any = {};
    for (const [k, v] of Object.entries(obj.paths)) {
      resolved[k] = typeof v === "string" ? resolveApiPath(v) : v;
    }
    obj.paths = resolved;
  }

  // Resolve image_path on performers, studios, tags
  if (typeof obj.image_path === "string") {
    obj.image_path = resolveApiPath(obj.image_path);
  }

  // Resolve front_image_path on groups
  if (typeof obj.front_image_path === "string") {
    obj.front_image_path = resolveApiPath(obj.front_image_path);
  }

  // Resolve nested performers
  if (Array.isArray(obj.performers)) {
    obj.performers = obj.performers.map((p: any) => ({
      ...p,
      image_path: typeof p.image_path === "string" ? resolveApiPath(p.image_path) : p.image_path,
    }));
  }

  // Resolve scene_performers[].performer.image_path (used by performer cards on scene detail)
  if (Array.isArray(obj.scene_performers)) {
    obj.scene_performers = obj.scene_performers.map((sp: any) => {
      if (sp?.performer && typeof sp.performer === "object" && typeof sp.performer.image_path === "string") {
        return { ...sp, performer: { ...sp.performer, image_path: resolveApiPath(sp.performer.image_path) } };
      }
      return sp;
    });
  }

  // Resolve nested studio
  if (obj.studio && typeof obj.studio === "object" && typeof obj.studio.image_path === "string") {
    obj.studio = { ...obj.studio, image_path: resolveApiPath(obj.studio.image_path) };
  }

  // Resolve nested groups
  if (Array.isArray(obj.groups)) {
    obj.groups = obj.groups.map((g: any) => {
      if (g.group && typeof g.group.front_image_path === "string") {
        return { ...g, group: { ...g.group, front_image_path: resolveApiPath(g.group.front_image_path) } };
      }
      return g;
    });
  }

  // Resolve nested tags (scene.tags) - image_path for tag cards/placeholders
  if (Array.isArray(obj.tags)) {
    obj.tags = obj.tags.map((t: any) => ({
      ...t,
      image_path: typeof t.image_path === "string" ? resolveApiPath(t.image_path) : t.image_path,
    }));
  }

  // Resolve scene_markers streams/previews/screenshots and tag image_path
  if (Array.isArray(obj.scene_markers)) {
    obj.scene_markers = obj.scene_markers.map((m: any) => {
      const out: any = {
        ...m,
        stream: typeof m.stream === "string" ? resolveApiPath(m.stream) : m.stream,
        preview: typeof m.preview === "string" ? resolveApiPath(m.preview) : m.preview,
        screenshot: typeof m.screenshot === "string" ? resolveApiPath(m.screenshot) : m.screenshot,
      };
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

async function postAndUnwrap<T>(path: string, body?: unknown): Promise<T> {
  const resp = await apiPost<DataEnvelope<T>>(path, body);
  return resp.data;
}

async function putAndUnwrap<T>(path: string, body?: unknown): Promise<T> {
  const resp = await apiPut<DataEnvelope<T>>(path, body);
  return resp.data;
}

async function deleteAndUnwrap<T>(path: string, body?: unknown): Promise<T> {
  const resp = await apiDelete<DataEnvelope<T>>(path, body);
  return resp.data;
}

// Simple fetch without envelope (for endpoints that return raw data)
async function fetchRaw<T>(path: string): Promise<T> {
  return apiGet<T>(path);
}

async function postRaw<T>(path: string, body?: unknown): Promise<T> {
  return apiPost<T>(path, body);
}

// ============================================================
// SCENE QUERY HOOKS
// ============================================================

export function useFindSceneQuery(
  baseOptions?: QueryHookOptions<T.FindSceneQuery, T.FindSceneQueryVariables>
) {
  const id = baseOptions?.variables?.id;
  return useRestQuery<T.FindSceneQuery>(
    queryKeys.scenes.detail(id ?? ""),
    async () => {
      const scene = await fetchOne(`/scenes/${id}`);
      return { findScene: resolveItemPaths(scene) } as T.FindSceneQuery;
    },
    { enabled: !!id && !baseOptions?.skip }
  );
}

export function useSceneStreamsQuery(
  baseOptions?: QueryHookOptions<T.SceneStreamsQuery, T.SceneStreamsQueryVariables>
) {
  const id = baseOptions?.variables?.id;
  return useRestQuery<T.SceneStreamsQuery>(
    ["scenes", "streams", id],
    async () => {
      const streams = await fetchOne(`/scenes/${id}/streams`);
      return { findScene: { sceneStreams: Array.isArray(streams) ? streams : [] } } as T.SceneStreamsQuery;
    },
    { enabled: !!id && !baseOptions?.skip }
  );
}

export function useFindScenesQuery(
  baseOptions?: QueryHookOptions<T.FindScenesQuery, T.FindScenesQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindScenesQuery>(
    ["scenes", "list", vars],
    async () => {
      const result = await fetchList("/scenes/query", {
        filter: vars?.filter,
        scene_filter: vars?.scene_filter,
        scene_ids: vars?.scene_ids,
      });
      return {
        findScenes: {
          __typename: "FindScenesResultType" as const,
          count: result.count,
          scenes: resolveItemsPaths(result.items),
        },
      } as T.FindScenesQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindScenesForSelectQuery(
  baseOptions?: QueryHookOptions<T.FindScenesForSelectQuery, T.FindScenesForSelectQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindScenesForSelectQuery>(
    ["scenes", "list-select", vars],
    async () => {
      const result = await fetchList("/scenes/query", {
        filter: vars?.filter,
        scene_filter: vars?.scene_filter,
        ids: vars?.ids,
      });
      return {
        findScenes: {
          __typename: "FindScenesResultType" as const,
          count: result.count,
          scenes: resolveItemsPaths(result.items),
        },
      } as T.FindScenesForSelectQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindScenesByPathRegexQuery(
  baseOptions?: QueryHookOptions<T.FindScenesByPathRegexQuery, T.FindScenesByPathRegexQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindScenesByPathRegexQuery>(
    ["scenes", "path-regex", vars],
    async () => {
      const result = await fetchList("/scenes/query", { filter: vars?.filter });
      return {
        findScenesByPathRegex: {
          count: result.count,
          scenes: resolveItemsPaths(result.items),
        },
      } as T.FindScenesByPathRegexQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindSimilarScenesQuery(
  baseOptions?: QueryHookOptions<T.FindSimilarScenesQuery, T.FindSimilarScenesQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindSimilarScenesQuery>(
    ["scenes", "similar", vars?.id],
    async () => {
      const items = await fetchOne(`/scenes/${vars?.id}/similar`);
      return { findSimilarScenes: resolveItemsPaths(Array.isArray(items) ? items : []) } as T.FindSimilarScenesQuery;
    },
    { enabled: !!vars?.id && !baseOptions?.skip }
  );
}

export function useFindDuplicateScenesQuery(
  baseOptions?: QueryHookOptions<T.FindDuplicateScenesQuery, T.FindDuplicateScenesQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindDuplicateScenesQuery>(
    ["scenes", "duplicates", vars],
    async () => {
      const result = await postAndUnwrap("/scenes/duplicates", {
        distance: vars?.distance,
        duration_diff: vars?.duration_diff,
      });
      return { findDuplicateScenes: result } as T.FindDuplicateScenesQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

// ============================================================
// IMAGE QUERY HOOKS
// ============================================================

export function useFindImageQuery(
  baseOptions?: QueryHookOptions<T.FindImageQuery, T.FindImageQueryVariables>
) {
  const id = baseOptions?.variables?.id;
  return useRestQuery<T.FindImageQuery>(
    queryKeys.images.detail(id ?? ""),
    async () => {
      const image = await fetchOne(`/images/${id}`);
      return { findImage: resolveItemPaths(image) } as T.FindImageQuery;
    },
    { enabled: !!id && !baseOptions?.skip }
  );
}

export function useFindImagesQuery(
  baseOptions?: QueryHookOptions<T.FindImagesQuery, T.FindImagesQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindImagesQuery>(
    ["images", "list", vars],
    async () => {
      const result = await fetchList("/images/query", {
        filter: vars?.filter,
        image_filter: vars?.image_filter,
        image_ids: vars?.image_ids,
      });
      return {
        findImages: {
          __typename: "FindImagesResultType" as const,
          count: result.count,
          megapixels: 0,
          filesize: 0,
          images: resolveItemsPaths(result.items),
        },
      } as T.FindImagesQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindImagesLazyQuery(
  baseOptions?: LazyQueryHookOptions<T.FindImagesQuery, T.FindImagesQueryVariables>
) {
  return useRestLazyQuery<T.FindImagesQuery, T.FindImagesQueryVariables>(
    (vars) => ["images", "list-lazy", vars],
    (vars) => async () => {
      const result = await fetchList("/images/query", {
        filter: vars?.filter,
        image_filter: vars?.image_filter,
        image_ids: vars?.image_ids,
      });
      return {
        findImages: {
          __typename: "FindImagesResultType" as const,
          count: result.count,
          megapixels: 0,
          filesize: 0,
          images: resolveItemsPaths(result.items),
        },
      } as T.FindImagesQuery;
    }
  );
}

// ============================================================
// GALLERY QUERY HOOKS
// ============================================================

export function useFindGalleryQuery(
  baseOptions?: QueryHookOptions<T.FindGalleryQuery, T.FindGalleryQueryVariables>
) {
  const id = baseOptions?.variables?.id;
  return useRestQuery<T.FindGalleryQuery>(
    queryKeys.galleries.detail(id ?? ""),
    async () => {
      const gallery = await fetchOne(`/galleries/${id}`);
      return { findGallery: resolveItemPaths(gallery) } as T.FindGalleryQuery;
    },
    { enabled: !!id && !baseOptions?.skip }
  );
}

export function useFindGalleryImageIdQuery(
  baseOptions?: QueryHookOptions<T.FindGalleryImageIdQuery, T.FindGalleryImageIdQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindGalleryImageIdQuery>(
    ["galleries", "image-id", vars?.id, vars?.index],
    async () => {
      const result = await fetchOne(`/galleries/${vars?.id}?imageIndex=${vars?.index ?? 0}`);
      return { findGallery: resolveItemPaths(result) } as T.FindGalleryImageIdQuery;
    },
    { enabled: !!vars?.id && !baseOptions?.skip }
  );
}

export function useFindGalleriesQuery(
  baseOptions?: QueryHookOptions<T.FindGalleriesQuery, T.FindGalleriesQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindGalleriesQuery>(
    ["galleries", "list", vars],
    async () => {
      const result = await fetchList("/galleries/query", {
        filter: vars?.filter,
        gallery_filter: vars?.gallery_filter,
      });
      return {
        findGalleries: {
          __typename: "FindGalleriesResultType" as const,
          count: result.count,
          galleries: resolveItemsPaths(result.items),
        },
      } as T.FindGalleriesQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindGalleriesForSelectQuery(
  baseOptions?: QueryHookOptions<T.FindGalleriesForSelectQuery, T.FindGalleriesForSelectQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindGalleriesForSelectQuery>(
    ["galleries", "list-select", vars],
    async () => {
      const result = await fetchList("/galleries/query", {
        filter: vars?.filter,
        gallery_filter: vars?.gallery_filter,
        ids: vars?.ids,
      });
      return {
        findGalleries: {
          __typename: "FindGalleriesResultType" as const,
          count: result.count,
          galleries: resolveItemsPaths(result.items),
        },
      } as T.FindGalleriesForSelectQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

// ============================================================
// PERFORMER QUERY HOOKS
// ============================================================

export function useFindPerformerQuery(
  baseOptions?: QueryHookOptions<T.FindPerformerQuery, T.FindPerformerQueryVariables>
) {
  const id = baseOptions?.variables?.id;
  return useRestQuery<T.FindPerformerQuery>(
    queryKeys.performers.detail(id ?? ""),
    async () => {
      const performer = await fetchOne(`/performers/${id}`);
      return { findPerformer: resolveItemPaths(performer) } as T.FindPerformerQuery;
    },
    { enabled: !!id && !baseOptions?.skip }
  );
}

export function useFindPerformersQuery(
  baseOptions?: QueryHookOptions<T.FindPerformersQuery, T.FindPerformersQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindPerformersQuery>(
    ["performers", "list", vars],
    async () => {
      const result = await fetchList("/performers/query", {
        filter: vars?.filter,
        performer_filter: vars?.performer_filter,
        performer_ids: vars?.performer_ids,
      });
      return {
        findPerformers: {
          __typename: "FindPerformersResultType" as const,
          count: result.count,
          performers: resolveItemsPaths(result.items),
        },
      } as T.FindPerformersQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindPerformersForSelectQuery(
  baseOptions?: QueryHookOptions<T.FindPerformersForSelectQuery, T.FindPerformersForSelectQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindPerformersForSelectQuery>(
    ["performers", "list-select", vars],
    async () => {
      const result = await fetchList("/performers/query", {
        filter: vars?.filter,
        performer_filter: vars?.performer_filter,
        ids: vars?.ids,
      });
      return {
        findPerformers: {
          __typename: "FindPerformersResultType" as const,
          count: result.count,
          performers: resolveItemsPaths(result.items),
        },
      } as T.FindPerformersForSelectQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

// ============================================================
// STUDIO QUERY HOOKS
// ============================================================

export function useFindStudioQuery(
  baseOptions?: QueryHookOptions<T.FindStudioQuery, T.FindStudioQueryVariables>
) {
  const id = baseOptions?.variables?.id;
  return useRestQuery<T.FindStudioQuery>(
    queryKeys.studios.detail(id ?? ""),
    async () => {
      const studio = await fetchOne(`/studios/${id}`);
      return { findStudio: resolveItemPaths(studio) } as T.FindStudioQuery;
    },
    { enabled: !!id && !baseOptions?.skip }
  );
}

export function useFindStudiosQuery(
  baseOptions?: QueryHookOptions<T.FindStudiosQuery, T.FindStudiosQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindStudiosQuery>(
    ["studios", "list", vars],
    async () => {
      const result = await fetchList("/studios/query", {
        filter: vars?.filter,
        studio_filter: vars?.studio_filter,
      });
      return {
        findStudios: {
          __typename: "FindStudiosResultType" as const,
          count: result.count,
          studios: resolveItemsPaths(result.items),
        },
      } as T.FindStudiosQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindStudiosForSelectQuery(
  baseOptions?: QueryHookOptions<T.FindStudiosForSelectQuery, T.FindStudiosForSelectQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindStudiosForSelectQuery>(
    ["studios", "list-select", vars],
    async () => {
      const result = await fetchList("/studios/query", {
        filter: vars?.filter,
        studio_filter: vars?.studio_filter,
        ids: vars?.ids,
      });
      return {
        findStudios: {
          __typename: "FindStudiosResultType" as const,
          count: result.count,
          studios: resolveItemsPaths(result.items),
        },
      } as T.FindStudiosForSelectQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

// ============================================================
// TAG QUERY HOOKS
// ============================================================

export function useFindTagQuery(
  baseOptions?: QueryHookOptions<T.FindTagQuery, T.FindTagQueryVariables>
) {
  const id = baseOptions?.variables?.id;
  return useRestQuery<T.FindTagQuery>(
    queryKeys.tags.detail(id ?? ""),
    async () => {
      const tag = await fetchOne(`/tags/${id}`);
      return { findTag: resolveItemPaths(tag) } as T.FindTagQuery;
    },
    { enabled: !!id && !baseOptions?.skip }
  );
}

export function useFindTagsQuery(
  baseOptions?: QueryHookOptions<T.FindTagsQuery, T.FindTagsQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindTagsQuery>(
    ["tags", "list", vars],
    async () => {
      const result = await fetchList("/tags/query", {
        filter: vars?.filter,
        tag_filter: vars?.tag_filter,
      });
      return {
        findTags: {
          __typename: "FindTagsResultType" as const,
          count: result.count,
          tags: resolveItemsPaths(result.items),
        },
      } as T.FindTagsQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindTagsForSelectQuery(
  baseOptions?: QueryHookOptions<T.FindTagsForSelectQuery, T.FindTagsForSelectQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindTagsForSelectQuery>(
    ["tags", "list-select", vars],
    async () => {
      const result = await fetchList("/tags/query", {
        filter: vars?.filter,
        tag_filter: vars?.tag_filter,
        ids: vars?.ids,
      });
      return {
        findTags: {
          __typename: "FindTagsResultType" as const,
          count: result.count,
          tags: resolveItemsPaths(result.items),
        },
      } as T.FindTagsForSelectQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindTagColorsQuery(
  baseOptions?: QueryHookOptions<T.FindTagColorsQuery, T.FindTagColorsQueryVariables>
) {
  return useRestQuery<T.FindTagColorsQuery>(
    ["tags", "colors"],
    async () => {
      const colors = await fetchOne<string[]>("/tags/colors");
      return { findTagColors: colors } as T.FindTagColorsQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

// ============================================================
// GROUP QUERY HOOKS
// ============================================================

export function useFindGroupQuery(
  baseOptions?: QueryHookOptions<T.FindGroupQuery, T.FindGroupQueryVariables>
) {
  const id = baseOptions?.variables?.id;
  return useRestQuery<T.FindGroupQuery>(
    queryKeys.groups.detail(id ?? ""),
    async () => {
      const group = await fetchOne(`/groups/${id}`);
      return { findGroup: resolveItemPaths(group) } as T.FindGroupQuery;
    },
    { enabled: !!id && !baseOptions?.skip }
  );
}

export function useFindGroupsQuery(
  baseOptions?: QueryHookOptions<T.FindGroupsQuery, T.FindGroupsQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindGroupsQuery>(
    ["groups", "list", vars],
    async () => {
      const result = await fetchList("/groups/query", {
        filter: vars?.filter,
        group_filter: vars?.group_filter,
      });
      return {
        findGroups: {
          __typename: "FindGroupsResultType" as const,
          count: result.count,
          groups: resolveItemsPaths(result.items),
        },
      } as T.FindGroupsQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindGroupsForSelectQuery(
  baseOptions?: QueryHookOptions<T.FindGroupsForSelectQuery, T.FindGroupsForSelectQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindGroupsForSelectQuery>(
    ["groups", "list-select", vars],
    async () => {
      const result = await fetchList("/groups/query", {
        filter: vars?.filter,
        group_filter: vars?.group_filter,
        ids: vars?.ids,
      });
      return {
        findGroups: {
          __typename: "FindGroupsResultType" as const,
          count: result.count,
          groups: resolveItemsPaths(result.items),
        },
      } as T.FindGroupsForSelectQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

// ============================================================
// GAME QUERY HOOKS
// ============================================================

export function useFindGameQuery(
  baseOptions?: QueryHookOptions<T.FindGameQuery, T.FindGameQueryVariables>
) {
  const id = baseOptions?.variables?.id;
  return useRestQuery<T.FindGameQuery>(
    queryKeys.games.detail(id ?? ""),
    async () => {
      const game = await fetchOne(`/games/${id}`);
      return { findGame: resolveItemPaths(game) } as T.FindGameQuery;
    },
    { enabled: !!id && !baseOptions?.skip }
  );
}

export function useFindGamesQuery(
  baseOptions?: QueryHookOptions<T.FindGamesQuery, T.FindGamesQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindGamesQuery>(
    ["games", "list", vars],
    async () => {
      const result = await fetchList("/games/query", {
        filter: vars?.filter,
        game_filter: vars?.game_filter,
      });
      return {
        findGames: {
          __typename: "FindGamesResultType" as const,
          count: result.count,
          games: resolveItemsPaths(result.items),
        },
      } as T.FindGamesQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

// ============================================================
// SCENE MARKER QUERY HOOKS
// ============================================================

export function useFindSceneMarkersQuery(
  baseOptions?: QueryHookOptions<T.FindSceneMarkersQuery, T.FindSceneMarkersQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindSceneMarkersQuery>(
    ["scene-markers", "list", vars],
    async () => {
      const result = await fetchList("/scene-markers/query", {
        filter: vars?.filter,
        scene_marker_filter: vars?.scene_marker_filter,
      });
      return {
        findSceneMarkers: {
          __typename: "FindSceneMarkersResultType" as const,
          count: result.count,
          scene_markers: resolveItemsPaths(result.items),
        },
      } as T.FindSceneMarkersQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useMarkerStringsQuery(
  baseOptions?: QueryHookOptions<T.MarkerStringsQuery, T.MarkerStringsQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.MarkerStringsQuery>(
    ["scene-markers", "strings", vars],
    async () => {
      const strings = await fetchOne("/scene-markers/strings");
      return { markerStrings: strings } as T.MarkerStringsQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindSceneMarkerTagsQuery(
  baseOptions?: QueryHookOptions<T.FindSceneMarkerTagsQuery, T.FindSceneMarkerTagsQueryVariables>
) {
  const sceneId = baseOptions?.variables?.id;
  return useRestQuery<T.FindSceneMarkerTagsQuery>(
    ["scene-markers", "tags", sceneId],
    async () => {
      const tags = await fetchOne(`/scene-markers/tags/${sceneId}`);
      return { sceneMarkerTags: tags } as T.FindSceneMarkerTagsQuery;
    },
    { enabled: !!sceneId && !baseOptions?.skip }
  );
}

// ============================================================
// COLOR PRESET & SAVED FILTER QUERY HOOKS
// ============================================================

export function useFindColorPresetsQuery(
  baseOptions?: QueryHookOptions<T.FindColorPresetsQuery, T.FindColorPresetsQueryVariables>
) {
  return useRestQuery<T.FindColorPresetsQuery>(
    queryKeys.colorPresets.all,
    async () => {
      const presets = await fetchOne("/color-presets");
      return { findColorPresets: presets } as T.FindColorPresetsQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindSavedFilterQuery(
  baseOptions?: QueryHookOptions<T.FindSavedFilterQuery, T.FindSavedFilterQueryVariables>
) {
  const id = baseOptions?.variables?.id;
  return useRestQuery<T.FindSavedFilterQuery>(
    queryKeys.filters.detail(id ?? ""),
    async () => {
      const filter = await fetchOne(`/filters/${id}`);
      return { findSavedFilter: filter } as T.FindSavedFilterQuery;
    },
    { enabled: !!id && !baseOptions?.skip }
  );
}

export function useFindSavedFiltersQuery(
  baseOptions?: QueryHookOptions<T.FindSavedFiltersQuery, T.FindSavedFiltersQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.FindSavedFiltersQuery>(
    queryKeys.filters.list(vars?.mode?.toString()),
    async () => {
      const params = vars?.mode ? `?mode=${vars.mode}` : "";
      const filters = await fetchOne(`/filters${params}`);
      return { findSavedFilters: filters } as T.FindSavedFiltersQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useFindDefaultFilterQuery(
  baseOptions?: QueryHookOptions<FindDefaultFilterQuery, FindDefaultFilterQueryVariables>
) {
  const mode = baseOptions?.variables?.mode;
  return useRestQuery<FindDefaultFilterQuery>(
    queryKeys.filters.default(mode?.toString() ?? ""),
    async () => {
      const filter = await fetchOne(`/filters/default?mode=${mode}`);
      return { findDefaultFilter: filter } as FindDefaultFilterQuery;
    },
    { enabled: !!mode && !baseOptions?.skip }
  );
}

// ============================================================
// CONFIGURATION & SYSTEM QUERY HOOKS
// ============================================================

export function useConfigurationQuery(
  baseOptions?: QueryHookOptions<T.ConfigurationQuery, T.ConfigurationQueryVariables>
) {
  return useRestQuery<T.ConfigurationQuery>(
    queryKeys.config,
    async () => {
      const config = await fetchOne("/config");
      return { configuration: config } as T.ConfigurationQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useSystemStatusQuery(
  baseOptions?: QueryHookOptions<T.SystemStatusQuery, T.SystemStatusQueryVariables>
) {
  return useRestQuery<T.SystemStatusQuery>(
    queryKeys.systemStatus,
    async () => {
      const status = await fetchOne("/system/status");
      return { systemStatus: status } as T.SystemStatusQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useVersionQuery(
  baseOptions?: QueryHookOptions<T.VersionQuery, T.VersionQueryVariables>
) {
  return useRestQuery<T.VersionQuery>(
    queryKeys.version,
    async () => {
      const version = await fetchOne("/system/version");
      return { version: version } as T.VersionQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useLatestVersionQuery(
  baseOptions?: QueryHookOptions<T.LatestVersionQuery, T.LatestVersionQueryVariables>
) {
  return useRestQuery<T.LatestVersionQuery>(
    ["system", "latest-version"],
    async () => {
      const version = await fetchOne("/system/latest-version");
      return { latestversion: version } as T.LatestVersionQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useStatsQuery(
  baseOptions?: QueryHookOptions<T.StatsQuery, T.StatsQueryVariables>
) {
  return useRestQuery<T.StatsQuery>(
    queryKeys.stats,
    async () => {
      const stats = await fetchOne("/stats");
      return { stats: stats } as T.StatsQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useOCountStatsQuery(
  baseOptions?: QueryHookOptions<T.OCountStatsQuery, T.OCountStatsQueryVariables>
) {
  return useRestQuery<T.OCountStatsQuery>(
    ["stats", "o-count"],
    async () => {
      const stats = await fetchOne("/stats/o-count");
      return { oCountStats: stats } as T.OCountStatsQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useDirectoryQuery(
  baseOptions?: QueryHookOptions<T.DirectoryQuery, T.DirectoryQueryVariables>
) {
  const path = baseOptions?.variables?.path;
  return useRestQuery<T.DirectoryQuery>(
    ["system", "directory", path],
    async () => {
      const params = path ? `?path=${encodeURIComponent(path)}` : "";
      const dir = await fetchOne(`/system/directory${params}`);
      return { directory: dir } as T.DirectoryQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

// ============================================================
// DLNA QUERY HOOKS
// ============================================================

export function useDlnaStatusQuery(
  baseOptions?: QueryHookOptions<T.DlnaStatusQuery, T.DlnaStatusQueryVariables>
) {
  return useRestQuery<T.DlnaStatusQuery>(
    queryKeys.dlna.status,
    async () => {
      const status = await fetchOne("/dlna/status");
      return { dlnaStatus: status } as T.DlnaStatusQuery;
    },
    {
      enabled: !baseOptions?.skip,
      cacheTime: 0,
      staleTime: 0,
    }
  );
}

// ============================================================
// JOB QUERY HOOKS
// ============================================================

export function useJobQueueQuery(
  baseOptions?: QueryHookOptions<T.JobQueueQuery, T.JobQueueQueryVariables>
) {
  return useRestQuery<T.JobQueueQuery>(
    queryKeys.jobs.all,
    async () => {
      const jobs = await fetchOne("/jobs");
      return { jobQueue: jobs } as T.JobQueueQuery;
    },
    {
      enabled: !baseOptions?.skip,
      cacheTime: 0,
      staleTime: 0,
    }
  );
}

export function useFindJobQuery(
  baseOptions?: QueryHookOptions<T.FindJobQuery, T.FindJobQueryVariables>
) {
  const id = baseOptions?.variables?.input?.id;
  const [pollInterval, setPollInterval] = useState<number | null>(null);

  const result = useQuery<T.FindJobQuery, Error>(
    ["jobs", "find", id],
    async () => {
      const job = await fetchOne(`/jobs/${id}`);
      return { findJob: job } as T.FindJobQuery;
    },
    {
      enabled: !!id && !baseOptions?.skip,
      cacheTime: 0,
      staleTime: 0,
      refetchInterval: pollInterval ?? undefined,
    }
  );

  const startPolling = useCallback((interval: number) => {
    setPollInterval(interval);
  }, []);

  const stopPolling = useCallback(() => {
    setPollInterval(null);
  }, []);

  return {
    data: result.data,
    loading: result.isLoading,
    error: result.error ?? undefined,
    refetch: result.refetch as (...args: unknown[]) => Promise<unknown>,
    networkStatus: result.isLoading
      ? NetworkStatus.loading
      : result.isSuccess
        ? NetworkStatus.ready
        : NetworkStatus.error,
    called: true,
    startPolling,
    stopPolling,
  };
}

// ============================================================
// LOG QUERY HOOKS
// ============================================================

export function useLogsQuery(
  baseOptions?: QueryHookOptions<T.LogsQuery, T.LogsQueryVariables>
) {
  return useRestQuery<T.LogsQuery>(
    ["logs"],
    async () => {
      const logs = await fetchOne("/logs");
      return { logs: logs } as T.LogsQuery;
    },
    {
      enabled: !baseOptions?.skip,
      cacheTime: 0,
      staleTime: 0,
    }
  );
}

// ============================================================
// PLUGIN QUERY HOOKS
// ============================================================

export function usePluginsQuery(
  baseOptions?: QueryHookOptions<T.PluginsQuery, T.PluginsQueryVariables>
) {
  return useRestQuery<T.PluginsQuery>(
    queryKeys.plugins.all,
    async () => {
      const plugins = await fetchOne("/plugins");
      return { plugins: plugins } as T.PluginsQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function usePluginTasksQuery(
  baseOptions?: QueryHookOptions<T.PluginTasksQuery, T.PluginTasksQueryVariables>
) {
  return useRestQuery<T.PluginTasksQuery>(
    queryKeys.plugins.tasks,
    async () => {
      const tasks = await fetchOne("/plugins/tasks");
      return { pluginTasks: tasks } as T.PluginTasksQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

// ============================================================
// SCRAPER QUERY HOOKS
// ============================================================

export function useListSceneScrapersQuery(
  baseOptions?: QueryHookOptions<T.ListSceneScrapersQuery, T.ListSceneScrapersQueryVariables>
) {
  return useRestQuery<T.ListSceneScrapersQuery>(
    queryKeys.scrapers.list(["SCENE"]),
    async () => {
      const scrapers = await fetchOne("/scrapers?type=SCENE");
      return { listSceneScrapers: scrapers } as unknown as T.ListSceneScrapersQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useListPerformerScrapersQuery(
  baseOptions?: QueryHookOptions<T.ListPerformerScrapersQuery, T.ListPerformerScrapersQueryVariables>
) {
  return useRestQuery<T.ListPerformerScrapersQuery>(
    queryKeys.scrapers.list(["PERFORMER"]),
    async () => {
      const scrapers = await fetchOne("/scrapers?type=PERFORMER");
      return { listPerformerScrapers: scrapers } as unknown as T.ListPerformerScrapersQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useListGroupScrapersQuery(
  baseOptions?: QueryHookOptions<T.ListGroupScrapersQuery, T.ListGroupScrapersQueryVariables>
) {
  return useRestQuery<T.ListGroupScrapersQuery>(
    queryKeys.scrapers.list(["GROUP"]),
    async () => {
      const scrapers = await fetchOne("/scrapers?type=GROUP");
      return { listGroupScrapers: scrapers } as unknown as T.ListGroupScrapersQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useListGalleryScrapersQuery(
  baseOptions?: QueryHookOptions<T.ListGalleryScrapersQuery, T.ListGalleryScrapersQueryVariables>
) {
  return useRestQuery<T.ListGalleryScrapersQuery>(
    queryKeys.scrapers.list(["GALLERY"]),
    async () => {
      const scrapers = await fetchOne("/scrapers?type=GALLERY");
      return { listGalleryScrapers: scrapers } as unknown as T.ListGalleryScrapersQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useListImageScrapersQuery(
  baseOptions?: QueryHookOptions<T.ListImageScrapersQuery, T.ListImageScrapersQueryVariables>
) {
  return useRestQuery<T.ListImageScrapersQuery>(
    queryKeys.scrapers.list(["IMAGE"]),
    async () => {
      const scrapers = await fetchOne("/scrapers?type=IMAGE");
      return { listImageScrapers: scrapers } as unknown as T.ListImageScrapersQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useScrapeSinglePerformerQuery(
  baseOptions?: QueryHookOptions<T.ScrapeSinglePerformerQuery, T.ScrapeSinglePerformerQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<T.ScrapeSinglePerformerQuery>(
    ["scrapers", "performer", vars],
    async () => {
      const result = await postAndUnwrap("/scrapers/performer", {
        source: vars?.source,
        input: vars?.input,
      });
      return { scrapeSinglePerformer: result } as T.ScrapeSinglePerformerQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useValidateStashBoxLazyQuery(
  baseOptions?: LazyQueryHookOptions<T.ValidateStashBoxQuery, T.ValidateStashBoxQueryVariables>
) {
  return useRestLazyQuery<T.ValidateStashBoxQuery, T.ValidateStashBoxQueryVariables>(
    (vars) => ["system", "validate-stashbox", vars],
    (vars) => async () => {
      const result = await postAndUnwrap("/system/validate-stashbox", {
        input: vars?.input,
      });
      return { validateStashBoxCredentials: result } as T.ValidateStashBoxQuery;
    }
  );
}

// ============================================================
// PACKAGE QUERY HOOKS
// ============================================================

export function useInstalledScraperPackagesQuery(
  baseOptions?: QueryHookOptions<T.InstalledScraperPackagesQuery, T.InstalledScraperPackagesQueryVariables>
) {
  return useRestQuery<T.InstalledScraperPackagesQuery>(
    queryKeys.packages.installed("scraper"),
    async () => {
      const packages = await fetchOne("/packages/installed?type=scraper");
      return { installedPackages: packages } as T.InstalledScraperPackagesQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useInstalledScraperPackagesStatusQuery(
  baseOptions?: QueryHookOptions<T.InstalledScraperPackagesStatusQuery, T.InstalledScraperPackagesStatusQueryVariables>
) {
  return useRestQuery<T.InstalledScraperPackagesStatusQuery>(
    [...queryKeys.packages.installed("scraper"), "status"],
    async () => {
      const packages = await fetchOne("/packages/installed?type=scraper&upgrades=true");
      return { installedPackages: packages } as T.InstalledScraperPackagesStatusQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useInstalledPluginPackagesQuery(
  baseOptions?: QueryHookOptions<T.InstalledPluginPackagesQuery, T.InstalledPluginPackagesQueryVariables>
) {
  return useRestQuery<T.InstalledPluginPackagesQuery>(
    queryKeys.packages.installed("plugin"),
    async () => {
      const packages = await fetchOne("/packages/installed?type=plugin");
      return { installedPackages: packages } as T.InstalledPluginPackagesQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

export function useInstalledPluginPackagesStatusQuery(
  baseOptions?: QueryHookOptions<T.InstalledPluginPackagesStatusQuery, T.InstalledPluginPackagesStatusQueryVariables>
) {
  return useRestQuery<T.InstalledPluginPackagesStatusQuery>(
    [...queryKeys.packages.installed("plugin"), "status"],
    async () => {
      const packages = await fetchOne("/packages/installed?type=plugin&upgrades=true");
      return { installedPackages: packages } as T.InstalledPluginPackagesStatusQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

// ============================================================
// VIEW HISTORY QUERY HOOKS
// ============================================================

export function useFindViewHistoryQuery(
  baseOptions?: QueryHookOptions<FindViewHistoryQuery, FindViewHistoryQueryVariables>
) {
  const vars = baseOptions?.variables;
  return useRestQuery<FindViewHistoryQuery>(
    ["view-history", vars],
    async () => {
      const result = await postAndUnwrap("/view-history/query", vars);
      return { findViewHistory: result } as FindViewHistoryQuery;
    },
    { enabled: !baseOptions?.skip }
  );
}

// ============================================================
// SCENE MUTATION HOOKS
// ============================================================

export function useSceneCreateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneCreateMutation, T.SceneCreateMutationVariables>(
    (vars) => postAndUnwrap("/scenes", vars.input),
    { invalidateKeys: [queryKeys.scenes.all, queryKeys.stats] }
  );
}

export function useSceneUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneUpdateMutation, T.SceneUpdateMutationVariables>(
    (vars) => putAndUnwrap(`/scenes/${vars.input.id}`, vars.input),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useBulkSceneUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.BulkSceneUpdateMutation, T.BulkSceneUpdateMutationVariables>(
    (vars) => putAndUnwrap("/scenes/bulk", vars.input),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useScenesUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ScenesUpdateMutation, T.ScenesUpdateMutationVariables>(
    (vars) => putAndUnwrap("/scenes/batch", vars.input),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneDestroyMutation, T.SceneDestroyMutationVariables>(
    (vars) => apiDelete(`/scenes/${vars.id}`, vars),
    { invalidateKeys: [queryKeys.scenes.all, queryKeys.stats] }
  );
}

export function useScenesDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ScenesDestroyMutation, T.ScenesDestroyMutationVariables>(
    (vars) => apiDelete("/scenes", vars),
    { invalidateKeys: [queryKeys.scenes.all, queryKeys.stats] }
  );
}

export function useSceneAddOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneAddOMutation, T.SceneAddOMutationVariables>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/o`),
    { invalidateKeys: [queryKeys.scenes.all, queryKeys.stats] }
  );
}

export function useSceneDeleteOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneDeleteOMutation, T.SceneDeleteOMutationVariables>(
    (vars) => deleteAndUnwrap(`/scenes/${vars.id}/o`, vars),
    { invalidateKeys: [queryKeys.scenes.all, queryKeys.stats] }
  );
}

export function useSceneResetOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneResetOMutation, T.SceneResetOMutationVariables>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/o/reset`),
    { invalidateKeys: [queryKeys.scenes.all, queryKeys.stats] }
  );
}

export function useSceneResetActivityMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneResetActivityMutation, T.SceneResetActivityMutationVariables>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/activity/reset`),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneIncrementOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneIncrementOmgMutation, T.SceneIncrementOmgMutationVariables>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/omg`),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneDecrementOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneDecrementOmgMutation, T.SceneDecrementOmgMutationVariables>(
    (vars) => deleteAndUnwrap(`/scenes/${vars.id}/omg`),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneResetOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneResetOmgMutation, T.SceneResetOmgMutationVariables>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/omg/reset`),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneAddOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneAddOmgMutation, T.SceneAddOmgMutationVariables>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/omg`),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneDeleteOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneDeleteOmgMutation, T.SceneDeleteOmgMutationVariables>(
    (vars) => deleteAndUnwrap(`/scenes/${vars.id}/omg`),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneGenerateScreenshotMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneGenerateScreenshotMutation, T.SceneGenerateScreenshotMutationVariables>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/screenshot`, vars),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneSaveActivityMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneSaveActivityMutation, T.SceneSaveActivityMutationVariables>(
    (vars) => putAndUnwrap(`/scenes/${vars.id}/activity`, vars),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneAddPlayMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneAddPlayMutation, T.SceneAddPlayMutationVariables>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/play`),
    { invalidateKeys: [queryKeys.scenes.all, queryKeys.stats] }
  );
}

export function useSceneDeletePlayMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneDeletePlayMutation, T.SceneDeletePlayMutationVariables>(
    (vars) => deleteAndUnwrap(`/scenes/${vars.id}/play`, vars),
    { invalidateKeys: [queryKeys.scenes.all, queryKeys.stats] }
  );
}

export function useSceneResetPlayCountMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneResetPlayCountMutation, T.SceneResetPlayCountMutationVariables>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/play/reset`),
    { invalidateKeys: [queryKeys.scenes.all, queryKeys.stats] }
  );
}

export function useScanVideoFileThreatsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ScanVideoFileThreatsMutation, T.ScanVideoFileThreatsMutationVariables>(
    (vars) => postAndUnwrap(`/files/${vars.fileId}/scan-threats`, vars),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useScanAllScenesForThreatsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ScanAllScenesForThreatsMutation, T.ScanAllScenesForThreatsMutationVariables>(
    () => postRaw("/files/scan-all-threats"),
  );
}

export function useRecalculateSceneSimilaritiesMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.RecalculateSceneSimilaritiesMutation, T.RecalculateSceneSimilaritiesMutationVariables>(
    (vars) => postAndUnwrap(`/scenes/${vars.scene_id}/recalculate-similarity`),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useOpenInExternalPlayerMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.OpenInExternalPlayerMutation, T.OpenInExternalPlayerMutationVariables>(
    (vars) => postRaw(`/misc/open-external-player/${vars.id}`),
  );
}

// Scene file operations
export function useSceneSetPrimaryFileMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { id: string; file_id: string }>(
    (vars) => putAndUnwrap(`/scenes/${vars.id}/primary-file`, { file_id: vars.file_id }),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneAssignFileMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, T.AssignSceneFileInput>(
    (vars) => putAndUnwrap(`/scenes/${vars.scene_id}/assign-file`, vars),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneMergeMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, T.SceneMergeInput>(
    (vars) => postAndUnwrap(`/scenes/${vars.destination}/merge`, vars),
    { invalidateKeys: [queryKeys.scenes.all, queryKeys.stats] }
  );
}

export function useSceneConvertMp4Mutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { id: string }>(
    (vars) => postRaw(`/scenes/${vars.id}/convert/mp4`),
  );
}

export function useSceneConvertHlsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { id: string }>(
    (vars) => postRaw(`/scenes/${vars.id}/convert/hls`),
  );
}

export function useSceneReduceResolutionMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { id: string; input: unknown }>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/reduce-resolution`, vars.input),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneTrimMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { id: string; input: unknown }>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/trim`, vars.input),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneRegenerateSpritesMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { id: string }>(
    (vars) => postRaw(`/scenes/${vars.id}/regenerate-sprites`),
  );
}

export function useSceneSetBrokenMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { id: string; broken: boolean }>(
    (vars) => putAndUnwrap(`/scenes/${vars.id}/broken`, { broken: vars.broken }),
    { invalidateKeys: [queryKeys.scenes.all] }
  );
}

export function useSceneSaveFilteredScreenshotMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { id: string; input: unknown }>(
    (vars) => postAndUnwrap(`/scenes/${vars.id}/filtered-screenshot`, vars.input),
  );
}

// ============================================================
// IMAGE MUTATION HOOKS
// ============================================================

export function useImageUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ImageUpdateMutation, T.ImageUpdateMutationVariables>(
    (vars) => putAndUnwrap(`/images/${vars.input.id}`, vars.input),
    { invalidateKeys: [queryKeys.images.all] }
  );
}

export function useBulkImageUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.BulkImageUpdateMutation, T.BulkImageUpdateMutationVariables>(
    (vars) => putAndUnwrap("/images/bulk", vars.input),
    { invalidateKeys: [queryKeys.images.all] }
  );
}

export function useImagesDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ImagesDestroyMutation, T.ImagesDestroyMutationVariables>(
    (vars) => apiDelete("/images", vars),
    { invalidateKeys: [queryKeys.images.all, queryKeys.stats] }
  );
}

export function useImageIncrementOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ImageIncrementOMutation, T.ImageIncrementOMutationVariables>(
    (vars) => postAndUnwrap(`/images/${vars.id}/o`),
    { invalidateKeys: [queryKeys.images.all, queryKeys.stats] }
  );
}

export function useImageDecrementOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ImageDecrementOMutation, T.ImageDecrementOMutationVariables>(
    (vars) => deleteAndUnwrap(`/images/${vars.id}/o`),
    { invalidateKeys: [queryKeys.images.all, queryKeys.stats] }
  );
}

export function useImageResetOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ImageResetOMutation, T.ImageResetOMutationVariables>(
    (vars) => postAndUnwrap(`/images/${vars.id}/o/reset`),
    { invalidateKeys: [queryKeys.images.all, queryKeys.stats] }
  );
}

export function useImageIncrementOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ImageIncrementOmgMutation, T.ImageIncrementOmgMutationVariables>(
    (vars) => postAndUnwrap(`/images/${vars.id}/omg`),
    { invalidateKeys: [queryKeys.images.all] }
  );
}

export function useImageDecrementOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ImageDecrementOmgMutation, T.ImageDecrementOmgMutationVariables>(
    (vars) => deleteAndUnwrap(`/images/${vars.id}/omg`),
    { invalidateKeys: [queryKeys.images.all] }
  );
}

export function useImageResetOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ImageResetOmgMutation, T.ImageResetOmgMutationVariables>(
    (vars) => postAndUnwrap(`/images/${vars.id}/omg/reset`),
    { invalidateKeys: [queryKeys.images.all] }
  );
}

// ============================================================
// GALLERY MUTATION HOOKS
// ============================================================

export function useGalleryCreateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryCreateMutation, T.GalleryCreateMutationVariables>(
    (vars) => postAndUnwrap("/galleries", vars.input),
    { invalidateKeys: [queryKeys.galleries.all, queryKeys.stats] }
  );
}

export function useGalleryUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryUpdateMutation, T.GalleryUpdateMutationVariables>(
    (vars) => putAndUnwrap(`/galleries/${vars.input.id}`, vars.input),
    { invalidateKeys: [queryKeys.galleries.all] }
  );
}

export function useBulkGalleryUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.BulkGalleryUpdateMutation, T.BulkGalleryUpdateMutationVariables>(
    (vars) => putAndUnwrap("/galleries/bulk", vars.input),
    { invalidateKeys: [queryKeys.galleries.all] }
  );
}

export function useGalleryDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryDestroyMutation, T.GalleryDestroyMutationVariables>(
    (vars) => apiDelete(`/galleries/${vars.ids?.[0]}`, vars),
    { invalidateKeys: [queryKeys.galleries.all, queryKeys.stats] }
  );
}

export function useGalleryAddOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryAddOMutation, T.GalleryAddOMutationVariables>(
    (vars) => postAndUnwrap(`/galleries/${vars.id}/o`),
    { invalidateKeys: [queryKeys.galleries.all, queryKeys.stats] }
  );
}

export function useGalleryDeleteOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryDeleteOMutation, T.GalleryDeleteOMutationVariables>(
    (vars) => deleteAndUnwrap(`/galleries/${vars.id}/o`, vars),
    { invalidateKeys: [queryKeys.galleries.all, queryKeys.stats] }
  );
}

export function useGalleryResetOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryResetOMutation, T.GalleryResetOMutationVariables>(
    (vars) => postAndUnwrap(`/galleries/${vars.id}/o/reset`),
    { invalidateKeys: [queryKeys.galleries.all, queryKeys.stats] }
  );
}

export function useGalleryIncrementOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryIncrementOmgMutation, T.GalleryIncrementOmgMutationVariables>(
    (vars) => postAndUnwrap(`/galleries/${vars.id}/omg`),
    { invalidateKeys: [queryKeys.galleries.all] }
  );
}

export function useGalleryDecrementOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryDecrementOmgMutation, T.GalleryDecrementOmgMutationVariables>(
    (vars) => deleteAndUnwrap(`/galleries/${vars.id}/omg`),
    { invalidateKeys: [queryKeys.galleries.all] }
  );
}

export function useGalleryResetOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryResetOmgMutation, T.GalleryResetOmgMutationVariables>(
    (vars) => postAndUnwrap(`/galleries/${vars.id}/omg/reset`),
    { invalidateKeys: [queryKeys.galleries.all] }
  );
}

export function useGalleryChapterCreateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryChapterCreateMutation, T.GalleryChapterCreateMutationVariables>(
    (vars) => postAndUnwrap(`/galleries/${vars.gallery_id}/chapters`, vars),
    { invalidateKeys: [queryKeys.galleries.all] }
  );
}

export function useGalleryChapterUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryChapterUpdateMutation, T.GalleryChapterUpdateMutationVariables>(
    (vars) => putAndUnwrap(`/galleries/${vars.gallery_id}/chapters/${vars.id}`, vars),
    { invalidateKeys: [queryKeys.galleries.all] }
  );
}

export function useGalleryChapterDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryChapterDestroyMutation, T.GalleryChapterDestroyMutationVariables>(
    (vars) => apiDelete(`/galleries/0/chapters/${vars.id}`),
    { invalidateKeys: [queryKeys.galleries.all] }
  );
}

export function useGalleryIncrementPlayMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryIncrementPlayMutation, T.GalleryIncrementPlayMutationVariables>(
    (vars) => postAndUnwrap(`/galleries/${vars.id}/play`),
    { invalidateKeys: [queryKeys.galleries.all, queryKeys.stats] }
  );
}

export function useGalleryDeletePlayMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryDeletePlayMutation, T.GalleryDeletePlayMutationVariables>(
    (vars) => deleteAndUnwrap(`/galleries/${vars.id}/play`, vars),
    { invalidateKeys: [queryKeys.galleries.all, queryKeys.stats] }
  );
}

export function useGalleryResetPlayCountMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GalleryResetPlayCountMutation, T.GalleryResetPlayCountMutationVariables>(
    (vars) => postAndUnwrap(`/galleries/${vars.id}/play/reset`),
    { invalidateKeys: [queryKeys.galleries.all, queryKeys.stats] }
  );
}

// ============================================================
// PERFORMER MUTATION HOOKS
// ============================================================

export function usePerformerCreateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.PerformerCreateMutation, T.PerformerCreateMutationVariables>(
    (vars) => postAndUnwrap("/performers", vars.input),
    { invalidateKeys: [queryKeys.performers.all, queryKeys.stats] }
  );
}

export function usePerformerUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.PerformerUpdateMutation, T.PerformerUpdateMutationVariables>(
    (vars) => putAndUnwrap(`/performers/${vars.input.id}`, vars.input),
    { invalidateKeys: [queryKeys.performers.all] }
  );
}

export function useBulkPerformerUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.BulkPerformerUpdateMutation, T.BulkPerformerUpdateMutationVariables>(
    (vars) => putAndUnwrap("/performers/bulk", vars.input),
    { invalidateKeys: [queryKeys.performers.all] }
  );
}

export function usePerformerDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.PerformerDestroyMutation, T.PerformerDestroyMutationVariables>(
    (vars) => apiDelete(`/performers/${vars.id}`),
    { invalidateKeys: [queryKeys.performers.all, queryKeys.stats] }
  );
}

export function usePerformersDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.PerformersDestroyMutation, T.PerformersDestroyMutationVariables>(
    (vars) => apiDelete("/performers", { ids: vars.ids }),
    { invalidateKeys: [queryKeys.performers.all, queryKeys.stats] }
  );
}

export function usePerformerProfileImageCreateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { performer_id: string; input: unknown }>(
    (vars) => postAndUnwrap(`/performers/${vars.performer_id}/profile-images`, vars.input),
    { invalidateKeys: [queryKeys.performers.all] }
  );
}

export function usePerformerProfileImageUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { performer_id: string; image_id: string; input: unknown }>(
    (vars) => putAndUnwrap(`/performers/${vars.performer_id}/profile-images/${vars.image_id}`, vars.input),
    { invalidateKeys: [queryKeys.performers.all] }
  );
}

export function usePerformerProfileImageDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { performer_id: string; image_id: string }>(
    (vars) => apiDelete(`/performers/${vars.performer_id}/profile-images/${vars.image_id}`),
    { invalidateKeys: [queryKeys.performers.all] }
  );
}

// ============================================================
// STUDIO MUTATION HOOKS
// ============================================================

export function useStudioCreateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.StudioCreateMutation, T.StudioCreateMutationVariables>(
    (vars) => postAndUnwrap("/studios", vars.input),
    { invalidateKeys: [queryKeys.studios.all, queryKeys.stats] }
  );
}

export function useStudioUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.StudioUpdateMutation, T.StudioUpdateMutationVariables>(
    (vars) => putAndUnwrap(`/studios/${vars.input.id}`, vars.input),
    { invalidateKeys: [queryKeys.studios.all] }
  );
}

export function useStudioDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.StudioDestroyMutation, T.StudioDestroyMutationVariables>(
    (vars) => apiDelete(`/studios/${vars.id}`),
    { invalidateKeys: [queryKeys.studios.all, queryKeys.stats] }
  );
}

export function useStudiosDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.StudiosDestroyMutation, T.StudiosDestroyMutationVariables>(
    (vars) => apiDelete("/studios", { ids: vars.ids }),
    { invalidateKeys: [queryKeys.studios.all, queryKeys.stats] }
  );
}

// ============================================================
// TAG MUTATION HOOKS
// ============================================================

export function useTagCreateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.TagCreateMutation, T.TagCreateMutationVariables>(
    (vars) => postAndUnwrap("/tags", vars.input),
    { invalidateKeys: [queryKeys.tags.all, queryKeys.stats] }
  );
}

export function useTagUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.TagUpdateMutation, T.TagUpdateMutationVariables>(
    (vars) => putAndUnwrap(`/tags/${vars.input.id}`, vars.input),
    { invalidateKeys: [queryKeys.tags.all] }
  );
}

export function useBulkTagUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.BulkTagUpdateMutation, T.BulkTagUpdateMutationVariables>(
    (vars) => putAndUnwrap("/tags/bulk", vars.input),
    { invalidateKeys: [queryKeys.tags.all] }
  );
}

export function useTagDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.TagDestroyMutation, T.TagDestroyMutationVariables>(
    (vars) => apiDelete(`/tags/${vars.id}`),
    { invalidateKeys: [queryKeys.tags.all, queryKeys.stats] }
  );
}

export function useTagsDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.TagsDestroyMutation, T.TagsDestroyMutationVariables>(
    (vars) => apiDelete("/tags", { ids: vars.ids }),
    { invalidateKeys: [queryKeys.tags.all, queryKeys.stats] }
  );
}

export function useTagsMergeMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.TagsMergeMutation, T.TagsMergeMutationVariables>(
    (vars) => postAndUnwrap("/tags/merge", vars),
    { invalidateKeys: [queryKeys.tags.all] }
  );
}

// ============================================================
// GROUP MUTATION HOOKS
// ============================================================

export function useGroupCreateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GroupCreateMutation, T.GroupCreateMutationVariables>(
    (vars) => postAndUnwrap("/groups", vars.input),
    { invalidateKeys: [queryKeys.groups.all, queryKeys.stats] }
  );
}

export function useGroupUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GroupUpdateMutation, T.GroupUpdateMutationVariables>(
    (vars) => putAndUnwrap(`/groups/${vars.input.id}`, vars.input),
    { invalidateKeys: [queryKeys.groups.all] }
  );
}

export function useBulkGroupUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.BulkGroupUpdateMutation, T.BulkGroupUpdateMutationVariables>(
    (vars) => putAndUnwrap("/groups/bulk", vars.input),
    { invalidateKeys: [queryKeys.groups.all] }
  );
}

export function useGroupDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GroupDestroyMutation, T.GroupDestroyMutationVariables>(
    (vars) => apiDelete(`/groups/${vars.id}`),
    { invalidateKeys: [queryKeys.groups.all, queryKeys.stats] }
  );
}

export function useGroupsDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GroupsDestroyMutation, T.GroupsDestroyMutationVariables>(
    (vars) => apiDelete("/groups", { ids: vars.ids }),
    { invalidateKeys: [queryKeys.groups.all, queryKeys.stats] }
  );
}

export function useReorderSubGroupsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ReorderSubGroupsMutation, T.ReorderSubGroupsMutationVariables>(
    (vars) => putAndUnwrap(`/groups/${vars.input.group_id}/sub-groups/reorder`, vars.input),
    { invalidateKeys: [queryKeys.groups.all] }
  );
}

export function useAddGroupSubGroupsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.AddGroupSubGroupsMutation, T.AddGroupSubGroupsMutationVariables>(
    (vars) => postAndUnwrap(`/groups/${vars.input.containing_group_id}/sub-groups`, vars.input),
    { invalidateKeys: [queryKeys.groups.all] }
  );
}

export function useRemoveGroupSubGroupsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.RemoveGroupSubGroupsMutation, T.RemoveGroupSubGroupsMutationVariables>(
    (vars) => apiDelete(`/groups/${vars.input.containing_group_id}/sub-groups`, vars.input),
    { invalidateKeys: [queryKeys.groups.all] }
  );
}

// ============================================================
// GAME MUTATION HOOKS
// ============================================================

export function useGameCreateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameCreateMutation, T.GameCreateMutationVariables>(
    (vars) => postAndUnwrap("/games", vars.input),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameUpdateMutation, T.GameUpdateMutationVariables>(
    (vars) => putAndUnwrap(`/games/${vars.input.id}`, vars.input),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameDestroyMutation, T.GameDestroyMutationVariables>(
    (vars) => apiDelete(`/games/${vars.input.ids?.[0]}`, vars.input),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameIncrementOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameIncrementOMutation, T.GameIncrementOMutationVariables>(
    (vars) => postAndUnwrap(`/games/${vars.id}/o`),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameDecrementOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameDecrementOMutation, T.GameDecrementOMutationVariables>(
    (vars) => deleteAndUnwrap(`/games/${vars.id}/o`),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameResetOMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameResetOMutation, T.GameResetOMutationVariables>(
    (vars) => postAndUnwrap(`/games/${vars.id}/o/reset`),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameAddOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameAddOmgMutation, T.GameAddOmgMutationVariables>(
    (vars) => postAndUnwrap(`/games/${vars.id}/omg`),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameDeleteOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameDeleteOmgMutation, T.GameDeleteOmgMutationVariables>(
    (vars) => deleteAndUnwrap(`/games/${vars.id}/omg`),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameResetOmgMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameResetOmgMutation, T.GameResetOmgMutationVariables>(
    (vars) => postAndUnwrap(`/games/${vars.id}/omg/reset`),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameAddViewMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameAddViewMutation, T.GameAddViewMutationVariables>(
    (vars) => postAndUnwrap(`/games/${vars.id}/view`),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameDeleteViewMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameDeleteViewMutation, T.GameDeleteViewMutationVariables>(
    (vars) => deleteAndUnwrap(`/games/${vars.id}/view`),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameResetViewsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameResetViewsMutation, T.GameResetViewsMutationVariables>(
    (vars) => postAndUnwrap(`/games/${vars.id}/view/reset`),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

export function useGameIncrementViewMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GameIncrementViewMutation, T.GameIncrementViewMutationVariables>(
    (vars) => postAndUnwrap(`/games/${vars.id}/view`),
    { invalidateKeys: [queryKeys.games.all] }
  );
}

// ============================================================
// SCENE MARKER MUTATION HOOKS
// ============================================================

export function useSceneMarkerCreateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneMarkerCreateMutation, T.SceneMarkerCreateMutationVariables>(
    (vars) => postAndUnwrap("/scene-markers", vars),
    { invalidateKeys: [queryKeys.sceneMarkers.all, queryKeys.scenes.all] }
  );
}

export function useSceneMarkerUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneMarkerUpdateMutation, T.SceneMarkerUpdateMutationVariables>(
    (vars) => putAndUnwrap(`/scene-markers/${vars.id}`, vars),
    { invalidateKeys: [queryKeys.sceneMarkers.all, queryKeys.scenes.all] }
  );
}

export function useSceneMarkerDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneMarkerDestroyMutation, T.SceneMarkerDestroyMutationVariables>(
    (vars) => apiDelete(`/scene-markers/${vars.id}`),
    { invalidateKeys: [queryKeys.sceneMarkers.all, queryKeys.scenes.all] }
  );
}

export function useSceneMarkersDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SceneMarkersDestroyMutation, T.SceneMarkersDestroyMutationVariables>(
    (vars) => apiDelete("/scene-markers", { ids: vars.ids }),
    { invalidateKeys: [queryKeys.sceneMarkers.all, queryKeys.scenes.all] }
  );
}

// ============================================================
// COLOR PRESET MUTATION HOOKS
// ============================================================

export function useColorPresetCreateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ColorPresetCreateMutation, T.ColorPresetCreateMutationVariables>(
    (vars) => postAndUnwrap("/color-presets", vars.input),
    { invalidateKeys: [queryKeys.colorPresets.all] }
  );
}

export function useColorPresetUpdateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ColorPresetUpdateMutation, T.ColorPresetUpdateMutationVariables>(
    (vars) => putAndUnwrap(`/color-presets/${vars.input.id}`, vars.input),
    { invalidateKeys: [queryKeys.colorPresets.all] }
  );
}

export function useColorPresetDestroyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ColorPresetDestroyMutation, T.ColorPresetDestroyMutationVariables>(
    (vars) => apiDelete(`/color-presets/${vars.input.id}`),
    { invalidateKeys: [queryKeys.colorPresets.all] }
  );
}

// ============================================================
// SAVED FILTER MUTATION HOOKS
// ============================================================

export function useSaveFilterMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SaveFilterMutation, T.SaveFilterMutationVariables>(
    (vars) => postAndUnwrap("/filters", vars.input),
    { invalidateKeys: [queryKeys.filters.all] }
  );
}

export function useSetDefaultFilterMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<SetDefaultFilterMutation, SetDefaultFilterMutationVariables>(
    (vars) => postAndUnwrap("/filters/default", vars.input),
    { invalidateKeys: [queryKeys.filters.all] }
  );
}

export function useDestroySavedFilterMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.DestroySavedFilterMutation, T.DestroySavedFilterMutationVariables>(
    (vars) => apiDelete(`/filters/${vars.input.id}`),
    { invalidateKeys: [queryKeys.filters.all] }
  );
}

// ============================================================
// CONFIG MUTATION HOOKS
// ============================================================

export function useConfigureGeneralMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ConfigureGeneralMutation, T.ConfigureGeneralMutationVariables>(
    (vars) => putAndUnwrap("/config/general", vars.input),
    { invalidateKeys: [queryKeys.config, queryKeys.scrapers.all, queryKeys.plugins.all] }
  );
}

export function useConfigureInterfaceMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ConfigureInterfaceMutation, T.ConfigureInterfaceMutationVariables>(
    (vars) => putAndUnwrap("/config/interface", vars.input),
    { invalidateKeys: [queryKeys.config] }
  );
}

export function useGenerateApiKeyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.GenerateApiKeyMutation, T.GenerateApiKeyMutationVariables>(
    (vars) => postAndUnwrap("/config/api-key", vars.input),
    { invalidateKeys: [queryKeys.config] }
  );
}

export function useConfigureDefaultsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ConfigureDefaultsMutation, T.ConfigureDefaultsMutationVariables>(
    (vars) => putAndUnwrap("/config/defaults", vars.input),
    { invalidateKeys: [queryKeys.config] }
  );
}

export function useConfigureUiMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ConfigureUiMutation, T.ConfigureUiMutationVariables>(
    (vars) => putAndUnwrap("/config/ui", vars.input),
    { invalidateKeys: [queryKeys.config] }
  );
}

export function useConfigureUiSettingMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ConfigureUiSettingMutation, T.ConfigureUiSettingMutationVariables>(
    (vars) => putAndUnwrap("/config/ui/setting", vars),
    { invalidateKeys: [queryKeys.config] }
  );
}

export function useConfigureScrapingMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ConfigureScrapingMutation, T.ConfigureScrapingMutationVariables>(
    (vars) => putAndUnwrap("/config/scraping", vars.input),
    { invalidateKeys: [queryKeys.config] }
  );
}

export function useConfigureDlnaMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ConfigureDlnaMutation, T.ConfigureDlnaMutationVariables>(
    (vars) => putAndUnwrap("/config/dlna", vars.input),
    { invalidateKeys: [queryKeys.config] }
  );
}

export function useConfigurePluginMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ConfigurePluginMutation, T.ConfigurePluginMutationVariables>(
    (vars) => putAndUnwrap(`/config/plugin/${vars.plugin_id}`, vars.input),
    { invalidateKeys: [queryKeys.config] }
  );
}

// ============================================================
// DLNA MUTATION HOOKS
// ============================================================

export function useEnableDlnaMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.EnableDlnaMutation, T.EnableDlnaMutationVariables>(
    (vars) => postRaw("/dlna/enable", vars?.input),
    { invalidateKeys: [queryKeys.dlna.status] }
  );
}

export function useDisableDlnaMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.DisableDlnaMutation, T.DisableDlnaMutationVariables>(
    (vars) => postRaw("/dlna/disable", vars?.input),
    { invalidateKeys: [queryKeys.dlna.status] }
  );
}

export function useAddTempDlnaipMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.AddTempDlnaipMutation, T.AddTempDlnaipMutationVariables>(
    (vars) => postAndUnwrap("/dlna/temp-ip", vars.input),
    { invalidateKeys: [queryKeys.dlna.status] }
  );
}

export function useRemoveTempDlnaipMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.RemoveTempDlnaipMutation, T.RemoveTempDlnaipMutationVariables>(
    (vars) => apiDelete("/dlna/temp-ip", vars.input),
    { invalidateKeys: [queryKeys.dlna.status] }
  );
}

// ============================================================
// SYSTEM / SETUP MUTATION HOOKS
// ============================================================

export function useSetupMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SetupMutation, T.SetupMutationVariables>(
    (vars) => postAndUnwrap("/system/setup", vars.input),
    { invalidateKeys: [queryKeys.config, queryKeys.systemStatus] }
  );
}

export function useMigrateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MigrateMutation, T.MigrateMutationVariables>(
    (vars) => postRaw("/system/migrate", vars.input),
  );
}

export function useDownloadFfMpegMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.DownloadFfMpegMutation, T.DownloadFfMpegMutationVariables>(
    () => postRaw("/system/download-ffmpeg"),
    { invalidateKeys: [queryKeys.systemStatus] }
  );
}

// ============================================================
// METADATA / TASK MUTATION HOOKS
// ============================================================

export function useMetadataScanMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MetadataScanMutation, T.MetadataScanMutationVariables>(
    (vars) => postRaw("/metadata/scan", vars.input),
  );
}

export function useMetadataIdentifyMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MetadataIdentifyMutation, T.MetadataIdentifyMutationVariables>(
    (vars) => postRaw("/metadata/identify", vars.input),
  );
}

export function useMetadataAutoTagMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MetadataAutoTagMutation, T.MetadataAutoTagMutationVariables>(
    (vars) => postRaw("/metadata/auto-tag", vars.input),
  );
}

export function useMetadataGenerateMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MetadataGenerateMutation, T.MetadataGenerateMutationVariables>(
    (vars) => postRaw("/metadata/generate", vars.input),
  );
}

export function useMetadataCleanMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MetadataCleanMutation, T.MetadataCleanMutationVariables>(
    (vars) => postRaw("/metadata/clean", vars.input),
  );
}

export function useMetadataCleanGeneratedMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MetadataCleanGeneratedMutation, T.MetadataCleanGeneratedMutationVariables>(
    (vars) => postRaw("/metadata/clean-generated", vars.input),
  );
}

export function useMetadataExportMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MetadataExportMutation, T.MetadataExportMutationVariables>(
    () => postRaw("/metadata/export"),
  );
}

export function useMetadataImportMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MetadataImportMutation, T.MetadataImportMutationVariables>(
    () => postRaw("/metadata/import"),
  );
}

export function useExportObjectsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ExportObjectsMutation, T.ExportObjectsMutationVariables>(
    (vars) => postRaw("/metadata/export-objects", vars.input),
  );
}

export function useImportObjectsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ImportObjectsMutation, T.ImportObjectsMutationVariables>(
    (vars) => postRaw("/metadata/import-objects", vars.input),
  );
}

export function useRunPluginTaskMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.RunPluginTaskMutation, T.RunPluginTaskMutationVariables>(
    (vars) => postRaw(`/plugins/${vars.plugin_id}/run`, {
      task_name: vars.task_name,
      args_map: vars.args_map,
    }),
  );
}

export function useRunPluginOperationMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { plugin_id: string; args?: unknown }>(
    (vars) => postRaw(`/plugins/${vars.plugin_id}/operation`, vars.args),
  );
}

// ============================================================
// SCRAPER / PLUGIN MANAGEMENT MUTATION HOOKS
// ============================================================

export function useReloadScrapersMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ReloadScrapersMutation, T.ReloadScrapersMutationVariables>(
    () => postRaw("/scrapers/reload"),
    { invalidateKeys: [queryKeys.scrapers.all] }
  );
}

export function useReloadPluginsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.ReloadPluginsMutation, T.ReloadPluginsMutationVariables>(
    () => postRaw("/plugins/reload"),
    { invalidateKeys: [queryKeys.plugins.all] }
  );
}

export function useSetPluginsEnabledMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SetPluginsEnabledMutation, T.SetPluginsEnabledMutationVariables>(
    (vars) => putAndUnwrap("/plugins/enabled", vars.enabledMap),
    { invalidateKeys: [queryKeys.plugins.all] }
  );
}

// ============================================================
// PACKAGE MUTATION HOOKS
// ============================================================

export function useInstallScraperPackagesMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.InstallScraperPackagesMutation, T.InstallScraperPackagesMutationVariables>(
    (vars) => postRaw("/packages/install", { type: "scraper", packages: vars.packages }),
    { invalidateKeys: [queryKeys.packages.installed("scraper"), queryKeys.scrapers.all] }
  );
}

export function useUpdateScraperPackagesMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.UpdateScraperPackagesMutation, T.UpdateScraperPackagesMutationVariables>(
    (vars) => postRaw("/packages/update", { type: "scraper", packages: vars.packages }),
    { invalidateKeys: [queryKeys.packages.installed("scraper"), queryKeys.scrapers.all] }
  );
}

export function useUninstallScraperPackagesMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.UninstallScraperPackagesMutation, T.UninstallScraperPackagesMutationVariables>(
    (vars) => postRaw("/packages/uninstall", { type: "scraper", packages: vars.packages }),
    { invalidateKeys: [queryKeys.packages.installed("scraper"), queryKeys.scrapers.all] }
  );
}

export function useInstallPluginPackagesMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.InstallPluginPackagesMutation, T.InstallPluginPackagesMutationVariables>(
    (vars) => postRaw("/packages/install", { type: "plugin", packages: vars.packages }),
    { invalidateKeys: [queryKeys.packages.installed("plugin"), queryKeys.plugins.all] }
  );
}

export function useUpdatePluginPackagesMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.UpdatePluginPackagesMutation, T.UpdatePluginPackagesMutationVariables>(
    (vars) => postRaw("/packages/update", { type: "plugin", packages: vars.packages }),
    { invalidateKeys: [queryKeys.packages.installed("plugin"), queryKeys.plugins.all] }
  );
}

export function useUninstallPluginPackagesMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.UninstallPluginPackagesMutation, T.UninstallPluginPackagesMutationVariables>(
    (vars) => postRaw("/packages/uninstall", { type: "plugin", packages: vars.packages }),
    { invalidateKeys: [queryKeys.packages.installed("plugin"), queryKeys.plugins.all] }
  );
}

// ============================================================
// DATABASE MUTATION HOOKS
// ============================================================

export function useBackupDatabaseMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.BackupDatabaseMutation, T.BackupDatabaseMutationVariables>(
    (vars) => postRaw("/database/backup", vars.input),
  );
}

export function useAnonymiseDatabaseMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.AnonymiseDatabaseMutation, T.AnonymiseDatabaseMutationVariables>(
    (vars) => postRaw("/database/anonymise", vars.input),
  );
}

export function useOptimiseDatabaseMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.OptimiseDatabaseMutation, T.OptimiseDatabaseMutationVariables>(
    () => postRaw("/database/optimise"),
  );
}

export function useMigrateHashNamingMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MigrateHashNamingMutation, T.MigrateHashNamingMutationVariables>(
    () => postRaw("/database/migrate-hash-naming"),
  );
}

export function useMigrateSceneScreenshotsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MigrateSceneScreenshotsMutation, T.MigrateSceneScreenshotsMutationVariables>(
    (vars) => postRaw("/database/migrate-screenshots", vars.input),
  );
}

export function useMigrateBlobsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.MigrateBlobsMutation, T.MigrateBlobsMutationVariables>(
    (vars) => postRaw("/database/migrate-blobs", vars.input),
  );
}

// ============================================================
// STASH-BOX MUTATION HOOKS
// ============================================================

export function useSubmitStashBoxFingerprintsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SubmitStashBoxFingerprintsMutation, T.SubmitStashBoxFingerprintsMutationVariables>(
    (vars) => postRaw("/stash-box/fingerprints", vars.input),
  );
}

export function useSubmitStashBoxSceneDraftMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SubmitStashBoxSceneDraftMutation, T.SubmitStashBoxSceneDraftMutationVariables>(
    (vars) => postRaw("/stash-box/scene-draft", vars.input),
  );
}

export function useSubmitStashBoxPerformerDraftMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.SubmitStashBoxPerformerDraftMutation, T.SubmitStashBoxPerformerDraftMutationVariables>(
    (vars) => postRaw("/stash-box/performer-draft", vars.input),
  );
}

export function useStashBoxBatchPerformerTagMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.StashBoxBatchPerformerTagMutation, T.StashBoxBatchPerformerTagMutationVariables>(
    (vars) => postRaw("/stash-box/batch/performers", vars.input),
  );
}

export function useStashBoxBatchStudioTagMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.StashBoxBatchStudioTagMutation, T.StashBoxBatchStudioTagMutationVariables>(
    (vars) => postRaw("/stash-box/batch/studios", vars.input),
  );
}

// ============================================================
// JOB MUTATION HOOKS
// ============================================================

export function useStopJobMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<T.StopJobMutation, T.StopJobMutationVariables>(
    (vars) => postRaw(`/jobs/${vars.job_id}/stop`),
    { invalidateKeys: [queryKeys.jobs.all] }
  );
}

export function useStopAllJobsMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, Record<string, never>>(
    () => postRaw("/jobs/stop-all"),
    { invalidateKeys: [queryKeys.jobs.all] }
  );
}

// ============================================================
// SCRAPER QUERY/MUTATION HOOKS (scrape operations)
// ============================================================

export function useScrapeSingleSceneMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, T.ScrapeSingleSceneInput>(
    (vars) => postAndUnwrap("/scrapers/scene", vars),
  );
}

export function useScrapeMultiScenesMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, T.ScrapeMultiScenesInput>(
    (vars) => postAndUnwrap("/scrapers/scenes", vars),
  );
}

export function useScrapeMultiPerformersMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, T.ScrapeMultiPerformersInput>(
    (vars) => postAndUnwrap("/scrapers/performers", vars),
  );
}

export function useScrapeSingleGalleryMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, unknown>(
    (vars) => postAndUnwrap("/scrapers/gallery", vars),
  );
}

export function useScrapeSingleGroupMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, unknown>(
    (vars) => postAndUnwrap("/scrapers/group", vars),
  );
}

export function useScrapeSingleImageMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, unknown>(
    (vars) => postAndUnwrap("/scrapers/image", vars),
  );
}

export function useScrapeURLMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { url: string }>(
    (vars) => postAndUnwrap("/scrapers/url", vars),
  );
}

// File management mutations
export function useMoveFilesMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, T.MoveFilesInput>(
    (vars) => postRaw("/files/move", vars),
    { invalidateKeys: [queryKeys.files.all] }
  );
}

export function useDeleteFilesMutation(baseOptions?: MutationHookOptions) {
  return useRestMutation<unknown, { ids: string[] }>(
    (vars) => apiDelete("/files", vars),
    { invalidateKeys: [queryKeys.files.all] }
  );
}

// ============================================================
// SUBSCRIPTION HOOKS (SSE-based)
// ============================================================

/**
 * Hook for job subscription updates via SSE.
 * Replaces useJobsSubscribeSubscription from Apollo/GraphQL.
 */
export function useJobsSubscribeSubscription(
  baseOptions?: SubscriptionHookOptions<T.JobsSubscribeSubscription, T.JobsSubscribeSubscriptionVariables>
) {
  const [data, setData] = useState<T.JobsSubscribeSubscription | undefined>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | undefined>();

  useEffect(() => {
    const sse = getSSEClient();
    setLoading(false);

    const unsub = sse.onJobUpdate((update: SSEJobStatusUpdate) => {
      setData({
        jobsSubscribe: {
          __typename: "JobStatusUpdate",
          type: update.type as T.JobStatusUpdateType,
          job: {
            __typename: "Job",
            id: update.job.id,
            status: update.job.status as T.JobStatus,
            subTasks: update.job.subTasks ?? null,
            description: update.job.description,
            progress: update.job.progress ?? null,
            error: update.job.error ?? null,
            startTime: update.job.startTime ?? null,
          },
        },
      } as T.JobsSubscribeSubscription);
    });

    return unsub;
  }, []);

  return { data, loading, error };
}

/**
 * Hook for logging subscription via SSE.
 * Replaces useLoggingSubscribeSubscription from Apollo/GraphQL.
 */
export function useLoggingSubscribeSubscription(
  baseOptions?: SubscriptionHookOptions<T.LoggingSubscribeSubscription, T.LoggingSubscribeSubscriptionVariables>
) {
  const [data, setData] = useState<T.LoggingSubscribeSubscription | undefined>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | undefined>();

  useEffect(() => {
    const sse = getSSEClient();
    setLoading(false);

    const unsub = sse.onLogEntries((entries: SSELogEntry[]) => {
      setData({
        loggingSubscribe: entries.map((e) => ({
          __typename: "LogEntry" as const,
          time: e.time,
          level: e.level as T.LogLevel,
          message: e.message,
        })),
      } as T.LoggingSubscribeSubscription);
    });

    return unsub;
  }, []);

  return { data, loading, error };
}

/**
 * Hook for scan complete subscription via SSE.
 * Replaces useScanCompleteSubscribeSubscription from Apollo/GraphQL.
 */
export function useScanCompleteSubscribeSubscription(
  baseOptions?: SubscriptionHookOptions<T.ScanCompleteSubscribeSubscription, T.ScanCompleteSubscribeSubscriptionVariables>
) {
  const [data, setData] = useState<T.ScanCompleteSubscribeSubscription | undefined>();

  useEffect(() => {
    const sse = getSSEClient();

    const unsub = sse.onScanComplete(() => {
      setData({
        scanCompleteSubscribe: true,
      } as T.ScanCompleteSubscribeSubscription);
    });

    return unsub;
  }, []);

  return { data, loading: false, error: undefined };
}

// ============================================================
// DOCUMENT CONSTANTS (for backward compatibility)
// These were previously gql`` template literal constants used
// for cache manipulation. Now they serve as query key identifiers.
// ============================================================

// Query Documents (used as query key identifiers for cache invalidation)
export const FindSceneDocument = queryKeys.scenes.all;
export const FindScenesDocument = queryKeys.scenes.all;
export const FindScenesForSelectDocument = queryKeys.scenes.all;
export const FindScenesByPathRegexDocument = queryKeys.scenes.all;
export const FindSimilarScenesDocument = queryKeys.scenes.all;
export const FindDuplicateScenesDocument = queryKeys.scenes.all;
export const FindImageDocument = queryKeys.images.all;
export const FindImagesDocument = queryKeys.images.all;
export const FindGalleryDocument = queryKeys.galleries.all;
export const FindGalleriesDocument = queryKeys.galleries.all;
export const FindGalleriesForSelectDocument = queryKeys.galleries.all;
export const FindPerformerDocument = queryKeys.performers.all;
export const FindPerformersDocument = queryKeys.performers.all;
export const FindPerformersForSelectDocument = queryKeys.performers.all;
export const FindStudioDocument = queryKeys.studios.all;
export const FindStudiosDocument = queryKeys.studios.all;
export const FindStudiosForSelectDocument = queryKeys.studios.all;
export const FindTagDocument = queryKeys.tags.all;
export const FindTagsDocument = queryKeys.tags.all;
export const FindTagsForSelectDocument = queryKeys.tags.all;
export const FindGroupDocument = queryKeys.groups.all;
export const FindGroupsDocument = queryKeys.groups.all;
export const FindGroupsForSelectDocument = queryKeys.groups.all;
export const FindGamesDocument = queryKeys.games.all;
export const FindGameDocument = queryKeys.games.all;
export const FindSceneMarkersDocument = queryKeys.sceneMarkers.all;
export const StatsDocument = queryKeys.stats;
export const ConfigurationDocument = queryKeys.config;
export const SystemStatusDocument = queryKeys.systemStatus;
export const PluginsDocument = queryKeys.plugins.all;
export const PluginTasksDocument = queryKeys.plugins.tasks;
export const ListSceneScrapersDocument = queryKeys.scrapers.all;
export const ListPerformerScrapersDocument = queryKeys.scrapers.all;
export const ListGroupScrapersDocument = queryKeys.scrapers.all;
export const ListGalleryScrapersDocument = queryKeys.scrapers.all;
export const ListImageScrapersDocument = queryKeys.scrapers.all;
export const InstalledScraperPackagesDocument = queryKeys.packages.installed("scraper");
export const InstalledScraperPackagesStatusDocument = [...queryKeys.packages.installed("scraper"), "status"] as const;
export const InstalledPluginPackagesDocument = queryKeys.packages.installed("plugin");
export const InstalledPluginPackagesStatusDocument = [...queryKeys.packages.installed("plugin"), "status"] as const;
export const AvailableScraperPackagesDocument = ["packages", "available", "scraper"] as const;
export const AvailablePluginPackagesDocument = ["packages", "available", "plugin"] as const;
export const LogsDocument = ["logs"] as const;
export const JobQueueDocument = queryKeys.jobs.all;

// Mutation Documents (no-ops, kept for backward compatibility)
export const SceneCreateDocument = "SceneCreate";
export const SceneUpdateDocument = "SceneUpdate";
export const SceneAssignFileDocument = "SceneAssignFile";
export const SceneMergeDocument = "SceneMerge";
export const ImageIncrementODocument = "ImageIncrementO";
export const ImageDecrementODocument = "ImageDecrementO";
export const ImageResetODocument = "ImageResetO";
export const ReloadScrapersDocument = "ReloadScrapers";
export const ReloadPluginsDocument = "ReloadPlugins";
export const SetPluginsEnabledDocument = "SetPluginsEnabled";
export const MetadataScanDocument = "MetadataScan";
export const MetadataIdentifyDocument = "MetadataIdentify";
export const MetadataAutoTagDocument = "MetadataAutoTag";
export const MetadataGenerateDocument = "MetadataGenerate";
export const MetadataCleanDocument = "MetadataClean";
export const MetadataCleanGeneratedDocument = "MetadataCleanGenerated";
export const MetadataExportDocument = "MetadataExport";
export const MetadataImportDocument = "MetadataImport";
export const ExportObjectsDocument = "ExportObjects";
export const ImportObjectsDocument = "ImportObjects";
export const RunPluginTaskDocument = "RunPluginTask";
export const StopJobDocument = "StopJob";
export const SetupDocument = "Setup";
export const MigrateDocument = "Migrate";
export const DownloadFfMpegDocument = "DownloadFFMpeg";
export const BackupDatabaseDocument = "BackupDatabase";
export const AnonymiseDatabaseDocument = "AnonymiseDatabase";
export const OptimiseDatabaseDocument = "OptimiseDatabase";
export const MigrateHashNamingDocument = "MigrateHashNaming";
export const MigrateSceneScreenshotsDocument = "MigrateSceneScreenshots";
export const MigrateBlobsDocument = "MigrateBlobs";
export const RecalculateSceneSimilaritiesDocument = "RecalculateSceneSimilarities";
export const ParseSceneFilenamesDocument = "ParseSceneFilenames";
export const ScanCompleteSubscribeDocument = "ScanCompleteSubscribe";
export const JobsSubscribeDocument = "JobsSubscribe";
export const LoggingSubscribeDocument = "LoggingSubscribe";

// ============================================================
// QueryResult type aliases for Apollo backward compatibility
// ============================================================
export type FindScenesQueryResult = QueryResult<T.FindScenesQuery>;
export type FindImagesQueryResult = QueryResult<T.FindImagesQuery>;
export type FindGalleriesQueryResult = QueryResult<T.FindGalleriesQuery>;
export type FindPerformersQueryResult = QueryResult<T.FindPerformersQuery>;
export type FindStudiosQueryResult = QueryResult<T.FindStudiosQuery>;
export type FindTagsQueryResult = QueryResult<T.FindTagsQuery>;
export type FindGroupsQueryResult = QueryResult<T.FindGroupsQuery>;
export type FindSceneMarkersQueryResult = QueryResult<T.FindSceneMarkersQuery>;
export type FindGamesQueryResult = QueryResult<T.FindGamesQuery>;

// Fragment documents (no-ops in REST)
export const ConfigGeneralDataFragmentDoc = {};
export const ConfigInterfaceDataFragmentDoc = {};
export const ConfigDlnaDataFragmentDoc = {};
export const ConfigScrapingDataFragmentDoc = {};
export const GalleryDataFragmentDoc = {};
export const SceneDataFragmentDoc = {};
