import React, { useEffect, useState, useMemo } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Button, Form, Col, Row, ButtonGroup } from "react-bootstrap";
import Mousetrap from "mousetrap";
import * as GQL from "src/core/generated-graphql";
import * as yup from "yup";
import {
  queryScrapeScene,
  queryScrapeSceneURL,
  useListSceneScrapers,
  mutateReloadScrapers,
  queryScrapeSceneQueryFragment,
  queryFindTags,
} from "src/core/StashService";
import { Icon } from "src/components/Shared/Icon";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { ImageInput } from "src/components/Shared/ImageInput";
import { DurationInput } from "src/components/Shared/DurationInput";
import { getPlayerPosition } from "src/components/ScenePlayer/util";
import { useToast } from "src/hooks/Toast";
import { useTrimContext } from "src/contexts/TrimContext";
import ImageUtils from "src/utils/image";
import { getStashIDs } from "src/utils/stashIds";
import { useFormik } from "formik";
import { Prompt } from "react-router-dom";
import { ConfigurationContext } from "src/hooks/Config";
import { ListFilterModel } from "src/models/list-filter/filter";
import { IGroupEntry, SceneGroupTable } from "./SceneGroupTable";
import { faSearch } from "@fortawesome/free-solid-svg-icons";
import { objectTitle } from "src/core/files";
import { galleryTitle } from "src/core/galleries";
import { lazyComponent } from "src/utils/lazyComponent";
import isEqual from "lodash-es/isEqual";
import {
  yupDateString,
  yupFormikValidate,
  yupUniqueStringList,
} from "src/utils/yup";
import {
  Performer,
  PerformerSelect,
} from "src/components/Performers/PerformerSelect";
import { formikUtils } from "src/utils/form";
import { Studio, StudioSelect } from "src/components/Studios/StudioSelect";
import { Gallery, GallerySelect } from "src/components/Galleries/GallerySelect";
import { Group } from "src/components/Groups/GroupSelect";
import { useTagsEdit } from "src/hooks/tagsEdit";
import { ScraperMenu } from "src/components/Shared/ScraperMenu";
import { PoseTagSelector } from "src/components/Shared/PoseTagSelector";

const SceneScrapeDialog = lazyComponent(() => import("./SceneScrapeDialog"));
const SceneQueryModal = lazyComponent(() => import("./SceneQueryModal"));

interface IProps {
  scene: Partial<GQL.SceneDataFragment>;
  initialCoverImage?: string;
  isNew?: boolean;
  isVisible: boolean;
  onSubmit: (input: GQL.SceneCreateInput) => Promise<void>;
  onDelete?: () => void;
}

