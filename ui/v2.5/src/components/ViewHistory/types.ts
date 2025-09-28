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

export interface ViewHistoryEntry {
  scene: Scene;
  viewDate: string;
  oDate?: string;
  viewCount?: number;
}

export interface ViewHistoryResult {
  count: number;
  items: ViewHistoryEntry[];
}