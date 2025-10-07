import React, { useEffect, useMemo, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Prompt } from "react-router-dom";
import { Button, Form, Col, Row } from "react-bootstrap";
import Mousetrap from "mousetrap";
import * as GQL from "src/core/generated-graphql";
import * as yup from "yup";
import {
  queryScrapeGallery,
  queryScrapeGalleryURL,
  useListGalleryScrapers,
  mutateReloadScrapers,
  queryFindTags,
  useFindColorPresets,
} from "src/core/StashService";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { useFormik } from "formik";
import { GalleryScrapeDialog } from "./GalleryScrapeDialog";
import isEqual from "lodash-es/isEqual";
import { handleUnsavedChanges } from "src/utils/navigation";
import {
  Performer,
  PerformerSelect,
} from "src/components/Performers/PerformerSelect";
import {
  yupDateString,
  yupFormikValidate,
  yupUniqueStringList,
} from "src/utils/yup";
import { formikUtils } from "src/utils/form";
import { Studio, StudioSelect } from "src/components/Studios/StudioSelect";
import { Scene, SceneSelect } from "src/components/Scenes/SceneSelect";
import { ListFilterModel } from "src/models/list-filter/filter";
import { useTagsEdit } from "src/hooks/tagsEdit";
import { ScraperMenu } from "src/components/Shared/ScraperMenu";
import { PoseTagSelector } from "src/components/Shared/PoseTagSelector";
import { TagRequirementsIndicator } from "src/components/Shared/TagRequirementsIndicator";
import { Tag } from "src/components/Tags/TagSelect";

interface IProps {
  gallery: Partial<GQL.GalleryDataFragment>;
  isVisible: boolean;
  onSubmit: (input: GQL.GalleryCreateInput) => Promise<void>;
  onDelete: () => void;
}