export const SceneEditPanel: React.FC<IProps> = ({
  scene,
  initialCoverImage,
  isNew = false,
  isVisible,
  onSubmit,
  onDelete,
}) => {
  const intl = useIntl();
  const Toast = useToast();

  const [galleries, setGalleries] = useState<Gallery[]>([]);
  const [performers, setPerformers] = useState<Performer[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [studio, setStudio] = useState<Studio | null>(null);

  const Scrapers = useListSceneScrapers();
  const [fragmentScrapers, setFragmentScrapers] = useState<GQL.Scraper[]>([]);
  const [queryableScrapers, setQueryableScrapers] = useState<GQL.Scraper[]>([]);

  const [scraper, setScraper] = useState<GQL.ScraperSourceInput>();
  const [isScraperQueryModalOpen, setIsScraperQueryModalOpen] =
    useState<boolean>(false);
  const [scrapedScene, setScrapedScene] = useState<GQL.ScrapedScene | null>();
  const [endpoint, setEndpoint] = useState<string>();
  const [selectedPoseTagIds, setSelectedPoseTagIds] = useState<string[]>([]);

  useEffect(() => {
    setGalleries(
      scene.galleries?.map((g) => ({
        id: g.id,
        title: galleryTitle(g),
        files: g.files,
        folder: g.folder,
      })) ?? []
    );
  }, [scene.galleries]);

  useEffect(() => {
    setPerformers(scene.performers ?? []);
  }, [scene.performers]);

  useEffect(() => {
    setGroups(scene.groups?.map((m) => m.group) ?? []);
  }, [scene.groups]);

  useEffect(() => {
    setStudio(scene.studio ?? null);
  }, [scene.studio]);

  const { configuration: stashConfig } = React.useContext(ConfigurationContext);

  // Network state
  const [isLoading, setIsLoading] = useState(false);
  const { trimEnabled, setTrimEnabled } = useTrimContext();

  const schema = yup.object({
    title: yup.string().ensure(),
    code: yup.string().ensure(),
    urls: yupUniqueStringList(intl),
    date: yupDateString(intl),
    director: yup.string().ensure(),
    gallery_ids: yup.array(yup.string().required()).defined(),
    studio_id: yup.string().required().nullable(),
    performer_ids: yup.array(yup.string().required()).defined(),
    groups: yup
      .array(
        yup.object({
          group_id: yup.string().required(),
          scene_index: yup.number().integer().nullable().defined(),
        })
      )
      .defined(),
    tag_ids: yup.array(yup.string().required()).defined(),
    stash_ids: yup.mixed<GQL.StashIdInput[]>().defined(),
    details: yup.string().ensure(),
    cover_image: yup.string().nullable().optional(),
    is_broken: yup.boolean().defined(),
    start_time: yup.number().nullable().optional(),
    end_time: yup.number().nullable().optional(),
  });

  const initialValues = useMemo(
    () => ({
      title: scene.title ?? "",
      code: scene.code ?? "",
      urls: scene.urls ?? [],
      date: scene.date ?? "",
      director: scene.director ?? "",
      gallery_ids: (scene.galleries ?? []).map((g) => g.id),
      studio_id: scene.studio?.id ?? null,
      performer_ids: (scene.performers ?? []).map((p) => p.id),
      groups: (scene.groups ?? []).map((m) => {
        return { group_id: m.group.id, scene_index: m.scene_index ?? null };
      }),
      tag_ids: (scene.tags ?? []).map((t) => t.id),
      stash_ids: getStashIDs(scene.stash_ids),
      details: scene.details ?? "",
      cover_image: initialCoverImage,
      is_broken: scene.is_broken ?? false,
      start_time: scene.start_time ?? null,
      end_time: scene.end_time ?? null,
    }),
    [scene, initialCoverImage]
  );

  type InputValues = yup.InferType<typeof schema>;

  const formik = useFormik<InputValues>({
    initialValues,
    enableReinitialize: true,
    validate: yupFormikValidate(schema),
    onSubmit: (values) => onSave(schema.cast(values)),
  });

  const { tags, updateTagsStateFromScraper, tagsControl, onSetTags } = useTagsEdit(
    scene.tags,
    (ids) => formik.setFieldValue("tag_ids", ids)
  );

  const [allTags, setAllTags] = useState<GQL.Tag[]>([]);
  
  useEffect(() => {
    const loadAllTags = async () => {
      try {
        const filter = new ListFilterModel(GQL.FilterMode.Tags);
        filter.itemsPerPage = -1;
        filter.sortBy = "name";
        filter.sortDirection = GQL.SortDirectionEnum.Asc;
        
        const result = await queryFindTags(filter);
        setAllTags(result.data.findTags.tags as unknown as GQL.Tag[]);
      } catch (error) {
        console.error("Error loading all tags:", error);
      }
    };
    
    loadAllTags();
  }, []);

  useEffect(() => {
    if (scene.tags) {
      const poseTagIds = scene.tags
        .filter(tag => tag.is_pose_tag)
        .map(tag => tag.id);
      setSelectedPoseTagIds(poseTagIds);
    }
  }, [scene.tags]);

  useEffect(() => {
    if (allTags.length > 0) {
      const currentTagIds = formik.values.tag_ids || [];
      const poseTagIdsFromTags = currentTagIds.filter(tagId => {
        const tag = allTags.find(t => t.id === tagId);
        return tag && tag.is_pose_tag;
      });
      
      if (!isEqual(poseTagIdsFromTags.sort(), selectedPoseTagIds.sort())) {
        setSelectedPoseTagIds(poseTagIdsFromTags);
      }
    }
  }, [formik.values.tag_ids, allTags]);

  const coverImagePreview = useMemo(() => {
    const sceneImage = scene.paths?.screenshot;
    const formImage = formik.values.cover_image;
    if (formImage === null && sceneImage) {
      const sceneImageURL = new URL(sceneImage);
      sceneImageURL.searchParams.set("default", "true");
      return sceneImageURL.toString();
    } else if (formImage) {
      return formImage;
    }
    return sceneImage;
  }, [formik.values.cover_image, scene.paths?.screenshot]);

  const groupEntries = useMemo(() => {
    return formik.values.groups
      .map((m) => {
        return {
          group: groups.find((mm) => mm.id === m.group_id),
          scene_index: m.scene_index,
        };
      })
      .filter((m) => m.group !== undefined) as IGroupEntry[];
  }, [formik.values.groups, groups]);

  function onSetGalleries(items: Gallery[]) {
    setGalleries(items);
    formik.setFieldValue(
      "gallery_ids",
      items.map((i) => i.id)
    );
  }

  function onSetPerformers(items: Performer[]) {
    setPerformers(items);
    formik.setFieldValue(
      "performer_ids",
      items.map((item) => item.id)
    );
  }

  function onSetStudio(item: Studio | null) {
    setStudio(item);
    formik.setFieldValue("studio_id", item ? item.id : null);
  }

  function onPoseTagSelectionChange(poseTagIds: string[]) {
    setSelectedPoseTagIds(poseTagIds);
    
    const currentTagIds = formik.values.tag_ids || [];
    
    const nonPoseTagIds = currentTagIds.filter(tagId => {
      const tag = allTags.find(t => t.id === tagId);
      return !tag || !tag.is_pose_tag;
    });
    
    const newTagIds = [...nonPoseTagIds, ...poseTagIds];
    
    if (allTags.length > 0) {
      const newTags = newTagIds.map(id => allTags.find(t => t.id === id)).filter(Boolean) as typeof tags;
      
      onSetTags(newTags);
    } else {
      formik.setFieldValue("tag_ids", newTagIds);
    }
  }

  useEffect(() => {
    if (isVisible) {
      Mousetrap.bind("s s", () => {
        if (formik.dirty) {
          formik.submitForm();
        }
      });
      Mousetrap.bind("d d", () => {
        if (onDelete) {
          onDelete();
        }
      });

      return () => {
        Mousetrap.unbind("s s");
        Mousetrap.unbind("d d");
      };
    }
  });

  useEffect(() => {
    const toFilter = Scrapers?.data?.listScrapers ?? [];

    const newFragmentScrapers = toFilter.filter((s) =>
      s.scene?.supported_scrapes.includes(GQL.ScrapeType.Fragment)
    );
    const newQueryableScrapers = toFilter.filter((s) =>
      s.scene?.supported_scrapes.includes(GQL.ScrapeType.Name)
    );

    setFragmentScrapers(newFragmentScrapers);
    setQueryableScrapers(newQueryableScrapers);
  }, [Scrapers, stashConfig]);

  function onSetGroups(items: Group[]) {
    setGroups(items);

    const existingGroups = formik.values.groups;

    const newGroups = items.map((m) => {
      const existing = existingGroups.find((mm) => mm.group_id === m.id);
      if (existing) {
        return existing;
      }

      return {
        group_id: m.id,
        scene_index: null,
      };
    });

    formik.setFieldValue("groups", newGroups);
  }

  async function onSave(input: InputValues) {
    setIsLoading(true);
    try {
      await onSubmit(input);
      formik.resetForm();
    } catch (e) {
      Toast.error(e);
    }
    setIsLoading(false);
  }

  const encodingImage = ImageUtils.usePasteImage(onImageLoad);

  function onImageLoad(imageData: string) {
    formik.setFieldValue("cover_image", imageData);
  }

  function onCoverImageChange(event: React.FormEvent<HTMLInputElement>) {
    ImageUtils.onImageChange(event, onImageLoad);
  }

  async function onScrapeClicked(s: GQL.ScraperSourceInput) {
    setIsLoading(true);
    try {
      const result = await queryScrapeScene(s, scene.id!);
      if (!result.data || !result.data.scrapeSingleScene?.length) {
        Toast.success("No scenes found");
        return;
      }
      // assume one returned scene
      setScrapedScene(result.data.scrapeSingleScene[0]);
      setEndpoint(s.stash_box_endpoint ?? undefined);
    } catch (e) {
      Toast.error(e);
    } finally {
      setIsLoading(false);
    }
  }

  async function scrapeFromQuery(
    s: GQL.ScraperSourceInput,
    fragment: GQL.ScrapedSceneDataFragment
  ) {
    setIsLoading(true);
    try {
      const input: GQL.ScrapedSceneInput = {
        date: fragment.date,
        code: fragment.code,
        details: fragment.details,
        director: fragment.director,
        remote_site_id: fragment.remote_site_id,
        title: fragment.title,
        urls: fragment.urls,
      };

      const result = await queryScrapeSceneQueryFragment(s, input);
      if (!result.data || !result.data.scrapeSingleScene?.length) {
        Toast.success("No scenes found");
        return;
      }
      // assume one returned scene
      setScrapedScene(result.data.scrapeSingleScene[0]);
    } catch (e) {
      Toast.error(e);
    } finally {
      setIsLoading(false);
    }
  }

  function onScrapeQueryClicked(s: GQL.ScraperSourceInput) {
    setScraper(s);
    setEndpoint(s.stash_box_endpoint ?? undefined);
    setIsScraperQueryModalOpen(true);
  }

  async function onReloadScrapers() {
    setIsLoading(true);
    try {
      await mutateReloadScrapers();
    } catch (e) {
      Toast.error(e);
    } finally {
      setIsLoading(false);
    }
  }

  function onScrapeDialogClosed(sceneData?: GQL.ScrapedSceneDataFragment) {
    if (sceneData) {
      updateSceneFromScrapedScene(sceneData);
    }
    setScrapedScene(undefined);
  }

  function maybeRenderScrapeDialog() {
    if (!scrapedScene) {
      return;
    }

    const currentScene = {
      id: scene.id!,
      ...formik.values,
    };

    if (!currentScene.cover_image) {
      currentScene.cover_image = scene.paths?.screenshot;
    }

    return (
      <SceneScrapeDialog
        scene={currentScene}
        sceneStudio={studio}
        sceneTags={tags}
        scenePerformers={performers}
        sceneGroups={groups}
        scraped={scrapedScene}
        endpoint={endpoint}
        onClose={(s) => onScrapeDialogClosed(s)}
      />
    );
  }

  function onSceneSelected(s: GQL.ScrapedSceneDataFragment) {
    if (!scraper) return;

    if (scraper?.stash_box_endpoint !== undefined) {
      // must be stash-box - assume full scene
      setScrapedScene(s);
    } else {
      // must be scraper
      scrapeFromQuery(scraper, s);
    }
  }

  const renderScrapeQueryModal = () => {
    if (!isScraperQueryModalOpen || !scraper) return;

    return (
      <SceneQueryModal
        scraper={scraper}
        onHide={() => setScraper(undefined)}
        onSelectScene={(s) => {
          setIsScraperQueryModalOpen(false);
          setScraper(undefined);
          onSceneSelected(s);
        }}
        name={formik.values.title || objectTitle(scene) || ""}
      />
    );
  };

  function urlScrapable(scrapedUrl: string): boolean {
    return (Scrapers?.data?.listScrapers ?? []).some((s) =>
      (s?.scene?.urls ?? []).some((u) => scrapedUrl.includes(u))
    );
  }

  function updateSceneFromScrapedScene(
    updatedScene: GQL.ScrapedSceneDataFragment
  ) {
    if (updatedScene.title) {
      formik.setFieldValue("title", updatedScene.title);
    }

    if (updatedScene.code) {
      formik.setFieldValue("code", updatedScene.code);
    }

    if (updatedScene.details) {
      formik.setFieldValue("details", updatedScene.details);
    }

    if (updatedScene.director) {
      formik.setFieldValue("director", updatedScene.director);
    }

    if (updatedScene.date) {
      formik.setFieldValue("date", updatedScene.date);
    }

    if (updatedScene.urls) {
      formik.setFieldValue("urls", updatedScene.urls);
    }

    if (updatedScene.studio && updatedScene.studio.stored_id) {
      onSetStudio({
        id: updatedScene.studio.stored_id,
        name: updatedScene.studio.name ?? "",
        aliases: [],
      });
    }

    if (updatedScene.performers && updatedScene.performers.length > 0) {
      const idPerfs = updatedScene.performers.filter((p) => {
        return p.stored_id !== undefined && p.stored_id !== null;
      });

      if (idPerfs.length > 0) {
        onSetPerformers(
          idPerfs.map((p) => {
            return {
              id: p.stored_id!,
              name: p.name ?? "",
              alias_list: [],
            };
          })
        );
      }
    }

    if (updatedScene.groups && updatedScene.groups.length > 0) {
      const idMovis = updatedScene.groups.filter((p) => {
        return p.stored_id !== undefined && p.stored_id !== null;
      });

      if (idMovis.length > 0) {
        onSetGroups(
          idMovis.map((p) => {
            return {
              id: p.stored_id!,
              name: p.name ?? "",
            };
          })
        );
      }
    }

    updateTagsStateFromScraper(updatedScene.tags ?? undefined);

    if (updatedScene.image) {
      // image is a base64 string
      formik.setFieldValue("cover_image", updatedScene.image);
    }

    if (updatedScene.remote_site_id && endpoint) {
      let found = false;
      formik.setFieldValue(
        "stash_ids",
        formik.values.stash_ids.map((s) => {
          if (s.endpoint === endpoint) {
            found = true;
            return {
              endpoint,
              stash_id: updatedScene.remote_site_id,
              updated_at: new Date().toISOString(),
            };
          }

          return s;
        })
      );

      if (!found) {
        formik.setFieldValue(
          "stash_ids",
          formik.values.stash_ids.concat({
            endpoint,
            stash_id: updatedScene.remote_site_id,
            updated_at: new Date().toISOString(),
          })
        );
      }
    }
  }

  async function onScrapeSceneURL(url: string) {
    if (!url) {
      return;
    }
    setIsLoading(true);
    try {
      const result = await queryScrapeSceneURL(url);
      if (!result.data || !result.data.scrapeSceneURL) {
        return;
      }
      setScrapedScene(result.data.scrapeSceneURL);
    } catch (e) {
      Toast.error(e);
    } finally {
      setIsLoading(false);
    }
  }

  const image = useMemo(() => {
    if (encodingImage) {
      return (
        <LoadingIndicator
          message={intl.formatMessage({ id: "actions.encoding_image" })}
        />
      );
    }

    if (coverImagePreview) {
      return (
        <img
          className="scene-cover"
          src={coverImagePreview}
          alt={intl.formatMessage({ id: "cover_image" })}
        />
      );
    }

    return <div></div>;
  }, [encodingImage, coverImagePreview, intl]);

  if (isLoading) return <LoadingIndicator />;

  const splitProps = {
    labelProps: {
      column: true,
      sm: 3,
    },
    fieldProps: {
      sm: 9,
    },
  };
  const fullWidthProps = {
    labelProps: {
      column: true,
      sm: 3,
      xl: 12,
    },
    fieldProps: {
      sm: 9,
      xl: 12,
    },
  };
  const {
    renderField,
    renderInputField,
    renderDateField,
    renderURLListField,
    renderStashIDsField,
  } = formikUtils(intl, formik, splitProps);

  function renderGalleriesField() {
    const title = intl.formatMessage({ id: "galleries" });
    const control = (
      <GallerySelect
        values={galleries}
        onSelect={(items) => onSetGalleries(items)}
        isMulti
      />
    );

    return renderField("gallery_ids", title, control);
  }

  function renderStudioField() {
    const title = intl.formatMessage({ id: "studio" });
    const control = (
      <StudioSelect
        onSelect={(items) => onSetStudio(items.length > 0 ? items[0] : null)}
        values={studio ? [studio] : []}
      />
    );

    return renderField("studio_id", title, control);
  }

  function renderPerformersField() {
    const date = (() => {
      try {
        return schema.validateSyncAt("date", formik.values);
      } catch (e) {
        return undefined;
      }
    })();

    const title = intl.formatMessage({ id: "performers" });
    const control = (
      <PerformerSelect
        isMulti
        onSelect={onSetPerformers}
        values={performers}
        ageFromDate={date}
      />
    );

    return renderField("performer_ids", title, control, fullWidthProps);
  }

  function onSetGroupEntries(input: IGroupEntry[]) {
    setGroups(input.map((m) => m.group));

    const newGroups = input.map((m) => ({
      group_id: m.group.id,
      scene_index: m.scene_index,
    }));

    formik.setFieldValue("groups", newGroups);
  }

  function renderGroupsField() {
    const title = intl.formatMessage({ id: "groups" });
    const control = (
      <SceneGroupTable value={groupEntries} onUpdate={onSetGroupEntries} />
    );

    return renderField("groups", title, control);
  }

  function renderTagsField() {
    const title = intl.formatMessage({ id: "tags" });
    return renderField("tag_ids", title, tagsControl(), fullWidthProps);
  }

  function renderPoseTagsField() {
    return (
      <PoseTagSelector
        selectedTagIds={selectedPoseTagIds}
        onSelectionChange={onPoseTagSelectionChange}
        disabled={isLoading}
      />
    );
  }

  function renderDetailsField() {
    const props = {
      labelProps: {
        column: true,
        sm: 3,
        lg: 12,
      },
      fieldProps: {
        sm: 9,
        lg: 12,
      },
    };

    return renderInputField("details", "textarea", "details", props);
  }

  function renderIsBrokenField() {
    const title = intl.formatMessage({ id: "is_broken" });
    const control = (
      <Form.Check
        type="checkbox"
        id="is_broken"
        checked={formik.values.is_broken}
        onChange={(e) => formik.setFieldValue("is_broken", e.target.checked)}
        isInvalid={!!formik.errors.is_broken}
      />
    );

    return renderField("is_broken", title, control);
  }

  function renderDurationField(fieldName: keyof InputValues & string, labelId: string) {
    const title = intl.formatMessage({ id: labelId });
    const control = (
      <DurationInput
        value={formik.values[fieldName] as number | null}
        setValue={(value) => formik.setFieldValue(fieldName, value)}
        disabled={isLoading}
        onReset={() => formik.setFieldValue(fieldName, getPlayerPosition() ?? null)}
      />
    );

    return renderField(fieldName, title, control);
  }

  return (
    <div id="scene-edit-details">
      <Prompt
        when={formik.dirty}
        message={intl.formatMessage({ id: "dialogs.unsaved_changes" })}
      />

      {renderScrapeQueryModal()}
      {maybeRenderScrapeDialog()}
      <Form noValidate onSubmit={formik.handleSubmit}>
        <Row className="form-container edit-buttons-container px-3 pt-3">
          <div className="edit-buttons mb-3 pl-0">
            <Button
              className="edit-button"
              variant="primary"
              disabled={
                (!isNew && !formik.dirty) || !isEqual(formik.errors, {})
              }
              onClick={() => formik.submitForm()}
            >
              <FormattedMessage id="actions.save" />
            </Button>
            {onDelete && (
              <Button
                className="edit-button"
                variant="danger"
                onClick={() => onDelete()}
              >
                <FormattedMessage id="actions.delete" />
              </Button>
            )}
            <Button
              className={`edit-button ${trimEnabled ? 'btn-success' : 'btn-secondary'}`}
              onClick={() => setTrimEnabled(!trimEnabled)}
              title={`${trimEnabled ? 'Disable' : 'Enable'} trim mode`}
            >
              {trimEnabled ? 'Trim ON' : 'Trim OFF'}
            </Button>
          </div>
          {!isNew && (
            <div className="ml-auto text-right d-flex">
              <ButtonGroup className="scraper-group">
                <ScraperMenu
                  toggle={intl.formatMessage({ id: "actions.scrape_with" })}
                  stashBoxes={stashConfig?.general.stashBoxes ?? []}
                  scrapers={fragmentScrapers}
                  onScraperClicked={onScrapeClicked}
                  onReloadScrapers={onReloadScrapers}
                />
                <ScraperMenu
                  variant="secondary"
                  toggle={<Icon icon={faSearch} />}
                  stashBoxes={stashConfig?.general.stashBoxes ?? []}
                  scrapers={queryableScrapers}
                  onScraperClicked={onScrapeQueryClicked}
                  onReloadScrapers={onReloadScrapers}
                />
              </ButtonGroup>
            </div>
          )}
        </Row>
        <Row className="form-container px-3">
          <Col lg={7} xl={12}>
            {renderInputField("title")}

            {renderURLListField("urls", onScrapeSceneURL, urlScrapable)}

            {renderDateField("date")}

            {renderDurationField("start_time", "start_time")}
            {renderDurationField("end_time", "end_time")}

            {renderPerformersField()}
            {renderPoseTagsField()}
            {renderTagsField()}
            {renderIsBrokenField()}
            {renderInputField("code", "text", "scene_code")}
            {renderStudioField()}
            {renderInputField("director")}
            {renderGroupsField()}
            {renderGalleriesField()}

            {renderStashIDsField(
              "stash_ids",
              "scenes",
              "stash_ids",
              fullWidthProps
            )}
          </Col>
          <Col lg={5} xl={12}>
            {renderDetailsField()}
            <Form.Group controlId="cover_image">
              <Form.Label>
                <FormattedMessage id="cover_image" />
              </Form.Label>
              {image}
              <ImageInput
                isEditing
                onImageChange={onCoverImageChange}
                onImageURL={onImageLoad}
              />
            </Form.Group>
          </Col>
        </Row>
      </Form>
    </div>
  );
};

export default SceneEditPanel;
