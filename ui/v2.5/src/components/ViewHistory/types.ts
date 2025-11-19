export interface IScene {
  id: string;
  title?: string;
  play_count?: number;
  resume_time?: number;
  force_hls?: boolean;
  is_broken?: boolean;
  is_probably_broken?: boolean;
  is_not_broken?: boolean;
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

export interface IGallery {
  id: string;
  title?: string;
  code?: string;
  date?: string;
  details?: string;
  photographer?: string;
  rating100?: number;
  organized?: boolean;
  pinned?: boolean;
  o_counter?: number;
  play_count?: number;
  view_history?: string[];
  display_mode?: number;
  image_count?: number;
  paths?: {
    cover?: string;
    preview?: string;
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
  files?: Array<{
    id: string;
    path: string;
    size: number;
    mod_time: string;
    fingerprints: Array<{
      type: string;
      value: string;
    }>;
  }>;
}

export interface IViewHistoryEntry {
  scene?: IScene;
  gallery?: IGallery;
  viewDate: string;
  oDate?: string;
  viewCount?: number;
}

export interface IViewHistoryResult {
  count: number;
  items: IViewHistoryEntry[];
}
