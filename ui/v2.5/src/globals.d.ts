
declare module "*.md" {
  const src: string;
  export default src;
}

declare module "intersection-observer-polyfill";

// eslint-disable-next-line @typescript-eslint/naming-convention
interface ImportMetaEnv {
  readonly VITE_APP_GITHASH?: string;
  readonly VITE_APP_STASH_VERSION?: string;
  readonly VITE_APP_DATE?: string;
  readonly VITE_APP_PLATFORM_URL?: string;
}
