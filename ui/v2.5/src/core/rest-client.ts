/**
 * REST API Client for Stash.
 *
 * Provides typed methods for communicating with the REST API at /api/v1.
 * Replaces Apollo Client / GraphQL for all data fetching and mutations.
 */

import { getPlatformURL } from "./createClient";

const API_BASE = "/api/v1";

function getApiURL(path: string): string {
  // Split path and query string to avoid encoding '?' as '%3F' in pathname
  const qIdx = path.indexOf("?");
  const pathname = qIdx >= 0 ? path.slice(0, qIdx) : path;
  const search = qIdx >= 0 ? path.slice(qIdx) : "";
  const url = getPlatformURL(API_BASE + pathname);
  if (search) {
    url.search = search;
  }
  return url.toString();
}

/**
 * Generic API error class with status code and optional error code.
 */
export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

/**
 * Handles the response from the REST API.
 * Throws ApiError on non-OK responses.
 */
async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let message = `HTTP ${response.status}: ${response.statusText}`;
    let code: string | undefined;

    try {
      const body = await response.json();
      if (body.error) {
        message = body.error;
      }
      if (body.code) {
        code = body.code;
      }
    } catch {
      // response body wasn't JSON
    }

    // Handle 401 Unauthorized
    if (response.status === 401) {
      if (import.meta.env.DEV) {
        alert(
          `REST API error: 401 Unauthorized\n` +
            `Authentication cannot be used with the dev server.`
        );
        throw new ApiError(401, message, code);
      }

      const loginURL = getPlatformURL("login");
      const newURL = new URL(loginURL.toString(), window.location.toString());
      newURL.searchParams.append("returnURL", window.location.href);
      window.location.href = newURL.toString();
    }

    throw new ApiError(response.status, message, code);
  }

  // 204 No Content
  if (response.status === 204) {
    return undefined as unknown as T;
  }

  return response.json();
}

/**
 * Low-level fetch wrapper with default headers and credentials.
 */
async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = getApiURL(path);

  const headers: Record<string, string> = {
    Accept: "application/json",
    ...(options.headers as Record<string, string>),
  };

  // Don't set Content-Type for FormData (browser will set it with boundary)
  if (options.body && !(options.body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
  }

  const response = await fetch(url, {
    ...options,
    headers,
    credentials: "same-origin",
  });

  return handleResponse<T>(response);
}

// --- HTTP Method helpers ---

export async function apiGet<T>(path: string): Promise<T> {
  return apiFetch<T>(path, { method: "GET" });
}

