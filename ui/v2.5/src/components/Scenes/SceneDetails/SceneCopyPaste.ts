import * as GQL from "src/core/generated-graphql";
import { ListFilterModel } from "src/models/list-filter/filter";
import {
  queryFindPerformersForSelect,
  queryFindTagsForSelect,
  queryFindStudiosForSelect,
  queryFindGroupsForSelect,
} from "src/core/StashService";

const SCENE_COPY_PREFIX = "stash-scene-data:";
const SCENE_COPY_VERSION = 1;

export interface SceneCopyPerformer {
  name: string;
  small_role: boolean;
  role_description: string | null;
}

export interface SceneCopyPerformerTags {
  performer_name: string;
  tag_names: string[];
}

export interface SceneCopyGroup {
  name: string;
  scene_index: number | null;
}

export interface SceneCopyPayload {
  version: number;
  title?: string | null;
  code?: string | null;
  details?: string | null;
  director?: string | null;
  urls: string[];
  date?: string | null;
  shoot_date?: string | null;
  rating100?: number | null;
  organized: boolean;
  play_count?: number | null;
  start_time?: number | null;
  end_time?: number | null;
  is_broken: boolean;
  is_not_broken: boolean;
  disable_next_scene_overlay: boolean;
  stash_ids: Array<{ endpoint: string; stash_id: string }>;
  gallery_ids: string[];
  performers: SceneCopyPerformer[];
  performer_tag_ids: SceneCopyPerformerTags[];
  tag_names: string[];
  studio_name: string | null;
  groups: SceneCopyGroup[];
  video_filters: GQL.VideoFiltersInput | null;
  video_transforms: GQL.VideoTransformsInput | null;
  audio_offset_ms?: number | null;
  audio_playback_speed?: number | null;
  play_history: string[];
  o_history: string[];
  omg_history: string[];
}

function getScenePerformers(
  scene: GQL.SceneDataFragment
): SceneCopyPerformer[] {
  const scenePerformers = (
    scene as {
      scene_performers?: Array<{
        performer: { id: string; name: string };
        small_role?: boolean;
        role_description?: string | null;
      }>;
    }
  ).scene_performers;
  if (scenePerformers?.length) {
    return scenePerformers.map((sp) => ({
      name: sp.performer.name,
      small_role: sp.small_role ?? false,
      role_description: sp.role_description ?? null,
    }));
  }
  const performers = scene.performers ?? [];
  return performers.map((p) => ({
    name: p.name,
    small_role: Boolean((p as { small_role?: boolean }).small_role),
    role_description: null,
  }));
}

function getPerformerTagIds(
  scene: GQL.SceneDataFragment
): SceneCopyPerformerTags[] {
  const performerTagIds = scene.performer_tag_ids ?? [];
  const performers = scene.performers ?? [];
  const fromScenePerformers = (scene as { scene_performers?: Array<{ performer: { id: string; name: string } }> }).scene_performers;
  const performerById = new Map<string, string>();
  for (const p of performers) {
    performerById.set(p.id, p.name);
  }
  if (fromScenePerformers) {
    for (const sp of fromScenePerformers) {
      performerById.set(sp.performer.id, sp.performer.name);
    }
  }
  const tagsById = new Map<string, string>();
  for (const t of scene.tags ?? []) {
    tagsById.set(t.id, t.name);
  }
  return performerTagIds
    .filter(
      (pt): pt is { performer_id: string; tag_ids: string[] } =>
        !!pt.performer_id && !!pt.tag_ids
    )
    .map((pt) => {
      const performerName = performerById.get(pt.performer_id);
      if (!performerName) return null;
      const tagNames = (pt.tag_ids ?? [])
        .map((id) => tagsById.get(id))
        .filter((n): n is string => !!n);
      return { performer_name: performerName, tag_names: tagNames };
    })
    .filter((x): x is SceneCopyPerformerTags => x !== null);
}