export const GalleryEditPanel: React.FC<IProps> = ({
  gallery,
  isVisible,
  onSubmit,
  onDelete,
}) => {
  const intl = useIntl();
  const Toast = useToast();
  const [scenes, setScenes] = useState<Scene[]>([]);

  const [performers, setPerformers] = useState<Performer[]>([]);
  const [studio, setStudio] = useState<Studio | null>(null);
  const [selectedPoseTagIds, setSelectedPoseTagIds] = useState<string[]>([]);
  const [allTags, setAllTags] = useState<Tag[]>([]);

  const isNew = gallery.id === undefined;

  const scrapers = useListGalleryScrapers();

  const { data: presetsData } = useFindColorPresets();
  const colorPresets = presetsData?.findColorPresets?.color_presets || [];

  const [scrapedGallery, setScrapedGallery] =
    useState<GQL.ScrapedGallery | null>();

  // Network state
  const [isLoading, setIsLoading] = useState(false);

  const titleRequired =
    isNew || (gallery?.files?.length === 0 && !gallery?.folder);

  const schema = yup.object({
    title: titleRequired ? yup.string().required() : yup.string().ensure(),
    code: yup.string().ensure(),
    urls: yupUniqueStringList(intl),
    date: yupDateString(intl),
    photographer: yup.string().ensure(),
    studio_id: yup.string().required().nullable(),
    performer_ids: yup.array(yup.string().required()).defined(),
    tag_ids: yup.array(yup.string().required()).defined(),
    scene_ids: yup.array(yup.string().required()).defined(),
    details: yup.string().ensure(),
  });

  const initialValues = {
    title: gallery?.title ?? "",
    code: gallery?.code ?? "",
    urls: gallery?.urls ?? [],
    date: gallery?.date ?? "",
    photographer: gallery?.photographer ?? "",
    studio_id: gallery?.studio?.id ?? null,
    performer_ids: (gallery?.performers ?? []).map((p) => p.id),
    tag_ids: (gallery?.tags ?? []).map((t) => t.id),
    scene_ids: (gallery?.scenes ?? []).map((s) => s.id),
    details: gallery?.details ?? "",
  };

  type InputValues = yup.InferType<typeof schema>;

  const formik = useFormik<InputValues>({
    initialValues,
    enableReinitialize: false, // Don't auto-reinitialize to prevent losing user changes
    validate: yupFormikValidate(schema),
    onSubmit: (values) => onSave(schema.cast(values)),
  });

  const { tags, updateTagsStateFromScraper, tagsControl, onSetTags } = useTagsEdit(
    gallery.tags,
    (ids) => formik.setFieldValue("tag_ids", ids),
    gallery.id,
    !isVisible
  );

  function onSetScenes(items: Scene[]) {
    setScenes(items);
    formik.setFieldValue(
      "scene_ids",
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
    
    // Получаем текущие теги из useTagsEdit (включая новосозданные)
    const currentTags = tags || [];
    
    // Фильтруем теги, исключая теги позы
    const nonPoseTags = currentTags.filter(tag => !tag.is_pose_tag);
    
    // Добавляем выбранные теги позы
    const poseTagObjects = poseTagIds.map(id => allTags.find(t => t.id === id)).filter(Boolean) as Tag[];
    
    const newTags = [...nonPoseTags, ...poseTagObjects];
    
    onSetTags(newTags);
  }

  useEffect(() => {
    // Only update form data when edit panel is not visible
    if (!isVisible) {
      setPerformers(gallery.performers ?? []);
    }
  }, [gallery.performers, isVisible]);

  useEffect(() => {
    // Only update form data when edit panel is not visible
    if (!isVisible) {
      setStudio(gallery.studio ?? null);
    }
  }, [gallery.studio, isVisible]);

  useEffect(() => {
    // Only update form data when edit panel is not visible
    if (!isVisible) {
      setScenes(gallery.scenes ?? []);
    }
  }, [gallery.scenes, isVisible]);

  // Загружаем все теги для поз тегов
  useEffect(() => {
    const loadAllTags = async () => {
      try {
        const filter = new ListFilterModel(GQL.FilterMode.Tags);
        filter.itemsPerPage = -1;
        filter.sortBy = "name";
        filter.sortDirection = GQL.SortDirectionEnum.Asc;
        
        const result = await queryFindTags(filter);
        const loadedTags = result.data.findTags.tags as unknown as Tag[];
        setAllTags(loadedTags);
      } catch (error) {
        console.error("Error loading tags:", error);
      }
    };

    loadAllTags();
  }, []);

  // Инициализируем выбранные поз теги
  useEffect(() => {
    if (gallery.tags && allTags.length > 0) {
      const poseTagIds = gallery.tags
        .filter(tag => tag.is_pose_tag)
        .map(tag => tag.id);
      setSelectedPoseTagIds(poseTagIds);
    }
  }, [gallery.tags, allTags]);

  useEffect(() => {
    if (isVisible) {
      Mousetrap.bind("s s", () => {
        if (formik.dirty) {
          formik.submitForm();
        }
      });
      Mousetrap.bind("d d", () => {
        onDelete();
      });

      return () => {
        Mousetrap.unbind("s s");
        Mousetrap.unbind("d d");
      };
    }
  });

  const fragmentScrapers = useMemo(() => {
    return (scrapers?.data?.listScrapers ?? []).filter((s) =>
      s.gallery?.supported_scrapes.includes(GQL.ScrapeType.Fragment)
    );
  }, [scrapers]);

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

  async function onScrapeClicked(s: GQL.ScraperSourceInput) {
    if (!gallery || !gallery.id) return;

    setIsLoading(true);
    try {
      const result = await queryScrapeGallery(s.scraper_id!, gallery.id);
      if (!result.data || !result.data.scrapeSingleGallery?.length) {
        Toast.success("No galleries found");
        return;
      }
      setScrapedGallery(result.data.scrapeSingleGallery[0]);
    } catch (e) {
      Toast.error(e);
    } finally {
      setIsLoading(false);
    }
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

  function onScrapeDialogClosed(data?: GQL.ScrapedGalleryDataFragment) {
    if (data) {
      updateGalleryFromScrapedGallery(data);
    }
    setScrapedGallery(undefined);
  }

  function maybeRenderScrapeDialog() {
    if (!scrapedGallery) {
      return;
    }

    const currentGallery = {
      id: gallery.id!,
      ...formik.values,
    };

    return (
      <GalleryScrapeDialog
        gallery={currentGallery}
        galleryStudio={studio}
        galleryTags={tags}
        galleryPerformers={performers}
        scraped={scrapedGallery}
        onClose={(data) => {
          onScrapeDialogClosed(data);
        }}
      />
    );
  }

  function urlScrapable(scrapedUrl: string): boolean {
    return (scrapers?.data?.listScrapers ?? []).some((s) =>
      (s?.gallery?.urls ?? []).some((u) => scrapedUrl.includes(u))
    );
  }

  function updateGalleryFromScrapedGallery(
    galleryData: GQL.ScrapedGalleryDataFragment
  ) {
    if (galleryData.title) {
      formik.setFieldValue("title", galleryData.title);
    }

    if (galleryData.code) {
      formik.setFieldValue("code", galleryData.code);
    }

    if (galleryData.details) {
      formik.setFieldValue("details", galleryData.details);
    }

    if (galleryData.photographer) {
      formik.setFieldValue("photographer", galleryData.photographer);
    }

    if (galleryData.date) {
      formik.setFieldValue("date", galleryData.date);
    }

    if (galleryData.urls) {
      formik.setFieldValue("urls", galleryData.urls);
    }

    if (galleryData.studio?.stored_id) {
      onSetStudio({
        id: galleryData.studio.stored_id,
        name: galleryData.studio.name ?? "",
        aliases: [],
      });
    }

    if (galleryData.performers?.length) {
      const idPerfs = galleryData.performers.filter((p) => {
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

    updateTagsStateFromScraper(galleryData.tags ?? undefined);
  }

  async function onScrapeGalleryURL(url: string) {
    if (!url) {
      return;
    }
    setIsLoading(true);
    try {
      const result = await queryScrapeGalleryURL(url);
      if (!result || !result.data || !result.data.scrapeGalleryURL) {
        return;
      }
      setScrapedGallery(result.data.scrapeGalleryURL);
    } catch (e) {
      Toast.error(e);
    } finally {
      setIsLoading(false);
    }
  }

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
  const { renderField, renderInputField, renderDateField, renderURLListField } =
    formikUtils(intl, formik, splitProps);

  function renderScenesField() {
    const title = intl.formatMessage({ id: "scenes" });
    const control = (
      <SceneSelect
        values={scenes}
        onSelect={(items) => onSetScenes(items)}
        isMulti
      />
    );

    return renderField("scene_ids", title, control);
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

  function renderTagsField() {
    const title = (
      <div className="d-flex align-items-center w-100 justify-content-between" style={{ paddingRight: "15px" }}>
        <FormattedMessage id="tags" />
        <TagRequirementsIndicator tags={tags as GQL.Tag[]} colorPresets={colorPresets} />
      </div>
    );
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

  return (
    <div id="gallery-edit-details">
      <Prompt
        when={formik.dirty}
        message={handleUnsavedChanges(intl, "galleries", gallery?.id)}
      />

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
            <Button
              className="edit-button"
              variant="danger"
              onClick={() => onDelete()}
            >
              <FormattedMessage id="actions.delete" />
            </Button>
          </div>
          <div className="ml-auto text-right d-flex">
            {!isNew && (
              <ScraperMenu
                toggle={intl.formatMessage({ id: "actions.scrape_with" })}
                scrapers={fragmentScrapers}
                onScraperClicked={onScrapeClicked}
                onReloadScrapers={onReloadScrapers}
              />
            )}
          </div>
        </Row>
        <Row className="form-container px-3">
          <Col lg={7} xl={12}>
            {renderInputField("title")}

            {renderURLListField("urls", onScrapeGalleryURL, urlScrapable)}

            {renderDateField("date")}

            {renderPerformersField()}
            {renderTagsField()}
            {renderPoseTagsField()}
            {renderInputField("code", "text", "scene_code")}
            {renderInputField("photographer")}
            {renderScenesField()}
            {renderStudioField()}
          </Col>
          <Col lg={5} xl={12}>
            {renderDetailsField()}
          </Col>
        </Row>
      </Form>
    </div>
  );
};
