/**
 * TanStack Query (React Query) configuration for Stash.
 *
 * Provides a pre-configured QueryClient with sensible defaults,
 * and integrates with the SSE client to invalidate queries
 * when relevant real-time events occur (e.g. scan complete).
 */

import { QueryClient } from "@tanstack/react-query";
import { getSSEClient } from "./sse-client";

/**
 * Default stale time: 30 seconds.
 * Data is considered fresh for this period and won't be refetched.
 */
const DEFAULT_STALE_TIME = 30 * 1000;

/**
 * Default cache time: 5 minutes.
 * Unused data is garbage collected after this period.
 */
const DEFAULT_CACHE_TIME = 5 * 60 * 1000;

/**
 * Create and configure the QueryClient singleton.
 */
export function createQueryClient(): QueryClient {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: DEFAULT_STALE_TIME,
        cacheTime: DEFAULT_CACHE_TIME,
        refetchOnWindowFocus: true,
        retry: 1,
      },
      mutations: {
        retry: 0,
      },
    },
  });

  // Wire up SSE events to invalidate relevant queries
  const sse = getSSEClient();

  // When a scan completes, invalidate all entity queries
  // (new files may have been discovered)
  sse.onScanComplete(() => {
    queryClient.invalidateQueries(["scenes"]);
    queryClient.invalidateQueries(["images"]);
    queryClient.invalidateQueries(["galleries"]);
    queryClient.invalidateQueries(["performers"]);
    queryClient.invalidateQueries(["studios"]);
    queryClient.invalidateQueries(["tags"]);
    queryClient.invalidateQueries(["groups"]);
    queryClient.invalidateQueries(["games"]);
    queryClient.invalidateQueries(["files"]);
    queryClient.invalidateQueries(["folders"]);
    queryClient.invalidateQueries(["stats"]);
  });

  // When jobs change, invalidate job queries
  sse.onJobUpdate(() => {
    queryClient.invalidateQueries(["jobs"]);
  });

  return queryClient;
}

// Singleton
let queryClientInstance: QueryClient | null = null;

export function getQueryClient(): QueryClient {
  if (!queryClientInstance) {
    queryClientInstance = createQueryClient();
  }
  return queryClientInstance;
}

// --- Query key factories ---
// Consistent key structure for cache management.

export const queryKeys = {
  // Scenes
  scenes: {
    all: ["scenes"] as const,
    lists: () => [...queryKeys.scenes.all, "list"] as const,
    list: (filter: unknown) =>
      [...queryKeys.scenes.lists(), filter] as const,
    details: () => [...queryKeys.scenes.all, "detail"] as const,
    detail: (id: string | number) =>
      [...queryKeys.scenes.details(), id] as const,
  },

  // Performers
  performers: {
    all: ["performers"] as const,
    lists: () => [...queryKeys.performers.all, "list"] as const,
    list: (filter: unknown) =>
      [...queryKeys.performers.lists(), filter] as const,
    details: () => [...queryKeys.performers.all, "detail"] as const,
    detail: (id: string | number) =>
      [...queryKeys.performers.details(), id] as const,
  },

  // Studios
  studios: {
    all: ["studios"] as const,
    lists: () => [...queryKeys.studios.all, "list"] as const,
    list: (filter: unknown) =>
      [...queryKeys.studios.lists(), filter] as const,
    details: () => [...queryKeys.studios.all, "detail"] as const,
    detail: (id: string | number) =>
      [...queryKeys.studios.details(), id] as const,
  },

  // Tags
  tags: {
    all: ["tags"] as const,
    lists: () => [...queryKeys.tags.all, "list"] as const,
    list: (filter: unknown) =>
      [...queryKeys.tags.lists(), filter] as const,
    details: () => [...queryKeys.tags.all, "detail"] as const,
    detail: (id: string | number) =>
      [...queryKeys.tags.details(), id] as const,
  },

  // Galleries
  galleries: {
    all: ["galleries"] as const,
    lists: () => [...queryKeys.galleries.all, "list"] as const,
    list: (filter: unknown) =>
      [...queryKeys.galleries.lists(), filter] as const,
    details: () => [...queryKeys.galleries.all, "detail"] as const,
    detail: (id: string | number) =>
      [...queryKeys.galleries.details(), id] as const,
  },

  // Images
  images: {
    all: ["images"] as const,
    lists: () => [...queryKeys.images.all, "list"] as const,
    list: (filter: unknown) =>
      [...queryKeys.images.lists(), filter] as const,
    details: () => [...queryKeys.images.all, "detail"] as const,
    detail: (id: string | number) =>
      [...queryKeys.images.details(), id] as const,
  },

  // Groups
  groups: {
    all: ["groups"] as const,
    lists: () => [...queryKeys.groups.all, "list"] as const,
    list: (filter: unknown) =>
      [...queryKeys.groups.lists(), filter] as const,
    details: () => [...queryKeys.groups.all, "detail"] as const,
    detail: (id: string | number) =>
      [...queryKeys.groups.details(), id] as const,
  },

  // Games
  games: {
    all: ["games"] as const,
    lists: () => [...queryKeys.games.all, "list"] as const,
    list: (filter: unknown) =>
      [...queryKeys.games.lists(), filter] as const,
    details: () => [...queryKeys.games.all, "detail"] as const,
    detail: (id: string | number) =>
      [...queryKeys.games.details(), id] as const,
  },

  // Scene Markers
  sceneMarkers: {
    all: ["scene-markers"] as const,
    lists: () => [...queryKeys.sceneMarkers.all, "list"] as const,
    list: (filter: unknown) =>
      [...queryKeys.sceneMarkers.lists(), filter] as const,
  },

  // Files & Folders
  files: {
    all: ["files"] as const,
    detail: (id: string | number) => ["files", "detail", id] as const,
  },
  folders: {
    all: ["folders"] as const,
    detail: (id: string | number) => ["folders", "detail", id] as const,
  },

  // Saved Filters
  filters: {
    all: ["filters"] as const,
    list: (mode?: string) => ["filters", "list", mode] as const,
    detail: (id: string | number) => ["filters", "detail", id] as const,
    default: (mode: string) => ["filters", "default", mode] as const,
  },

  // Color Presets
  colorPresets: {
    all: ["color-presets"] as const,
  },

  // System
  config: ["config"] as const,
  systemStatus: ["system", "status"] as const,
  version: ["system", "version"] as const,
  stats: ["stats"] as const,

  // Jobs
  jobs: {
    all: ["jobs"] as const,
    detail: (id: string | number) => ["jobs", "detail", id] as const,
  },

  // Scrapers
  scrapers: {
    all: ["scrapers"] as const,
    list: (types?: string[]) => ["scrapers", "list", types] as const,
  },

  // Plugins
  plugins: {
    all: ["plugins"] as const,
    tasks: ["plugins", "tasks"] as const,
  },

  // Packages
  packages: {
    installed: (type: string) => ["packages", "installed", type] as const,
    available: (type: string, source: string) =>
      ["packages", "available", type, source] as const,
  },

  // DLNA
  dlna: {
    status: ["dlna", "status"] as const,
  },

  // View History
  viewHistory: {
    all: ["view-history"] as const,
  },
};
