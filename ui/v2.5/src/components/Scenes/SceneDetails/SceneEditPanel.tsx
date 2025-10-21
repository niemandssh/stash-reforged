import React, { useEffect, useState, useMemo, useCallback, useRef } from "react";
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
  useFindTags,
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
import { IPerformerEntry, ScenePerformerTable } from "./ScenePerformerTable";
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
import { PerformerPopover } from "src/components/Performers/PerformerPopover";
import { formikUtils } from "src/utils/form";
import { Studio, StudioSelect } from "src/components/Studios/StudioSelect";
import { Gallery, GallerySelect } from "src/components/Galleries/GallerySelect";
import { Group } from "src/components/Groups/GroupSelect";
import { useTagsEdit } from "src/hooks/tagsEdit";
import { Tag, TagSelect } from "src/components/Tags/TagSelect";
import { ScraperMenu } from "src/components/Shared/ScraperMenu";
import { PoseTagSelector } from "src/components/Shared/PoseTagSelector";

const SceneScrapeDialog = lazyComponent(() => import("./SceneScrapeDialog"));
const SceneQueryModal = lazyComponent(() => import("./SceneQueryModal"));

interface PerformerTagFieldProps {
  performer: Performer;
  tags: Tag[];
  onTagsChange: (tags: Tag[]) => void;
  sceneId?: string;
  hasInitialized: boolean;
  allTags: Tag[];
  fullWidthProps: {
    labelProps: { column: boolean; sm: number };
    fieldProps: { sm: number };
  };
}

const PerformerTagField: React.FC<PerformerTagFieldProps> = ({
  performer,
  tags,
  onTagsChange,
  sceneId,
  hasInitialized,
  allTags,
  fullWidthProps,
}) => {
  // Simple onSelect handler for performer tags
  const onSelectPerformerTags = React.useCallback((selectedTags: Tag[]) => {
    onTagsChange(selectedTags);
  }, [onTagsChange]);

  const title = (
    <>
      <FormattedMessage id="tags" />
      <PerformerPopover id={performer.id}>
        <span className="ml-2 text-muted">({performer.name})</span>
      </PerformerPopover>
    </>
  );

  return (
    <Form.Group key={performer.id} controlId={`performer_tags_${performer.id}`} as={Row}>
      <Form.Label {...fullWidthProps.labelProps}>{title}</Form.Label>
      <Col {...fullWidthProps.fieldProps}>
        <TagSelect
          values={tags}
          onSelect={onSelectPerformerTags}
          isMulti
          creatable
          instanceId={`performer-${performer.id}-tags`}
        />
      </Col>
    </Form.Group>
  );
};

interface IProps {
  scene: Partial<GQL.SceneDataFragment>;
  initialCoverImage?: string;
  isNew?: boolean;
  isVisible: boolean;
  onSubmit: (input: GQL.SceneUpdateInput) => Promise<void>;
  onForceRefresh?: () => void;
  onDelete?: () => void;
  onTagsChange?: (tags: GQL.Tag[], performerTagIds: any[]) => void;
}

