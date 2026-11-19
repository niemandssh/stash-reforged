import { useToast } from "src/hooks/Toast";
import * as GQL from "src/core/generated-graphql";
import {
  useGroupCreate,
  usePerformerCreate,
  useStudioCreate,
  useTagCreate,
} from "src/core/StashService";
import { ObjectScrapeResult, ScrapeResult } from "./scrapeResult";
import { useIntl } from "react-intl";
import { scrapedPerformerToCreateInput } from "src/core/performers";
import { scrapedGroupToCreateInput } from "src/core/groups";

function useCreateObject<T>(
  entityTypeID: string,
  createFunc: (o: T) => Promise<void>
) {
  const Toast = useToast();
  const intl = useIntl();

  async function createNewObject(o: T) {
    try {
      await createFunc(o);

      Toast.success(
        intl.formatMessage(
          { id: "toast.created_entity" },
          {
            entity: intl
              .formatMessage({ id: entityTypeID })
              .toLocaleLowerCase(),
          }
        )
      );
    } catch (e) {
      Toast.error(e);
    }
  }

  return createNewObject;
}

interface IUseCreateNewStudioProps {
  scrapeResult: ObjectScrapeResult<GQL.ScrapedStudio>;
  setScrapeResult: (
    scrapeResult: ObjectScrapeResult<GQL.ScrapedStudio>
  ) => void;
  setNewObject: (newObject: GQL.ScrapedStudio | undefined) => void;
}

export function useCreateScrapedStudio(props: IUseCreateNewStudioProps) {
  const [createStudio] = useStudioCreate();

  const { scrapeResult, setScrapeResult, setNewObject } = props;

  async function createNewStudio(toCreate: GQL.ScrapedStudio) {
    const result = await createStudio({
      variables: {
        input: {
          name: toCreate.name,
          url: toCreate.url,
          parent_id: toCreate.parent?.stored_id,
        },
      },
    });

    // set the new studio as the value
    setScrapeResult(
      scrapeResult.cloneWithValue({
        stored_id: result.data!.studioCreate!.id,
        name: toCreate.name,
      })
    );
    setNewObject(undefined);
  }

  return useCreateObject("studio", createNewStudio);
}

interface IUseCreateNewObjectProps<T> {
  scrapeResult: ScrapeResult<T[]>;
  setScrapeResult: (scrapeResult: ScrapeResult<T[]>) => void;
  newObjects: T[];
  setNewObjects: (newObject: T[]) => void;
}

export function useCreateScrapedPerformer(
  props: IUseCreateNewObjectProps<GQL.ScrapedPerformer>
) {
  const [createPerformer] = usePerformerCreate();

  const { scrapeResult, setScrapeResult, newObjects, setNewObjects } = props;

  async function createNewPerformer(toCreate: GQL.ScrapedPerformer) {
    const input = scrapedPerformerToCreateInput(toCreate);

    const result = await createPerformer({
      variables: { input },
    });

    const newValue = [...(scrapeResult.newValue ?? [])];
    if (result.data?.performerCreate)
      newValue.push({
        stored_id: result.data.performerCreate.id,
        name: result.data.performerCreate.name,
      });

    // add the new performer to the new performers value
    const performerClone = scrapeResult.cloneWithValue(newValue);
    setScrapeResult(performerClone);

    // remove the performer from the list
    const newPerformersClone = newObjects.concat();
    const pIndex = newPerformersClone.findIndex(
      (p) => p.name === toCreate.name
    );
    if (pIndex === -1) throw new Error("Could not find performer to remove");

    newPerformersClone.splice(pIndex, 1);

    setNewObjects(newPerformersClone);
  }

  return useCreateObject("performer", createNewPerformer);
}

export function useCreateScrapedGroup(
  props: IUseCreateNewObjectProps<GQL.ScrapedGroup>
) {
  const { scrapeResult, setScrapeResult, newObjects, setNewObjects } = props;
  const [createGroup] = useGroupCreate();

  async function createNewGroup(toCreate: GQL.ScrapedGroup) {
    const input = scrapedGroupToCreateInput(toCreate);

    const result = await createGroup({
      variables: { input: input },
    });

    const newValue = [...(scrapeResult.newValue ?? [])];
    if (result.data?.groupCreate)
      newValue.push({
        stored_id: result.data.groupCreate.id,
        name: result.data.groupCreate.name,
      });

    // add the new object to the new object value
    const resultClone = scrapeResult.cloneWithValue(newValue);
    setScrapeResult(resultClone);

    // remove the object from the list
    const newObjectsClone = newObjects.concat();
    const pIndex = newObjectsClone.findIndex((p) => p.name === toCreate.name);
    if (pIndex === -1) throw new Error("Could not find group to remove");

    newObjectsClone.splice(pIndex, 1);

    setNewObjects(newObjectsClone);
  }

  return useCreateObject("group", createNewGroup);
}

export function useCreateScrapedTag(
  props: IUseCreateNewObjectProps<GQL.ScrapedTag> & {
    onTagCreated?: (tag: GQL.TagDataFragment) => void;
  }
) {
  const [createTag] = useTagCreate();

  const {
    scrapeResult,
    setScrapeResult,
    newObjects,
    setNewObjects,
    onTagCreated,
  } = props;

  async function createNewTags(toCreate: GQL.ScrapedTag[]) {
    const newValue = [...(scrapeResult.newValue ?? [])];
    const created = new Set<string>();

    try {
      for (const tag of toCreate) {
        const result = await createTag({
          variables: { input: { name: tag.name ?? "" } },
        });

        if (result.data?.tagCreate) {
          onTagCreated?.(result.data.tagCreate);
          newValue.push({
            stored_id: result.data.tagCreate.id,
            name: result.data.tagCreate.name,
          });
        }
        created.add(tag.name);
      }
    } finally {
      setScrapeResult(scrapeResult.cloneWithValue(newValue));
      setNewObjects(newObjects.filter((t) => !created.has(t.name)));
    }
  }

  return useCreateObject("tag", createNewTags);
}