export function serializeSceneCopyPayload(
  scene: GQL.SceneDataFragment
): string {
  const payload: SceneCopyPayload = {
    version: SCENE_COPY_VERSION,
    title: scene.title ?? undefined,
    code: scene.code ?? undefined,
    details: scene.details ?? undefined,
    director: scene.director ?? undefined,
    urls: scene.urls ?? [],
    date: scene.date ?? undefined,
    shoot_date: (scene as { shoot_date?: string }).shoot_date ?? undefined,
    rating100: scene.rating100 ?? undefined,
    organized: scene.organized ?? false,
    play_count: scene.play_count ?? undefined,
    start_time: scene.start_time ?? undefined,
    end_time: scene.end_time ?? undefined,
    is_broken: scene.is_broken ?? false,
    is_not_broken: scene.is_not_broken ?? false,
    disable_next_scene_overlay: scene.disable_next_scene_overlay ?? false,
    stash_ids: (scene.stash_ids ?? []).map((s) => ({
      endpoint: s.endpoint,
      stash_id: s.stash_id,
    })),
    gallery_ids: (scene.galleries ?? []).map((g) => g.id),
    performers: getScenePerformers(scene),
    performer_tag_ids: getPerformerTagIds(scene),
    tag_names: (scene.tags ?? []).map((t) => t.name),
    studio_name: scene.studio?.name ?? null,
    groups: (scene.groups ?? []).map((g) => ({
      name: g.group.name,
      scene_index: g.scene_index ?? null,
    })),
    video_filters: scene.video_filters
      ? {
          contrast: scene.video_filters.contrast,
          brightness: scene.video_filters.brightness,
          gamma: scene.video_filters.gamma,
          saturate: scene.video_filters.saturate,
          hue_rotate: scene.video_filters.hue_rotate,
          white_balance: scene.video_filters.white_balance,
          red: scene.video_filters.red,
          green: scene.video_filters.green,
          blue: scene.video_filters.blue,
          blur: scene.video_filters.blur,
          volume_level: scene.video_filters.volume_level,
          volume_muted: scene.video_filters.volume_muted,
          vr_projection: scene.video_filters.vr_projection ?? undefined,
        }
      : null,
    video_transforms: scene.video_transforms
      ? {
          rotate: scene.video_transforms.rotate,
          scale: scene.video_transforms.scale,
          aspect_ratio: scene.video_transforms.aspect_ratio,
          flip_horizontal: (scene.video_transforms as GQL.VideoTransforms).flip_horizontal,
          flip_vertical: (scene.video_transforms as GQL.VideoTransforms).flip_vertical,
        }
      : null,
    audio_offset_ms: scene.audio_offset_ms,
    audio_playback_speed: scene.audio_playback_speed,
    play_history: scene.play_history ?? [],
    o_history: scene.o_history ?? [],
    omg_history: scene.omg_history ?? [],
  };
  return SCENE_COPY_PREFIX + JSON.stringify(payload);
}

export function deserializeSceneCopyPayload(text: string): SceneCopyPayload | null {
  const trimmed = text.trim();
  if (!trimmed.startsWith(SCENE_COPY_PREFIX)) {
    return null;
  }
  try {
    const json = trimmed.slice(SCENE_COPY_PREFIX.length);
    const payload = JSON.parse(json) as SceneCopyPayload;
    if (typeof payload.version !== "number" || payload.version > SCENE_COPY_VERSION) {
      return null;
    }
    return payload;
  } catch {
    return null;
  }
}

export function isSceneCopyPayload(text: string): boolean {
  return text.trim().startsWith(SCENE_COPY_PREFIX);
}

export interface ScenePasteServices {
  updateScene: (opts: { variables: { input: GQL.SceneUpdateInput } }) => Promise<unknown>;
  createPerformer: (opts: { variables: { input: GQL.PerformerCreateInput } }) => Promise<{ data?: { performerCreate?: { id: string; name: string } } }>;
  createTag: (opts: { variables: { input: GQL.TagCreateInput } }) => Promise<{ data?: { tagCreate?: { id: string; name: string } } }>;
  createStudio: (opts: { variables: { input: GQL.StudioCreateInput } }) => Promise<{ data?: { studioCreate?: { id: string; name: string } } }>;
  createGroup: (opts: { variables: { input: GQL.GroupCreateInput } }) => Promise<{ data?: { groupCreate?: { id: string; name: string } } }>;
  addPlay: (opts: { variables: { id: string; times?: string[] } }) => Promise<unknown>;
  addO: (opts: { variables: { id: string; times?: string[] } }) => Promise<unknown>;
  addOmg: (opts: { variables: { id: string; times?: string[] } }) => Promise<unknown>;
}

async function findOrCreatePerformer(
  name: string,
  createPerformer: ScenePasteServices["createPerformer"]
): Promise<string> {
  const filter = new ListFilterModel(GQL.FilterMode.Performers);
  filter.searchTerm = name;
  filter.itemsPerPage = 100;
  const result = await queryFindPerformersForSelect(filter);
  const found = result.data.findPerformers.performers.find(
    (p) => p.name.toLowerCase() === name.toLowerCase()
  );
  if (found) return found.id;
  const created = await createPerformer({
    variables: { input: { name } },
  });
  if (created.data?.performerCreate?.id) {
    return created.data.performerCreate.id;
  }
  throw new Error(`Failed to create performer: ${name}`);
}

async function findOrCreateTag(
  name: string,
  createTag: ScenePasteServices["createTag"]
): Promise<string> {
  const filter = new ListFilterModel(GQL.FilterMode.Tags);
  filter.searchTerm = name;
  filter.itemsPerPage = 100;
  const result = await queryFindTagsForSelect(filter);
  const found = result.data.findTags.tags.find(
    (t) => t.name.toLowerCase() === name.toLowerCase()
  );
  if (found) return found.id;
  const created = await createTag({
    variables: { input: { name } },
  });
  if (created.data?.tagCreate?.id) {
    return created.data.tagCreate.id;
  }
  throw new Error(`Failed to create tag: ${name}`);
}