export const SceneEditPanel: React.FC<IProps> = ({
  scene,
  initialCoverImage,
  isNew = false,
  isVisible,
  onSubmit,
  onForceRefresh,
  onDelete,
  onTagsChange,
}) => {
  const intl = useIntl();
  const Toast = useToast();

  const [galleries, setGalleries] = useState<Gallery[]>([]);
  const [performers, setPerformers] = useState<Performer[]>([]);
  const [performerEntries, setPerformerEntries] = useState<IPerformerEntry[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [studio, setStudio] = useState<Studio | null>(null);
  const [allPerformerTags, setAllPerformerTags] = useState<Map<string, string[]>>(new Map());
  
  const Scrapers = useListSceneScrapers();
  const [fragmentScrapers, setFragmentScrapers] = useState<GQL.Scraper[]>([]);
  const [queryableScrapers, setQueryableScrapers] = useState<GQL.Scraper[]>([]);

  const [scraper, setScraper] = useState<GQL.ScraperSourceInput>();
  const [isScraperQueryModalOpen, setIsScraperQueryModalOpen] =
    useState<boolean>(false);
  const [scrapedScene, setScrapedScene] = useState<GQL.ScrapedScene | null>();
  const [endpoint, setEndpoint] = useState<string>();
  const [selectedPoseTagIds, setSelectedPoseTagIds] = useState<string[]>([]);
  const [hasUserInteractedWithPoseTags, setHasUserInteractedWithPoseTags] = useState(false);
  const [hasUserInteractedWithTags, setHasUserInteractedWithTags] = useState(false);
  const [hasUserInteractedWithPerformerTags, setHasUserInteractedWithPerformerTags] = useState(false);
  const [hasInitialized, setHasInitialized] = useState(false);
  const isMountedRef = useRef(true);

  useEffect(() => {
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  // Function to force refresh form data
  const forceRefreshForm = () => {
    setHasInitialized(false);
    // Reset interaction flags to allow data updates
    setHasUserInteractedWithTags(false);
    setHasUserInteractedWithPoseTags(false);
    setHasUserInteractedWithPerformerTags(false);
    // Call parent refresh function
    if (onForceRefresh) {
      onForceRefresh();
    }
  };

  const stableSceneGalleries = useMemo(() => scene.galleries, [scene.galleries?.map(g => `${g.id}-${galleryTitle(g)}`).join(',')]);

  useEffect(() => {
    // Update galleries only when first initializing or when forced refresh
    if (!hasInitialized) {
      setGalleries(
        stableSceneGalleries?.map((g) => ({
          id: g.id,
          title: galleryTitle(g),
          files: g.files,
          folder: g.folder,
        })) ?? []
      );
    }
  }, [stableSceneGalleries, hasInitialized]);

  const stableScenePerformers = useMemo(() => scene.performers, [scene.performers?.map(p => p.id).join(',')]);

  useEffect(() => {
    // Update performers only when first initializing or when forced refresh
    if (!hasInitialized) {
      setPerformers(stableScenePerformers ?? []);
      
      // Initialize performerEntries from scene_performers if available
      if ((scene as any).scene_performers) {
        setPerformerEntries(
          (scene as any).scene_performers.map((sp: any) => ({
            performer: sp.performer,
            small_role: sp.performer.small_role || sp.small_role,
            role_description: sp.role_description ?? null,
          }))
        );
      }

      // Initialize allPerformerTags from scene data
      const initialPerformerTags = new Map<string, string[]>();
      (scene.performer_tag_ids || []).forEach((pt: any) => {
        if (pt.performer_id && pt.tag_ids) {
          initialPerformerTags.set(pt.performer_id, pt.tag_ids);
        }
      });
      setAllPerformerTags(initialPerformerTags);
    }
  }, [stableScenePerformers, hasInitialized, (scene as any).scene_performers]);

  const stableSceneGroups = useMemo(() => scene.groups, [scene.groups?.map(m => `${m.group.id}-${m.scene_index}`).join(',')]);

  useEffect(() => {
    // Update groups only when first initializing or when forced refresh
    if (!hasInitialized) {
      setGroups(stableSceneGroups?.map((m) => m.group) ?? []);
    }
  }, [stableSceneGroups, hasInitialized]);

  const stableSceneStudio = useMemo(() => scene.studio, [scene.studio?.id]);

  useEffect(() => {
    // Update studio only when first initializing or when forced refresh
    if (!hasInitialized) {
      setStudio(stableSceneStudio ?? null);
    }
  }, [stableSceneStudio, hasInitialized]);

  const { configuration: stashConfig } = React.useContext(ConfigurationContext);

  // Network state
  const [isLoading, setIsLoading] = useState(false);

  // Mark as initialized when edit tab becomes visible
  useEffect(() => {
    if (isVisible && !hasInitialized) {
      setHasInitialized(true);
    }
  }, [isVisible, hasInitialized]);

  const { trimEnabled, setTrimEnabled } = useTrimContext();

  const schema = yup.object({
    title: yup.string().ensure(),
    code: yup.string().ensure(),
    urls: yupUniqueStringList(intl),
    date: yupDateString(intl),
    shoot_date: yupDateString(intl),
    director: yup.string().ensure(),
    gallery_ids: yup.array(yup.string().required()).defined(),
    studio_id: yup.string().required().nullable(),
    performer_ids: yup.array(yup.string().required()).defined(),
    scene_performers: yup
      .array(
        yup.object({
          performer_id: yup.string().required(),
          small_role: yup.boolean().required(),
          role_description: yup.string().nullable().optional(),
        })
      )
      .defined(),
    groups: yup
      .array(
        yup.object({
          group_id: yup.string().required(),
          scene_index: yup.number().integer().nullable().defined(),
        })
      )
      .defined(),
    tag_ids: yup.array(yup.string().required()).defined(),
    performer_tag_ids: yup.array(
      yup.object({
        performer_id: yup.string().required(),
        tag_ids: yup.array(yup.string().required()).defined(),
      })
    ).defined(),
    stash_ids: yup.mixed<GQL.StashIdInput[]>().defined(),
    details: yup.string().ensure(),
    cover_image: yup.string().nullable().optional(),
    is_broken: yup.boolean().defined(),
    is_not_broken: yup.boolean().defined(),
    start_time: yup.number().nullable().optional(),
    end_time: yup.number().nullable().optional(),
  });

  const initialValues = useMemo(
    () => ({
      title: scene.title ?? "",
      code: scene.code ?? "",
      urls: scene.urls ?? [],
      date: scene.date ?? "",
      shoot_date: (scene as any).shoot_date ?? "",
      director: scene.director ?? "",
      gallery_ids: (scene.galleries ?? []).map((g) => g.id),
      studio_id: scene.studio?.id ?? null,
      performer_ids: (scene.performers ?? []).map((p) => p.id),
      scene_performers: ((scene as any).scene_performers ?? []).map((sp: any) => ({
        performer_id: sp.performer.id,
        small_role: sp.performer.small_role || sp.small_role,
        role_description: sp.role_description ?? null,
      })),
      groups: (scene.groups ?? []).map((m) => {
        return { group_id: m.group.id, scene_index: m.scene_index ?? null };
      }),
      tag_ids: (() => {
        const allSceneTags = scene.tags ?? [];
        const performerTagIds = new Set<string>();


        // Collect all tag IDs used in performer tags
        (scene.performer_tag_ids ?? []).forEach((pt: any) => {
          if (pt.tag_ids) {
            pt.tag_ids.forEach((tagId: string) => performerTagIds.add(tagId));
          }
        });


        // Include ALL tags (both scene tags and performer tags)
        const allTagIds = new Set<string>();
        
        // Add scene tags
        allSceneTags.forEach(tag => allTagIds.add(tag.id));
        
        // Add performer tags
        performerTagIds.forEach(tagId => allTagIds.add(tagId));

        return Array.from(allTagIds);
      })(),
      performer_tag_ids: (scene.performer_tag_ids ?? [])
        .filter((pt: any) => pt.performer_id && pt.performer_id !== 'undefined' && pt.performer_id !== 'null')
        .map((pt: any) => ({
          performer_id: pt.performer_id,
          tag_ids: pt.tag_ids || []
        })),
      stash_ids: getStashIDs(scene.stash_ids),
      details: scene.details ?? "",
      cover_image: initialCoverImage,
      is_broken: scene.is_broken ?? false,
      is_not_broken: scene.is_not_broken ?? false,
      start_time: scene.start_time ?? null,
      end_time: scene.end_time ?? null,
    }),
    [scene, initialCoverImage]
  );

  type InputValues = yup.InferType<typeof schema>;

  const formik = useFormik<InputValues>({
    initialValues,
    enableReinitialize: false, // Disable automatic reinitialization to preserve dirty state
    validate: yupFormikValidate(schema),
    onSubmit: (values) => onSave(schema.cast(values)),
  });

  // Time fields are not automatically updated - they only update when form is reinitialized


  // Calculate regular scene tags (excluding performer tags for display purposes only)
  const regularSceneTags = useMemo(() => {
    const allSceneTags = scene.tags ?? [];

    // Convert GQL.TagDataFragment to Tag format for TagSelect
    const convertedSceneTags = allSceneTags.map(tag => ({
      id: tag.id,
      name: tag.name,
      sort_name: tag.sort_name,
      aliases: tag.aliases,
      image_path: tag.image_path,
      is_pose_tag: tag.is_pose_tag,
      color: tag.color
    }));

    const performerTagIds = new Set<string>();

    // Only filter if performer_tag_ids data is actually loaded and has valid structure
    if (scene.performer_tag_ids && Array.isArray(scene.performer_tag_ids)) {
      // Collect all tag IDs used in performer tags
      scene.performer_tag_ids.forEach((pt: any) => {
        // Check if this is a valid performer tag entry (has performer_id and tag_ids)
        if (pt && pt.performer_id && pt.tag_ids && Array.isArray(pt.tag_ids)) {
          pt.tag_ids.forEach((tagId: string) => {
            if (tagId && typeof tagId === 'string') {
              performerTagIds.add(tagId);
            }
          });
        }
      });
    }

    // Filter out performer tag IDs from scene tags only if we have performer tag data
    if (performerTagIds.size > 0) {
      return convertedSceneTags.filter((t) => !performerTagIds.has(t.id));
    } else {
      // If no performer tag data loaded yet, show all scene tags
      return convertedSceneTags;
    }
  }, [scene.tags, scene.performer_tag_ids]);

  const { tags, updateTagsStateFromScraper, tagsControl, onSetTags, undoTags, redoTags, clearHistory } = useTagsEdit(
    regularSceneTags,
    (ids) => formik.setFieldValue("tag_ids", ids),
    scene.id,
    false // Disable automatic synchronization - only manual refresh
  );



  const [allTags, setAllTags] = useState<GQL.Tag[]>([]);

  useEffect(() => {
    if (formik.touched.tag_ids && Array.isArray(formik.touched.tag_ids)) {
      setHasUserInteractedWithTags(true);
    }
  }, [formik.touched.tag_ids]);

  // Reset interaction flags when switching away from edit tab
  useEffect(() => {
    if (!isVisible) {
      setHasUserInteractedWithTags(false);
      setHasUserInteractedWithPoseTags(false);
      setHasUserInteractedWithPerformerTags(false);
    }
  }, [isVisible]);

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
    // Update pose tags from scene data only when first initializing or when forced refresh, and only if user hasn't interacted with them
    if (scene.tags && !hasUserInteractedWithPoseTags && !hasInitialized) {
      const poseTagIds = scene.tags
        .filter(tag => tag.is_pose_tag)
        .map(tag => tag.id);
      setSelectedPoseTagIds(poseTagIds);
    }
  }, [scene.tags, hasUserInteractedWithPoseTags, hasInitialized]);

  // Function to update performer tags and mark form as dirty
  const updatePerformerTags = useCallback((performerId: string, tags: Tag[]) => {
    // Mark that user has interacted with performer tags
    setHasUserInteractedWithPerformerTags(true);

    // Get current performer tags from formik
    const currentPerformerTags = formik.values.performer_tag_ids || [];

    // Remove any existing entries for this performer (to avoid duplicates)
    let updatedPerformerTags = currentPerformerTags.filter((pt: any) => pt.performer_id !== performerId);

    // Add the updated entry
    updatedPerformerTags = [...updatedPerformerTags, {
      performer_id: performerId,
      tag_ids: Array.from(new Set(tags.map(tag => tag.id))) // Remove duplicates
    }];

    // Update performer_tag_ids
    formik.setFieldValue("performer_tag_ids", updatedPerformerTags, true);
    formik.setFieldTouched("performer_tag_ids", true);

    // Update allPerformerTags to preserve tags even when performer is removed
    const tagIds = tags.map(tag => tag.id);
    setAllPerformerTags(prev => {
      const newMap = new Map(prev);
      newMap.set(performerId, tagIds);
      return newMap;
    });

    // Update main tag_ids to include all tags (regular + performer tags)
    const currentMainTags = formik.values.tag_ids || [];
    const performerTagIds = new Set(tags.map(tag => tag.id));
    
    // Add performer tags to main tags if not already present
    const updatedMainTags = Array.from(new Set([...currentMainTags, ...performerTagIds]));
    
    formik.setFieldValue("tag_ids", updatedMainTags, true);
    formik.setFieldTouched("tag_ids", true);
  }, [formik]);



  useEffect(() => {
    // Sync pose tags between tags and selectedPoseTagIds only when first initializing or when forced refresh, and only if user hasn't interacted with them
    if (allTags.length > 0 && tags.length > 0 && !hasUserInteractedWithPoseTags && !hasInitialized) {
      const poseTagIdsFromTags = tags
        .filter(tag => tag.is_pose_tag)
        .map(tag => tag.id);

      if (!isEqual(poseTagIdsFromTags.sort(), selectedPoseTagIds.sort())) {
        setSelectedPoseTagIds(poseTagIdsFromTags);
      }
    }
  }, [tags, allTags, selectedPoseTagIds, hasUserInteractedWithPoseTags, hasInitialized]);


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

  const performerEntriesFromFormik = useMemo(() => {
    return formik.values.scene_performers
      .map((sp) => {
        const performer = performerEntries.find((p) => p.performer.id === sp.performer_id);
        if (!performer) return null;
        return {
          performer: performer.performer,
          small_role: sp.small_role,
          role_description: sp.role_description,
        };
      })
      .filter((p) => p !== null) as IPerformerEntry[];
  }, [formik.values.scene_performers, performerEntries]);

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

    // Update performerEntries to match the new performers
    const newPerformerEntries = items.map((performer) => ({
      performer,
      small_role: performer.small_role || false,
      role_description: null,
    }));
    setPerformerEntries(newPerformerEntries);

    // Update scene_performers in formik
    const newScenePerformers = items.map((performer) => ({
      performer_id: performer.id,
      small_role: performer.small_role || false,
      role_description: null,
    }));
    formik.setFieldValue("scene_performers", newScenePerformers);

    // Update performer tags - remove entries for performers that are no longer selected
    const currentPerformerIds = new Set(items.map(p => p.id));
    const currentPerformerTags = formik.values.performer_tag_ids || [];
    const updatedPerformerTags = currentPerformerTags.filter((pt: any) =>
      currentPerformerIds.has(pt.performer_id)
    );

    formik.setFieldValue("performer_tag_ids", updatedPerformerTags);

    // Mark form as dirty when performers change
    formik.setFieldTouched("performer_ids", true);
    formik.setFieldTouched("performer_tag_ids", true);
    formik.setFieldTouched("scene_performers", true);
  }

  function onSetStudio(item: Studio | null) {
    setStudio(item);
    formik.setFieldValue("studio_id", item ? item.id : null);
  }

  function onPoseTagSelectionChange(poseTagIds: string[]) {
    // Filter out invalid poseTagIds
    const filteredPoseTagIds = Array.from(new Set(poseTagIds.filter(id => id && typeof id === 'string' && id.trim() !== '')));
    setSelectedPoseTagIds(filteredPoseTagIds);
    setHasUserInteractedWithPoseTags(true);

    // Получаем текущие теги из useTagsEdit (включая новосозданные)
    const currentTags = tags || [];

    // Фильтруем теги, исключая теги позы
    const nonPoseTags = currentTags.filter(tag => !tag.is_pose_tag);

    // Добавляем выбранные теги позы
    const poseTagObjects = filteredPoseTagIds.map(id => allTags.find(t => t.id === id)).filter(Boolean) as Tag[];

    const newTags = [...nonPoseTags, ...poseTagObjects];

    onSetTags(newTags);

    // Ensure form recognizes the change
    formik.setFieldTouched("tag_ids", true);
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

      const handleGlobalKeyDown = (e: KeyboardEvent) => {
        if ((e.ctrlKey || e.metaKey) && e.code === 'KeyZ') {
          e.preventDefault();
          e.stopImmediatePropagation();
          if (e.shiftKey) {
            redoTags();
          } else {
            undoTags();
          }
        }
      };

      document.addEventListener('keydown', handleGlobalKeyDown, true);

      return () => {
        Mousetrap.unbind("s s");
        Mousetrap.unbind("d d");
        document.removeEventListener('keydown', handleGlobalKeyDown, true);
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
    try {
      
      // Clean up performer tags for performers that are no longer in the scene
      const currentPerformerIds = new Set((performerEntries || []).map(p => p.performer.id));
      const cleanedPerformerTagIds = (input.performer_tag_ids || []).filter((pt: any) => 
        currentPerformerIds.has(pt.performer_id)
      );
      
      // Get tag IDs that should be removed (from deleted performers) using allPerformerTags
      const removedPerformerTagIds = new Set<string>();
      Array.from(allPerformerTags.entries()).forEach(([performerId, tagIds]) => {
        if (!currentPerformerIds.has(performerId)) {
          tagIds.forEach(tagId => removedPerformerTagIds.add(tagId));
        }
      });
      
      // Remove tags of deleted performers from main tag_ids
      const cleanedTagIds = (input.tag_ids || []).filter(tagId => !removedPerformerTagIds.has(tagId));
      
      // Sync performer_ids with scene_performers to avoid data inconsistency
      const syncedPerformerIds = (input.scene_performers || []).map(sp => sp.performer_id);
      
      // Extract scene_performers from input to handle it separately
      const { scene_performers, ...inputWithoutScenePerformers } = input;
      
      const inputWithPerformerTags: GQL.SceneUpdateInput = {
        ...inputWithoutScenePerformers,
        id: scene.id!,
        tag_ids: cleanedTagIds,
        performer_ids: syncedPerformerIds,
        performer_tag_ids: cleanedPerformerTagIds,
        scene_performers: scene_performers || [],
      };

      await onSubmit(inputWithPerformerTags);
      // Clear form dirty state after successful save
      if (isMountedRef.current) {
        // Get current values to preserve them
        const currentValues = { ...formik.values };
        
        // Force a complete form reset by calling resetForm with current values
        formik.resetForm({
          values: currentValues,
          touched: {},
          errors: {},
          status: undefined
        });
        
        // Additional cleanup
        clearHistory();
        setHasUserInteractedWithTags(false);
        setHasUserInteractedWithPoseTags(false);
        setHasUserInteractedWithPerformerTags(false);
        
        // Force a re-render to ensure formik state is updated
        setTimeout(() => {
          if (isMountedRef.current) {
            formik.setTouched({});
            formik.setErrors({});
          }
        }, 100);
      }
    } catch (e) {
      Toast.error(e);
    }
    // Delay setting loading to false to avoid state updates during component re-render
    setTimeout(() => {
      if (isMountedRef.current) {
        setIsLoading(false);
      }
    }, 0);
  }

  const encodingImage = ImageUtils.usePasteImage(onImageLoad);

  function onImageLoad(imageData: string) {
    formik.setFieldValue("cover_image", imageData);
  }

  function onCoverImageChange(event: React.FormEvent<HTMLInputElement>) {
    ImageUtils.onImageChange(event, onImageLoad);
  }

  async function onScrapeClicked(s: GQL.ScraperSourceInput) {
    if (isMountedRef.current) {
      setIsLoading(true);
    }
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
      if (isMountedRef.current) {
        setIsLoading(false);
      }
    }
  }

  async function scrapeFromQuery(
    s: GQL.ScraperSourceInput,
    fragment: GQL.ScrapedSceneDataFragment
  ) {
    if (isMountedRef.current) {
      setIsLoading(true);
    }
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
      if (isMountedRef.current) {
        setIsLoading(false);
      }
    }
  }

  function onScrapeQueryClicked(s: GQL.ScraperSourceInput) {
    setScraper(s);
    setEndpoint(s.stash_box_endpoint ?? undefined);
    setIsScraperQueryModalOpen(true);
  }

  async function onReloadScrapers() {
    if (isMountedRef.current) {
      setIsLoading(true);
    }
    try {
      await mutateReloadScrapers();
    } catch (e) {
      Toast.error(e);
    } finally {
      if (isMountedRef.current) {
        setIsLoading(false);
      }
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
    if (isMountedRef.current) {
      setIsLoading(true);
    }
    try {
      const result = await queryScrapeSceneURL(url);
      if (!result.data || !result.data.scrapeSceneURL) {
        return;
      }
      setScrapedScene(result.data.scrapeSceneURL);
    } catch (e) {
      Toast.error(e);
    } finally {
      if (isMountedRef.current) {
        setIsLoading(false);
      }
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
      <ScenePerformerTable
        value={performerEntries}
        onUpdate={onSetPerformerEntries}
        onFieldUpdate={onSetPerformerFieldUpdate}
        ageFromDate={date}
      />
    );

    return renderField("scene_performers", title, control, fullWidthProps);
  }

  function onSetPerformerFieldUpdate(input: IPerformerEntry[]) {
    // Update only the performer entries and scene_performers without touching tags
    setPerformerEntries(input);

    const scenePerformers = input.map((p) => ({
      performer_id: p.performer.id,
      small_role: p.small_role,
      role_description: p.role_description ?? null,
    }));

    formik.setFieldValue("scene_performers", scenePerformers);
    formik.setFieldTouched("scene_performers", true);
  }

  function onSetPerformerEntries(input: IPerformerEntry[]) {
    setPerformerEntries(input);

    const scenePerformers = input.map((p) => ({
      performer_id: p.performer.id,
      small_role: p.small_role,
      role_description: p.role_description ?? null,
    }));

    formik.setFieldValue("scene_performers", scenePerformers);
    formik.setFieldTouched("scene_performers", true);

    // Handle performer tags - preserve existing tags and add new performers
    const currentPerformerTags = formik.values.performer_tag_ids || [];
    const currentPerformerIds = new Set(currentPerformerTags.map((pt: any) => pt.performer_id));
    const inputPerformerIds = new Set(input.map(p => p.performer.id));
    
    // Keep existing performer tags for performers that are still in the input
    const preservedPerformerTags = currentPerformerTags.filter((pt: any) => 
      inputPerformerIds.has(pt.performer_id)
    );
    
    // Find new performers that don't have tags yet
    const newPerformers = input.filter(p => !currentPerformerIds.has(p.performer.id));
    
    // Add performer tag entries for new performers (use saved tags if available)
    const newPerformerTags = newPerformers.map(p => {
      const savedTags = allPerformerTags.get(p.performer.id);
      return {
        performer_id: p.performer.id,
        tag_ids: savedTags || []
      };
    });
    
    // Combine preserved and new performer tags
    const updatedPerformerTags = [...preservedPerformerTags, ...newPerformerTags];
    
    // Final performer tags (newPerformerTags already includes restored tags)
    const finalPerformerTags = updatedPerformerTags;
    
    // Update main tag_ids to include all performer tags
    const currentMainTags = formik.values.tag_ids || [];
    const allPerformerTagIds = new Set<string>();
    finalPerformerTags.forEach((pt: any) => {
      if (pt.tag_ids) {
        pt.tag_ids.forEach((tagId: string) => allPerformerTagIds.add(tagId));
      }
    });
    
    // Add performer tags to main tags if not already present
    const updatedMainTags = Array.from(new Set([...currentMainTags, ...allPerformerTagIds]));
    
    
    formik.setFieldValue("performer_tag_ids", finalPerformerTags, true);
    formik.setFieldTouched("performer_tag_ids", true);
    formik.setFieldValue("tag_ids", updatedMainTags, true);
    formik.setFieldTouched("tag_ids", true);
    
    // Force formik to re-render by triggering validation
    formik.validateForm();
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

  // Include both scene tags and performer tags for requirements checking
  // Use current form values to show requirements for unsaved changes
  const allSceneTags = useMemo(() => {
    const currentSceneTagIds = new Set((formik.values.tag_ids || tags.map(t => t.id)));
    const currentPerformerTagIds = new Set<string>();

    // Always use formik values to prevent overwriting user changes
    const sourcePerformerTags = formik.values.performer_tag_ids || [];

    // Add all performer tag IDs
    (sourcePerformerTags || []).forEach((pt: any) => {
      if (pt.tag_ids) {
        pt.tag_ids.forEach((tagId: string) => currentPerformerTagIds.add(tagId));
      }
    });

    // Combine and deduplicate
    const allTagIds = new Set([...currentSceneTagIds, ...currentPerformerTagIds]);

    // Convert back to Tag objects
    return Array.from(allTagIds).map(id => allTags.find(t => t.id === id)).filter(Boolean) as GQL.Tag[];
  }, [formik.values.tag_ids, formik.values.performer_tag_ids, tags, scene.performer_tag_ids, allTags]);

  // Notify parent component when tags change
  useEffect(() => {
    if (onTagsChange && allSceneTags.length > 0) {
      onTagsChange(allSceneTags, formik.values.performer_tag_ids || []);
    }
  }, [allSceneTags, formik.values.performer_tag_ids, onTagsChange]);

  // Memoize performer tag fields to avoid unnecessary re-renders
  const performerTagFields = useMemo(() => {
    // Always use formik values to prevent overwriting user changes
    const sourceTagIds = formik.values.performer_tag_ids || [];
    const validPerformerTagIds = sourceTagIds?.filter(
      (pt: any) => pt.performer_id && pt.performer_id !== 'undefined' && pt.performer_id !== 'null'
    ) || [];

    // Precompute performer tag data
    const performerTagDataMap = new Map<string, any>();
    (performerEntries || []).filter(performer => performer.performer.id && performer.performer.id !== 'undefined' && performer.performer.id !== 'null').forEach(performer => {
      const performerTagData = validPerformerTagIds.find(
        (pt: any) => pt.performer_id === performer.performer.id
      );
      performerTagDataMap.set(performer.performer.id, performerTagData);
    });

    return (performerEntries || []).filter(performer => performer.performer.id && performer.performer.id !== 'undefined' && performer.performer.id !== 'null').map(performer => {
      const performerTagData = performerTagDataMap.get(performer.performer.id);
      const currentTags = performerTagData ? (Array.from(new Set(performerTagData.tag_ids
        .filter((id: string) => id && typeof id === 'string' && id !== 'undefined' && id.trim() !== '') // Filter out invalid IDs
      ))
        // @ts-ignore
        .map((id: string) => allTags.find(t => t.id === id))
        .filter((t): t is GQL.Tag => !!t)) as Tag[] : [];

      return (
        <PerformerTagField
          key={`${performer.performer.id}-${performer.performer.name}`}
          performer={performer.performer}
          tags={currentTags}
          sceneId={scene.id}
          hasInitialized={hasInitialized}
          allTags={allTags}
          fullWidthProps={fullWidthProps}
          onTagsChange={(tags) => {
            updatePerformerTags(performer.performer.id, tags);
          }}
        />
      );
    });
  }, [performerEntries, formik.values.performer_tag_ids, allTags, scene.id, hasInitialized, fullWidthProps, updatePerformerTags]);

  function renderTagsField() {
    return (
      <Form.Group controlId="tag_ids" as={Row}>
        <Form.Label {...fullWidthProps.labelProps}>
          <FormattedMessage id="tags" />
        </Form.Label>
        <Col {...fullWidthProps.fieldProps}>
          <div key="main-tag-select">
            {tagsControl()}
          </div>
        </Col>
      </Form.Group>
    );
  }


  function renderPoseTagsField() {
    return (
      <div key="pose-tag-selector">
        <PoseTagSelector
          selectedTagIds={selectedPoseTagIds}
          onSelectionChange={onPoseTagSelectionChange}
          disabled={isLoading}
        />
      </div>
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

  function renderIsNotBrokenField() {
    const title = intl.formatMessage({ id: "is_not_broken" });
    const control = (
      <Form.Check
        type="checkbox"
        id="is_not_broken"
        checked={formik.values.is_not_broken}
        onChange={(e) => {
          const checked = e.target.checked;
          formik.setFieldValue("is_not_broken", checked);
          // If "Not Broken" is checked, automatically uncheck "Broken"
          if (checked) {
            formik.setFieldValue("is_broken", false);
          }
        }}
        isInvalid={!!formik.errors.is_not_broken}
      />
    );

    return renderField("is_not_broken", title, control);
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

  if (isLoading) {
    return <LoadingIndicator />;
  }

  return (
    <div id="scene-edit-details">
      <Prompt
        when={formik.dirty}
        message={intl.formatMessage({ id: "dialogs.unsaved_changes" })}
      />

      {renderScrapeQueryModal()}
      {maybeRenderScrapeDialog()}
      <Form key={hasInitialized ? 'initialized' : 'not-initialized'} noValidate onSubmit={formik.handleSubmit}>
        <Row className="form-container edit-buttons-container px-3 pt-3">
          <div className="edit-buttons mb-3 pl-0">
            <Button
              className="edit-button"
              variant="primary"
              disabled={
                (!isNew && !formik.dirty && !hasUserInteractedWithPerformerTags) || !isEqual(formik.errors, {})
              }
              title={
                (!isNew && !formik.dirty && !hasUserInteractedWithPerformerTags) ? "No changes to save" :
                !isEqual(formik.errors, {}) ? `Validation errors: ${Object.keys(formik.errors).join(', ')}` :
                ""
              }
              onClick={() => formik.submitForm()}
            >
              <FormattedMessage id="actions.save" />
            </Button> 
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
            <div className="mb-n2">
              {renderInputField("title")}
            </div>

            {renderTagsField()}

            {performerTagFields}

            {renderPoseTagsField()}

            {renderPerformersField()}
            {renderURLListField("urls", onScrapeSceneURL, urlScrapable)}

            {renderDurationField("start_time", "start_time")}
            {renderDurationField("end_time", "end_time")}

            {renderDateField("date", "release_date")}
            {renderDateField("shoot_date")}

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
            {renderIsBrokenField()}
            {renderIsNotBrokenField()}

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
