/**
 * Client configuration and URL helpers.
 *
 * Previously this file set up Apollo Client, WebSocket links, and caching.
 * Now it only provides URL utility functions used throughout the application.
 * Data fetching is handled by rest-client.ts + rest-hooks.ts + TanStack Query.
 */

export const baseURL =
  document.querySelector("base")?.getAttribute("href") ?? "/";

export const getPlatformURL = (path?: string) => {
  let url = new URL(window.location.origin + baseURL);

  if (import.meta.env.DEV) {
    if (import.meta.env.VITE_APP_PLATFORM_URL) {
      url = new URL(import.meta.env.VITE_APP_PLATFORM_URL);
    } else {
      url.port = import.meta.env.VITE_APP_PLATFORM_PORT ?? "9999";
    }
  }

  if (path) {
    // Avoid double slashes: if pathname ends with "/" and path starts with "/",
    // strip the leading "/" from path before appending
    if (url.pathname.endsWith("/") && path.startsWith("/")) {
      url.pathname += path.slice(1);
    } else {
      url.pathname += path;
    }
  }

  return url;
};

/**
 * Resolve a relative API path (e.g. "/api/v1/scenes/123/screenshot")
 * into an absolute URL pointing to the backend.
 * In production this is a no-op (same origin).
 * In dev mode it rewrites the origin to the backend port.
 */
export const resolveApiPath = (path: string | null | undefined): string | undefined => {
  if (!path) return undefined;
  // Already absolute
  if (path.startsWith("http://") || path.startsWith("https://")) return path;
  return getPlatformURL(path).toString();
};
