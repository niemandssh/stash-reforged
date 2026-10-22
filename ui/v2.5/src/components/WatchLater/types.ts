export interface IWatchLaterScene {
  id: string;
  title?: string;
  play_count?: number;
  resume_time?: number;
  force_hls?: boolean;
  is_broken?: boolean;
  is_probably_broken?: boolean;
  is_not_broken?: boolean;
  watch_later?: boolean;
  watch_later_at?: string;
  video_filters?: {
    contrast?: number | null;
    brightness?: number | null;
    gamma?: number | null;
    saturate?: number | null;
    hue_rotate?: number | null;
    white_balance?: number | null;
    red?: number | null;
    green?: number | null;
    blue?: number | null;
    blur?: number | null;
  } | null;
  video_transforms?: {
    rotate?: number | null;
    scale?: number | null;
    aspect_ratio?: number | null;
  } | null;
  files?: Array<{
    path?: string;
    duration?: number;
  }>;
  paths?: {
    screenshot?: string;
    preview?: string;
    vtt?: string;
  };
  performers?: Array<{
    id: string;
    name?: string;
    gender?: string;
  }>;
  studio?: {
    id: string;
    name?: string;
    image_path?: string;
  };
}

export interface IWatchLaterEntry {
  scene: IWatchLaterScene;
  addedAt: string;
}

export interface IWatchLaterResult {
  count: number;
  items: IWatchLaterEntry[];
}
