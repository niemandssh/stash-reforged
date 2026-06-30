import * as GQL from "src/core/generated-graphql";
import { ListFilterModel } from "src/models/list-filter/filter";
import {
  getClient,
  queryFindPerformersForSelect,
  queryFindTagsForSelect,
  queryFindStudiosForSelect,
} from "src/core/StashService";

type AIFillResult = NonNullable<
  GQL.FillSceneWithAiQuery["fillSceneWithAI"]
>;

type CreatePerformerFn = (opts: {
  variables: { input: GQL.PerformerCreateInput };
}) => Promise<{ data?: { performerCreate?: { id: string } | null } | null }>;

type CreateTagFn = (opts: {
  variables: { input: GQL.TagCreateInput };
}) => Promise<{ data?: { tagCreate?: { id: string } | null } | null }>;

type CreateStudioFn = (opts: {
  variables: { input: GQL.StudioCreateInput };
}) => Promise<{ data?: { studioCreate?: { id: string } | null } | null }>;

type UpdateSceneFn = (opts: {
  variables: { input: GQL.SceneUpdateInput };
}) => Promise<unknown>;

async function findOrCreatePerformer(
  name: string,
  createPerformer: CreatePerformerFn
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
  createTag: CreateTagFn
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
  createStudio: CreateStudioFn
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

export async function applyAIFilledSceneWithServices(
  sceneId: string,
  scraped: GQL.ScrapedSceneDataFragment,
  aiResult: AIFillResult,
  services: {
    updateScene: UpdateSceneFn;
    createPerformer: CreatePerformerFn;
    createTag: CreateTagFn;
    createStudio: CreateStudioFn;
  }
): Promise<void> {
  const performerIds = new Map<string, string>();

  if (scraped.performers) {
    for (const p of scraped.performers) {
      if (p.stored_id) {
        performerIds.set((p.name ?? "").toLowerCase(), p.stored_id);
      } else if (p.name) {
        const id = await findOrCreatePerformer(p.name, services.createPerformer);
        performerIds.set(p.name.toLowerCase(), id);
      }
    }
  }

  const tagIds = new Map<string, string>();
  if (scraped.tags) {
    for (const t of scraped.tags) {
      if (t.stored_id) {
        tagIds.set(t.name.toLowerCase(), t.stored_id);
      } else {
        const id = await findOrCreateTag(t.name, services.createTag);
        tagIds.set(t.name.toLowerCase(), id);
      }
    }
  }

  for (const pt of aiResult.performer_tags ?? []) {
    for (const tagName of pt.tag_names) {
      if (!tagIds.has(tagName.toLowerCase())) {
        const id = await findOrCreateTag(tagName, services.createTag);
        tagIds.set(tagName.toLowerCase(), id);
      }
    }
  }

  let studioId: string | null | undefined;
  if (scraped.studio?.stored_id) {
    studioId = scraped.studio.stored_id;
  } else if (scraped.studio?.name) {
    studioId = await findOrCreateStudio(
      scraped.studio.name,
      services.createStudio
    );
  }

  const scene_performers: GQL.ScenePerformerInput[] = [];
  if (scraped.performers) {
    for (const p of scraped.performers) {
      const id =
        p.stored_id ?? performerIds.get((p.name ?? "").toLowerCase());
      if (id) {
        scene_performers.push({
          performer_id: id,
          small_role: false,
        });
      }
    }
  }

  const performer_tag_ids: GQL.PerformerTagInput[] = [];
  for (const pt of aiResult.performer_tags ?? []) {
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

  const tag_ids = scraped.tags
    ?.map((t) => t.stored_id ?? tagIds.get(t.name.toLowerCase()))
    .filter((id): id is string => !!id);

  const input: GQL.SceneUpdateInput = {
    id: sceneId,
  };

  if (scraped.title) input.title = scraped.title;
  if (scraped.code) input.code = scraped.code;
  if (scraped.details) input.details = scraped.details;
  if (scraped.director) input.director = scraped.director;
  if (scraped.date) input.date = scraped.date;
  if (aiResult.shoot_date) input.shoot_date = aiResult.shoot_date;
  if (scraped.urls) input.urls = scraped.urls;
  if (studioId) input.studio_id = studioId;
  if (scene_performers.length > 0) input.scene_performers = scene_performers;
  if (tag_ids && tag_ids.length > 0) input.tag_ids = tag_ids;
  if (performer_tag_ids.length > 0) input.performer_tag_ids = performer_tag_ids;

  input.is_ai_filled = true;

  await services.updateScene({ variables: { input } });
}

export async function applyAIFilledScene(
  sceneId: string,
  scraped: GQL.ScrapedSceneDataFragment,
  aiResult: AIFillResult
): Promise<void> {
  const client = getClient();

  await applyAIFilledSceneWithServices(sceneId, scraped, aiResult, {
    updateScene: (opts) =>
      client.mutate<GQL.SceneUpdateMutation>({
        mutation: GQL.SceneUpdateDocument,
        variables: opts.variables,
      }) as Promise<unknown>,
    createPerformer: (opts) =>
      client.mutate<GQL.PerformerCreateMutation>({
        mutation: GQL.PerformerCreateDocument,
        variables: opts.variables,
      }),
    createTag: (opts) =>
      client.mutate<GQL.TagCreateMutation>({
        mutation: GQL.TagCreateDocument,
        variables: opts.variables,
      }),
    createStudio: (opts) =>
      client.mutate<GQL.StudioCreateMutation>({
        mutation: GQL.StudioCreateDocument,
        variables: opts.variables,
      }),
  });
}