export async function apiPost<T>(
  path: string,
  body?: unknown
): Promise<T> {
  return apiFetch<T>(path, {
    method: "POST",
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
}

export async function apiPut<T>(
  path: string,
  body?: unknown
): Promise<T> {
  return apiFetch<T>(path, {
    method: "PUT",
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
}

export async function apiPatch<T>(
  path: string,
  body?: unknown
): Promise<T> {
  return apiFetch<T>(path, {
    method: "PATCH",
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
}

export async function apiDelete<T>(
  path: string,
  body?: unknown
): Promise<T> {
  return apiFetch<T>(path, {
    method: "DELETE",
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
}

/**
 * Upload a file using FormData.
 */
export async function apiUpload<T>(
  path: string,
  formData: FormData
): Promise<T> {
  return apiFetch<T>(path, {
    method: "POST",
    body: formData,
  });
}

// --- Query helpers ---

/**
 * Build a query endpoint for POST-based filtering/searching.
 * Used for endpoints like /scenes/query, /performers/query, etc.
 */
export async function apiQuery<T>(
  entityPath: string,
  filter?: Record<string, unknown>,
  entityFilter?: Record<string, unknown>
): Promise<T> {
  return apiPost<T>(`${entityPath}/query`, {
    filter,
    ...(entityFilter || {}),
  });
}

// --- Typed API namespaces ---

export const restApi = {
  // System
  systemStatus: () => apiGet<unknown>("/system/status"),
  version: () => apiGet<unknown>("/system/version"),
  stats: () => apiGet<unknown>("/system/stats"),

  // Config
  getConfig: () => apiGet<unknown>("/config"),
  updateConfig: (input: unknown) => apiPost<unknown>("/config/general", input),

  // Tags
  findTag: (id: number | string) => apiGet<unknown>(`/tags/${id}`),
  findTags: (body: unknown) => apiPost<unknown>("/tags/query", body),
  createTag: (input: unknown) => apiPost<unknown>("/tags", input),
  updateTag: (id: number | string, input: unknown) =>
    apiPut<unknown>(`/tags/${id}`, input),
  deleteTag: (id: number | string) => apiDelete<unknown>(`/tags/${id}`),

  // Studios
  findStudio: (id: number | string) => apiGet<unknown>(`/studios/${id}`),
  findStudios: (body: unknown) => apiPost<unknown>("/studios/query", body),
  createStudio: (input: unknown) => apiPost<unknown>("/studios", input),
  updateStudio: (id: number | string, input: unknown) =>
    apiPut<unknown>(`/studios/${id}`, input),
  deleteStudio: (id: number | string) =>
    apiDelete<unknown>(`/studios/${id}`),

  // Scenes
  findScene: (id: number | string) => apiGet<unknown>(`/scenes/${id}`),
  findScenes: (body: unknown) => apiPost<unknown>("/scenes/query", body),
  createScene: (input: unknown) => apiPost<unknown>("/scenes", input),
  updateScene: (id: number | string, input: unknown) =>
    apiPut<unknown>(`/scenes/${id}`, input),
  deleteScene: (id: number | string) =>
    apiDelete<unknown>(`/scenes/${id}`),
  bulkUpdateScenes: (input: unknown) =>
    apiPut<unknown>("/scenes/bulk", input),

  // Performers
  findPerformer: (id: number | string) =>
    apiGet<unknown>(`/performers/${id}`),
  findPerformers: (body: unknown) =>
    apiPost<unknown>("/performers/query", body),
  createPerformer: (input: unknown) =>
    apiPost<unknown>("/performers", input),
  updatePerformer: (id: number | string, input: unknown) =>
    apiPut<unknown>(`/performers/${id}`, input),
  deletePerformer: (id: number | string) =>
    apiDelete<unknown>(`/performers/${id}`),
  bulkUpdatePerformers: (input: unknown) =>
    apiPut<unknown>("/performers/bulk", input),

  // Galleries
  findGallery: (id: number | string) =>
    apiGet<unknown>(`/galleries/${id}`),
  findGalleries: (body: unknown) =>
    apiPost<unknown>("/galleries/query", body),
  createGallery: (input: unknown) =>
    apiPost<unknown>("/galleries", input),
  updateGallery: (id: number | string, input: unknown) =>
    apiPut<unknown>(`/galleries/${id}`, input),
  deleteGallery: (id: number | string) =>
    apiDelete<unknown>(`/galleries/${id}`),

  // Images
  findImage: (id: number | string) => apiGet<unknown>(`/images/${id}`),
  findImages: (body: unknown) => apiPost<unknown>("/images/query", body),
  updateImage: (id: number | string, input: unknown) =>
    apiPut<unknown>(`/images/${id}`, input),
  deleteImage: (id: number | string) =>
    apiDelete<unknown>(`/images/${id}`),

  // Groups
  findGroup: (id: number | string) => apiGet<unknown>(`/groups/${id}`),
  findGroups: (body: unknown) => apiPost<unknown>("/groups/query", body),
  createGroup: (input: unknown) => apiPost<unknown>("/groups", input),
  updateGroup: (id: number | string, input: unknown) =>
    apiPut<unknown>(`/groups/${id}`, input),
  deleteGroup: (id: number | string) =>
    apiDelete<unknown>(`/groups/${id}`),

  // Games
  findGame: (id: number | string) => apiGet<unknown>(`/games/${id}`),
  findGames: (body: unknown) => apiPost<unknown>("/games/query", body),
  createGame: (input: unknown) => apiPost<unknown>("/games", input),
  updateGame: (id: number | string, input: unknown) =>
    apiPut<unknown>(`/games/${id}`, input),
  deleteGame: (id: number | string) =>
    apiDelete<unknown>(`/games/${id}`),

  // Scene Markers
  findSceneMarkers: (body: unknown) =>
    apiPost<unknown>("/scene-markers/query", body),
  createSceneMarker: (input: unknown) =>
    apiPost<unknown>("/scene-markers", input),
  updateSceneMarker: (id: number | string, input: unknown) =>
    apiPut<unknown>(`/scene-markers/${id}`, input),
  deleteSceneMarker: (id: number | string) =>
    apiDelete<unknown>(`/scene-markers/${id}`),

  // Saved Filters
  findSavedFilter: (id: number | string) =>
    apiGet<unknown>(`/filters/${id}`),
  findSavedFilters: (mode?: string) =>
    apiGet<unknown>(`/filters${mode ? `?mode=${mode}` : ""}`),
  findDefaultFilter: (mode: string) =>
    apiGet<unknown>(`/filters/default?mode=${mode}`),
  saveFilter: (input: unknown) => apiPost<unknown>("/filters", input),
  setDefaultFilter: (input: unknown) =>
    apiPost<unknown>("/filters/default", input),
  destroySavedFilter: (id: number | string) =>
    apiDelete<unknown>(`/filters/${id}`),

  // Files & Folders
  findFile: (id: number | string) => apiGet<unknown>(`/files/${id}`),
  findFiles: (body: unknown) => apiPost<unknown>("/files/query", body),
  moveFiles: (input: unknown) => apiPost<unknown>("/files/move", input),
  deleteFiles: (input: unknown) =>
    apiDelete<unknown>("/files", input),
  setFileFingerprints: (input: unknown) =>
    apiPost<unknown>("/files/fingerprints", input),
  findFolder: (id: number | string) =>
    apiGet<unknown>(`/folders/${id}`),
  findFolders: (body: unknown) =>
    apiPost<unknown>("/folders/query", body),

  // Metadata operations
  metadataScan: (input: unknown) =>
    apiPost<unknown>("/metadata/scan", input),
  metadataGenerate: (input: unknown) =>
    apiPost<unknown>("/metadata/generate", input),
  metadataAutoTag: (input: unknown) =>
    apiPost<unknown>("/metadata/auto-tag", input),
  metadataClean: (input: unknown) =>
    apiPost<unknown>("/metadata/clean", input),
  metadataCleanGenerated: (input: unknown) =>
    apiPost<unknown>("/metadata/clean-generated", input),
  metadataIdentify: (input: unknown) =>
    apiPost<unknown>("/metadata/identify", input),
  metadataExport: () => apiPost<unknown>("/metadata/export"),
  metadataImport: () => apiPost<unknown>("/metadata/import"),
  exportObjects: (input: unknown) =>
    apiPost<unknown>("/metadata/export-objects", input),
  importObjects: (input: unknown) =>
    apiPost<unknown>("/metadata/import-objects", input),

  // Scrapers
  listScrapers: (types?: string[]) =>
    apiGet<unknown>(
      `/scrapers${types ? `?${types.map((t) => `type=${t}`).join("&")}` : ""}`
    ),
  reloadScrapers: () => apiPost<unknown>("/scrapers/reload"),

  // Plugins
  getPlugins: () => apiGet<unknown>("/plugins"),
  getPluginTasks: () => apiGet<unknown>("/plugins/tasks"),
  reloadPlugins: () => apiPost<unknown>("/plugins/reload"),
  setPluginsEnabled: (input: Record<string, boolean>) =>
    apiPost<unknown>("/plugins/enabled", input),
  runPluginTask: (input: unknown) =>
    apiPost<unknown>("/plugins/run-task", input),
  runPluginOperation: (input: unknown) =>
    apiPost<unknown>("/plugins/run-operation", input),

  // Packages
  getInstalledPackages: (type: string, upgrades = false) =>
    apiGet<unknown>(
      `/packages/installed?type=${type}${upgrades ? "&upgrades=true" : ""}`
    ),
  getAvailablePackages: (type: string, source: string) =>
    apiGet<unknown>(
      `/packages/available?type=${type}&source=${encodeURIComponent(source)}`
    ),
  installPackages: (input: unknown) =>
    apiPost<unknown>("/packages/install", input),
  updatePackages: (input: unknown) =>
    apiPost<unknown>("/packages/update", input),
  uninstallPackages: (input: unknown) =>
    apiPost<unknown>("/packages/uninstall", input),

  // Jobs
  getJobQueue: () => apiGet<unknown>("/jobs"),
  findJob: (id: number | string) => apiGet<unknown>(`/jobs/${id}`),
  stopJob: (id: number | string) => apiDelete<unknown>(`/jobs/${id}`),
  stopAllJobs: () => apiDelete<unknown>("/jobs"),

  // DLNA
  getDLNAStatus: () => apiGet<unknown>("/dlna/status"),
  enableDLNA: (input?: unknown) => apiPost<unknown>("/dlna/enable", input),
  disableDLNA: (input?: unknown) =>
    apiPost<unknown>("/dlna/disable", input),
  addTempDLNAIP: (input: unknown) =>
    apiPost<unknown>("/dlna/ip/add", input),
  removeTempDLNAIP: (input: unknown) =>
    apiPost<unknown>("/dlna/ip/remove", input),

  // SQL
  querySQL: (sql: string, args?: unknown[]) =>
    apiPost<unknown>("/sql/query", { sql, args }),
  execSQL: (sql: string, args?: unknown[]) =>
    apiPost<unknown>("/sql/exec", { sql, args }),

  // StashBox
  submitStashBoxFingerprints: (input: unknown) =>
    apiPost<unknown>("/stashbox/fingerprints", input),
  submitStashBoxSceneDraft: (input: unknown) =>
    apiPost<unknown>("/stashbox/scene-draft", input),
  submitStashBoxPerformerDraft: (input: unknown) =>
    apiPost<unknown>("/stashbox/performer-draft", input),
  stashBoxBatchPerformerTag: (input: unknown) =>
    apiPost<unknown>("/stashbox/batch-performer-tag", input),
  stashBoxBatchStudioTag: (input: unknown) =>
    apiPost<unknown>("/stashbox/batch-studio-tag", input),

  // View History
  findViewHistory: (body: unknown) =>
    apiPost<unknown>("/view-history/query", body),

  // Database
  backupDatabase: (input?: unknown) =>
    apiPost<unknown>("/database/backup", input),
  anonymiseDatabase: (input?: unknown) =>
    apiPost<unknown>("/database/anonymise", input),
  optimiseDatabase: () => apiPost<unknown>("/database/optimise"),
  migrateHashNaming: () =>
    apiPost<unknown>("/database/migrate-hash-naming"),
  migrateSceneScreenshots: (input: unknown) =>
    apiPost<unknown>("/database/migrate-scene-screenshots", input),
  migrateBlobs: (input: unknown) =>
    apiPost<unknown>("/database/migrate-blobs", input),

  // Color Presets
  findColorPresets: () => apiGet<unknown>("/color-presets"),
  createColorPreset: (input: unknown) =>
    apiPost<unknown>("/color-presets", input),
  updateColorPreset: (id: number | string, input: unknown) =>
    apiPut<unknown>(`/color-presets/${id}`, input),
  deleteColorPreset: (id: number | string) =>
    apiDelete<unknown>(`/color-presets/${id}`),
};

export default restApi;
