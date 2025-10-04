export interface Scene {
  id: string;
  title?: string;
  play_count?: number;
  resume_time?: number;
  force_hls?: boolean;
  is_broken?: boolean;
  is_probably_broken?: boolean;
  is_not_broken?: boolean;
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

export interface Gallery {
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
  display_mode?: number;
  paths?: {
    cover?: string;
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
    path?: string;
  }>;
}

export interface ViewHistoryEntry {
  scene?: Scene;
  gallery?: Gallery;
  viewDate: string;
  oDate?: string;
  viewCount?: number;
}

export interface ViewHistoryResult {
  count: number;
  items: ViewHistoryEntry[];
}