async function findOrCreateStudio(
  name: string,
  createStudio: ScenePasteServices["createStudio"]
): Promise<string | null> {
  if (!name.trim()) return null;
  const filter = new ListFilterModel(GQL.FilterMode.Studios);
  filter.searchTerm = name;
  filter.itemsPerPage = 100;
  const result = await queryFindStudiosForSelect(filter);
  const found = result.data.findStudios.studios.find(
    (s) => s.name.toLowerCase() === name.toLowerCase()
  );
  if (found) return found.id;
  const created = await createStudio({
    variables: { input: { name } },
  });
  if (created.data?.studioCreate?.id) {
    return created.data.studioCreate.id;
  }
  throw new Error(`Failed to create studio: ${name}`);
}

async function findOrCreateGroup(
  name: string,
  createGroup: ScenePasteServices["createGroup"]
): Promise<string> {
  const filter = new ListFilterModel(GQL.FilterMode.Groups);
  filter.searchTerm = name;
  filter.itemsPerPage = 100;
  const result = await queryFindGroupsForSelect(filter);
  const found = result.data.findGroups.groups.find(
    (g) => g.name.toLowerCase() === name.toLowerCase()
  );
  if (found) return found.id;
  const created = await createGroup({
    variables: { input: { name } },
  });
  if (created.data?.groupCreate?.id) {
    return created.data.groupCreate.id;
  }
  throw new Error(`Failed to create group: ${name}`);
}

export async function applySceneCopyPayload(
  targetSceneId: string,
  payload: SceneCopyPayload,
  services: ScenePasteServices
): Promise<void> {
  const performerIds = new Map<string, string>();
  for (const p of payload.performers) {
    const id = await findOrCreatePerformer(p.name, services.createPerformer);
    performerIds.set(p.name.toLowerCase(), id);
  }

  const tagIds = new Map<string, string>();
  for (const name of payload.tag_names) {
    const id = await findOrCreateTag(name, services.createTag);
    tagIds.set(name.toLowerCase(), id);
  }

  const studioId = payload.studio_name
    ? await findOrCreateStudio(payload.studio_name, services.createStudio)
    : null;

  const groupIds = new Map<string, string>();
  for (const g of payload.groups) {
    const id = await findOrCreateGroup(g.name, services.createGroup);
    groupIds.set(g.name.toLowerCase(), id);
  }

  const scene_performers: GQL.ScenePerformerInput[] = payload.performers.map(
    (p) => {
      const performerId = performerIds.get(p.name.toLowerCase());
      if (!performerId) throw new Error(`Performer not resolved: ${p.name}`);
      return {
        performer_id: performerId,
        small_role: p.small_role,
        role_description: p.role_description,
      };
    }
  );

  const performer_tag_ids: GQL.PerformerTagInput[] = [];
  for (const pt of payload.performer_tag_ids) {
    const performerId = performerIds.get(pt.performer_name.toLowerCase());
    if (!performerId) continue;
    const ids: string[] = [];
    for (const tagName of pt.tag_names) {
      const tagId = tagIds.get(tagName.toLowerCase());
      if (tagId) ids.push(tagId);
    }
    if (ids.length > 0) {
      performer_tag_ids.push({ performer_id: performerId, tag_ids: ids });
    }
  }

  const tag_ids = payload.tag_names
    .map((n) => tagIds.get(n.toLowerCase()))
    .filter((id): id is string => !!id);

  const groups: GQL.SceneGroupInput[] = payload.groups.map((g) => {
    const groupId = groupIds.get(g.name.toLowerCase());
    if (!groupId) throw new Error(`Group not resolved: ${g.name}`);
    return { group_id: groupId, scene_index: g.scene_index };
  });

  const input: GQL.SceneUpdateInput = {
    id: targetSceneId,
    title: payload.title ?? undefined,
    code: payload.code ?? undefined,
    details: payload.details ?? undefined,
    director: payload.director ?? undefined,
    urls: payload.urls,
    date: payload.date ?? undefined,
    shoot_date: payload.shoot_date ?? undefined,
    rating100: payload.rating100 ?? undefined,
    organized: payload.organized,
    start_time: payload.start_time ?? undefined,
    end_time: payload.end_time ?? undefined,
    is_broken: payload.is_broken,
    is_not_broken: payload.is_not_broken,
    disable_next_scene_overlay: payload.disable_next_scene_overlay,
    stash_ids: payload.stash_ids.map((s) => ({
      ...s,
      updated_at: new Date().toISOString(),
    })),
    gallery_ids: payload.gallery_ids,
    performer_ids: scene_performers.map((sp) => sp.performer_id),
    scene_performers,
    performer_tag_ids,
    tag_ids,
    studio_id: studioId,
    groups,
    video_filters: payload.video_filters ?? undefined,
    video_transforms: payload.video_transforms ?? undefined,
    audio_offset_ms: payload.audio_offset_ms ?? undefined,
    audio_playback_speed: payload.audio_playback_speed ?? undefined,
  };

  await services.updateScene({ variables: { input } });

  if (payload.play_history?.length) {
    await services.addPlay({
      variables: { id: targetSceneId, times: payload.play_history },
    });
  }
  if (payload.o_history?.length) {
    await services.addO({
      variables: { id: targetSceneId, times: payload.o_history },
    });
  }
  if (payload.omg_history?.length) {
    await services.addOmg({
      variables: { id: targetSceneId, times: payload.omg_history },
    });
  }
}
