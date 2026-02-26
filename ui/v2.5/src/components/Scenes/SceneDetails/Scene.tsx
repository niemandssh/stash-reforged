import {
  Tab,
  Nav,
  Dropdown,
  Button,
  Alert,
  OverlayTrigger,
  Tooltip,
} from "react-bootstrap";
import React, {
  Suspense,
  useEffect,
  useState,
  useMemo,
  useContext,
  useRef,
  useCallback,
} from "react";
import { FormattedDate, FormattedMessage, useIntl } from "react-intl";
import { Link, RouteComponentProps } from "react-router-dom";
import { Helmet } from "react-helmet";
import * as GQL from "src/core/generated-graphql";
import {
  mutateMetadataScan,
  useFindScene,
  useSceneIncrementO,
  useSceneIncrementOmg,
  useSceneAddOmg,
  useSceneGenerateScreenshot,
  useSceneSaveFilteredScreenshot,
  useSceneUpdate,
  queryFindScenes,
  queryFindScenesByID,
  useSceneIncrementPlayCount,
  useSceneConvertToMP4,
  useSceneConvertHLSToMP4,
  useSceneSetBroken,
  useSceneSetNotBroken,
  useScanVideoFileThreats,
  useFindColorPresets,
} from "src/core/StashService";

import { SceneEditPanel } from "./SceneEditPanel";
import { ErrorMessage } from "src/components/Shared/ErrorMessage";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { Icon } from "src/components/Shared/Icon";
import { Counter } from "src/components/Shared/Counter";
import { BrokenBadge } from "src/components/Shared/BrokenBadge";
import { ProbablyBrokenBadge } from "src/components/Shared/ProbablyBrokenBadge";
import { HLSBadge } from "src/components/Shared/HLSBadge";
import { TagRequirementsIndicator } from "src/components/Shared/TagRequirementsIndicator";
import { useToast } from "src/hooks/Toast";
import SceneQueue, { QueuedScene } from "src/models/sceneQueue";
import { ListFilterModel } from "src/models/list-filter/filter";
import Mousetrap from "mousetrap";
import { OrganizedButton } from "./OrganizedButton";
import { ConfigurationContext } from "src/hooks/Config";
import { getPlayerPosition } from "src/components/ScenePlayer/util";
import {
  faEllipsisV,
  faChevronRight,
  faChevronLeft,
  faVideo,
  faSync,
  faSearch,
  faCog,
  faPlay,
  faWrench,
  faCamera,
  faImage,
  faCompressAlt,
  faCut,
  faImages,
  faPhotoVideo,
  faExclamationTriangle,
  faCheckCircle,
  faUpload,
  faExchangeAlt,
  faTrash,
  faShieldAlt,
  faInfoCircle,
  faListUl,
  faMapMarkerAlt,
  faLayerGroup,
  faSlidersH,
  faFileAlt,
  faHistory,
  faEdit,
  faTimes,
} from "@fortawesome/free-solid-svg-icons";
import { objectPath, objectTitle } from "src/core/files";
import { RatingSystem } from "src/components/Shared/Rating/RatingSystem";
import TextUtils from "src/utils/text";
import {
  OCounterButton,
  OMGCounterButton,
  ViewCountButton,
} from "src/components/Shared/CountButton";
import { SceneCountAttributeModal } from "./SceneCountAttributeModal";
import { useRatingKeybinds } from "src/hooks/keybinds";
import { lazyComponent } from "src/utils/lazyComponent";
import cx from "classnames";
import { PatchComponent, PatchContainerComponent } from "src/patch";
import { isHLSVideo } from "src/utils/hlsDetection";
import isEqual from "lodash-es/isEqual";

const SubmitStashBoxDraft = lazyComponent(
  () => import("src/components/Dialogs/SubmitDraft")
);
const ScenePlayer = lazyComponent(
  () => import("src/components/ScenePlayer/ScenePlayer")
);

const GalleryViewer = lazyComponent(
  () => import("src/components/Galleries/GalleryViewer")
);
const ExternalPlayerButton = lazyComponent(
  () => import("./ExternalPlayerButton")
);

const QueueViewer = lazyComponent(() => import("./QueueViewer"));
const SceneMarkersPanel = lazyComponent(() => import("./SceneMarkersPanel"));
const SceneFileInfoPanel = lazyComponent(() => import("./SceneFileInfoPanel"));
const SceneDetailPanel = lazyComponent(() => import("./SceneDetailPanel"));
const SceneHistoryPanel = lazyComponent(() => import("./SceneHistoryPanel"));
const SceneGroupPanel = lazyComponent(() => import("./SceneGroupPanel"));
const SceneGalleriesPanel = lazyComponent(
  () => import("./SceneGalleriesPanel")
);
const DeleteScenesDialog = lazyComponent(() => import("../DeleteScenesDialog"));
const GenerateDialog = lazyComponent(
  () => import("../../Dialogs/GenerateDialog")
);
const SceneVideoFilterPanel = lazyComponent(
  () => import("./SceneVideoFilterPanel")
);
import { SceneMergeModal } from "../SceneMergeDialog";
import { ReduceResolutionModal } from "./ReduceResolutionModal";
import { TrimVideoModal } from "./TrimVideoModal";
import { RegenerateSpritesModal } from "./RegenerateSpritesModal";
import { ModalComponent } from "src/components/Shared/Modal";
import { SceneDataUpdateNotification } from "./SceneDataUpdateNotification";
import { captureFilteredSceneScreenshot } from "./captureFilteredScreenshot";

const VideoFrameRateResolution: React.FC<{
  width?: number;
  height?: number;
  frameRate?: number;
}> = ({ width, height, frameRate }) => {
  const intl = useIntl();

  const resolution = useMemo(() => {
    if (width && height) {
      const r = TextUtils.resolution(width, height);
      return (
        <span className="resolution" data-value={r}>
          {r}
        </span>
      );
    }
    return undefined;
  }, [width, height]);

  const frameRateDisplay = useMemo(() => {
    if (frameRate) {
      return (
        <span className="frame-rate" data-value={frameRate}>
          <FormattedMessage
            id="frames_per_second"
            values={{ value: intl.formatNumber(frameRate ?? 0) }}
          />
        </span>
      );
    }
    return undefined;
  }, [intl, frameRate]);

  const divider = useMemo(() => {
    return resolution && frameRateDisplay ? (
      <span className="divider"> | </span>
    ) : undefined;
  }, [resolution, frameRateDisplay]);

  return (
    <span>
      {frameRateDisplay}
      {divider}
      {resolution}
    </span>
  );
};

interface IProps {
  scene: GQL.SceneDataFragment;
  setTimestamp: (num: number, programmatic?: boolean) => void;
  queueScenes: QueuedScene[];
  onQueueNext: () => void;
  onQueuePrevious: () => void;
  onQueueRandom: () => void;
  onQueueSceneClicked: (sceneID: string) => void;
  onDelete: () => void;
  continuePlaylist: boolean;
  queueHasMoreScenes: boolean;
  onQueueMoreScenes: () => void;
  onQueueLessScenes: () => void;
  queueStart: number;
  collapsed: boolean;
  setCollapsed: (state: boolean) => void;
  setContinuePlaylist: (value: boolean) => void;
  onSaved?: () => Promise<void> | void;
  onPlayMarkers: (markers: GQL.SceneMarkerDataFragment[]) => void;
  onStopMarkers: () => void;
  playingTagId?: string;
  onPlayAllMarkers: (markers: GQL.SceneMarkerDataFragment[]) => void;
  setRightSidebarPanel?: React.Dispatch<
    React.SetStateAction<"queue" | "markers" | "fileinfo" | null>
  >;
}

interface ISceneParams {
  id: string;
}

const ScenePageTabs = PatchContainerComponent<IProps>("ScenePage.Tabs");
const ScenePageTabContent = PatchContainerComponent<IProps>(
  "ScenePage.TabContent"
);

const ScenePage: React.FC<IProps> = PatchComponent("ScenePage", (props) => {
  const {
    scene,
    setTimestamp,
    queueScenes,
    onQueueNext,
    onQueuePrevious,
    onQueueRandom,
    onQueueSceneClicked,
    onDelete,
    continuePlaylist,
    queueHasMoreScenes,
    onQueueMoreScenes,
    onQueueLessScenes,
    queueStart,
    collapsed,
    setCollapsed,
    setContinuePlaylist,
    onSaved,
    onPlayMarkers,
    onStopMarkers,
    playingTagId,
    onPlayAllMarkers,
    setRightSidebarPanel: setRightSidebarPanelFromProps,
  } = props;

  const Toast = useToast();
  const intl = useIntl();
  const [updateScene] = useSceneUpdate();
  const [generateScreenshot] = useSceneGenerateScreenshot();
  const [saveFilteredScreenshot] = useSceneSaveFilteredScreenshot();
  const { configuration } = useContext(ConfigurationContext);

  const [showDraftModal, setShowDraftModal] = useState(false);
  const [showReduceResolutionModal, setShowReduceResolutionModal] =
    useState(false);
  const [showTrimVideoModal, setShowTrimVideoModal] = useState(false);
  const [showRegenerateSpritesModal, setShowRegenerateSpritesModal] =
    useState(false);
  const [showConvertToMP4Confirm, setShowConvertToMP4Confirm] = useState(false);
  const [showConvertHLSToMP4Confirm, setShowConvertHLSToMP4Confirm] =
    useState(false);
  const [isSavingFilteredScreenshot, setIsSavingFilteredScreenshot] =
    useState(false);
  const boxes = configuration?.general?.stashBoxes ?? [];

  const [incrementO] = useSceneIncrementO(scene.id);
  const [incrementOmg] = useSceneIncrementOmg(scene.id);
  const [addOmg] = useSceneAddOmg(scene.id);

  const [showOAttributeModal, setShowOAttributeModal] = useState(false);
  const [showOmgAttributeModal, setShowOmgAttributeModal] = useState(false);

  const scenePerformers = useMemo(() => {
    const fromScene = scene.performers ?? [];
    const fromScenePerformers = (scene.scene_performers ?? []).map(
      (sp) => sp.performer
    );
    return fromScene.length > 0 ? fromScene : fromScenePerformers;
  }, [scene.performers, scene.scene_performers]);

  const [incrementPlay] = useSceneIncrementPlayCount();
  const [convertToMP4] = useSceneConvertToMP4();
  const [convertHLSToMP4] = useSceneConvertHLSToMP4();
  const [setBroken] = useSceneSetBroken();
  const [setNotBroken] = useSceneSetNotBroken();
  const [scanVideoFileThreats] = useScanVideoFileThreats();

  const [scanningThreats, setScanningThreats] = useState(false);

  const { data: presetsData } = useFindColorPresets();
  const colorPresets = presetsData?.findColorPresets?.color_presets || [];

  const [organizedLoading, setOrganizedLoading] = useState(false);
  const [activeTabKey, setActiveTabKey] = useState("scene-details-panel");
  const [editingTags, setEditingTags] = useState<GQL.Tag[] | null>(null);

  // Combine scene tags and performer tags for TagRequirementsIndicator
  // Use editing tags if in edit mode, otherwise use saved scene tags
  const allSceneTags = useMemo(() => {
    // If we're editing and have editing tags, use those
    if (activeTabKey === "scene-edit-panel" && editingTags) {
      return editingTags;
    }

    // Otherwise use saved scene data
    const sceneTagIds = new Set((scene.tags || []).map((tag) => tag.id));
    const performerTagIds = new Set<string>();

    // Add all performer tag IDs from scene data
    (scene.performer_tag_ids || []).forEach((pt: { tag_ids?: string[] }) => {
      if (pt.tag_ids) {
        pt.tag_ids.forEach((tagId: string) => performerTagIds.add(tagId));
      }
    });

    // Combine and deduplicate
    const allTagIds = new Set([...sceneTagIds, ...performerTagIds]);

    // Convert back to Tag objects
    return Array.from(allTagIds)
      .map((id) => (scene.tags || []).find((t) => t.id === id))
      .filter(Boolean) as GQL.Tag[];
  }, [scene.tags, scene.performer_tag_ids, activeTabKey, editingTags]);

  // Reset editing tags when leaving edit panel
  useEffect(() => {
    if (activeTabKey !== "scene-edit-panel") {
      setEditingTags(null);
    }
  }, [activeTabKey]);

  function incrementPlayCount() {
    incrementPlay({
      variables: {
        id: scene.id,
      },
    });
  }

  const [isDeleteAlertOpen, setIsDeleteAlertOpen] = useState<boolean>(false);
  const [isGenerateDialogOpen, setIsGenerateDialogOpen] = useState(false);
  const [isMergeIntoDialogOpen, setIsMergeIntoDialogOpen] = useState(false);
  const [isMergeFromDialogOpen, setIsMergeFromDialogOpen] = useState(false);
  const [showDataUpdateNotification, setShowDataUpdateNotification] =
    useState(false);
  const [lastSceneData, setLastSceneData] =
    useState<GQL.SceneDataFragment | null>(null);

  // Track scene data changes to show update notification
  useEffect(() => {
    if (lastSceneData) {
      // Check if scene data has changed (excluding similar scenes and file info)
      const hasChanged =
        lastSceneData.title !== scene.title ||
        lastSceneData.details !== scene.details ||
        lastSceneData.date !== scene.date ||
        lastSceneData.rating100 !== scene.rating100 ||
        lastSceneData.studio?.id !== scene.studio?.id ||
        lastSceneData.director !== scene.director ||
        lastSceneData.code !== scene.code ||
        lastSceneData.urls !== scene.urls ||
        lastSceneData.organized !== scene.organized ||
        lastSceneData.is_broken !== scene.is_broken ||
        lastSceneData.is_not_broken !== scene.is_not_broken ||
        lastSceneData.start_time !== scene.start_time ||
        lastSceneData.end_time !== scene.end_time ||
        !isEqual(
          lastSceneData.tags?.map((t) => t.id).sort(),
          scene.tags?.map((t) => t.id).sort()
        ) ||
        !isEqual(
          lastSceneData.performers?.map((p) => p.id).sort(),
          scene.performers?.map((p) => p.id).sort()
        ) ||
        !isEqual(
          lastSceneData.galleries?.map((g) => g.id).sort(),
          scene.galleries?.map((g) => g.id).sort()
        ) ||
        !isEqual(
          lastSceneData.groups?.map((g) => g.group.id).sort(),
          scene.groups?.map((g) => g.group.id).sort()
        ) ||
        !isEqual(
          lastSceneData.stash_ids?.map((s) => s.stash_id).sort(),
          scene.stash_ids?.map((s) => s.stash_id).sort()
        );

      if (hasChanged) {
        setShowDataUpdateNotification(true);
      }
    }

    // Update last scene data only when not showing notification
    if (!showDataUpdateNotification && scene) {
      setLastSceneData(scene as GQL.SceneDataFragment);
    }
  }, [scene, lastSceneData, showDataUpdateNotification]);

  // Function to force refresh scene data
  const forceRefreshSceneData = () => {
    setShowDataUpdateNotification(false);
    if (scene) {
      setLastSceneData(scene as GQL.SceneDataFragment);
    }
    // Trigger a refetch of the scene data
    if (onSaved) {
      onSaved();
    }
  };

  const onIncrementOMGClick = async () => {
    try {
      if (scenePerformers.length === 0) {
        await incrementOmg();
      } else {
        setShowOmgAttributeModal(true);
      }
    } catch (e) {
      Toast.error(e);
    }
  };

  const onIncrementOClick = async () => {
    try {
      if (scenePerformers.length === 0) {
        await incrementO();
      } else {
        setShowOAttributeModal(true);
      }
    } catch (e) {
      Toast.error(e);
    }
  };

  const handleOAttributeConfirm = useCallback(
    async (performerIds: string[] | null) => {
      try {
        await incrementO({
          variables: {
            id: scene.id,
            performer_ids: performerIds ?? undefined,
          } as GQL.SceneAddOMutationVariables,
        });
        setShowOAttributeModal(false);
      } catch (e) {
        Toast.error(e);
      }
    },
    [incrementO, scene.id]
  );

  const handleOmgAttributeConfirm = useCallback(
    async (performerIds: string[] | null) => {
      try {
        await addOmg({
          variables: {
            id: scene.id,
            performer_ids: performerIds ?? undefined,
          } as GQL.SceneAddOmgMutationVariables,
        });
        setShowOmgAttributeModal(false);
      } catch (e) {
        Toast.error(e);
      }
    },
    [addOmg, scene.id]
  );

  function setRating(v: number | null) {
    updateScene({
      variables: {
        input: {
          id: scene.id,
          rating100: v,
        },
      },
    });
  }

  useRatingKeybinds(
    true,
    configuration?.ui.ratingSystemOptions?.type,
    setRating
  );

  // set up hotkeys
  useEffect(() => {
    Mousetrap.bind("a", () => setActiveTabKey("scene-details-panel"));
    if (queueScenes.length > 0 && setRightSidebarPanelFromProps) {
      Mousetrap.bind("q", () =>
        setRightSidebarPanelFromProps((p) => (p === "queue" ? null : "queue"))
      );
    }
    Mousetrap.bind("e", () => setActiveTabKey("scene-edit-panel"));
    if (setRightSidebarPanelFromProps) {
      Mousetrap.bind("k", () =>
        setRightSidebarPanelFromProps((p) =>
          p === "markers" ? null : "markers"
        )
      );
    }
    if (setRightSidebarPanelFromProps) {
      Mousetrap.bind("i", () =>
        setRightSidebarPanelFromProps((p) =>
          p === "fileinfo" ? null : "fileinfo"
        )
      );
    }
    Mousetrap.bind("h", () => setActiveTabKey("scene-history-panel"));
    Mousetrap.bind("o", () => {
      onIncrementOClick();
    });
    Mousetrap.bind("p n", () => onQueueNext());
    Mousetrap.bind("p p", () => onQueuePrevious());
    Mousetrap.bind("p r", () => onQueueRandom());
    Mousetrap.bind(",", () => setCollapsed(!collapsed));

    return () => {
      Mousetrap.unbind("a");
      Mousetrap.unbind("q");
      Mousetrap.unbind("e");
      Mousetrap.unbind("k");
      Mousetrap.unbind("i");
      Mousetrap.unbind("h");
      Mousetrap.unbind("o");
      Mousetrap.unbind("p n");
      Mousetrap.unbind("p p");
      Mousetrap.unbind("p r");
      Mousetrap.unbind(",");
    };
  });

  async function onSave(input: GQL.SceneUpdateInput) {
    await updateScene({
      variables: {
        input: input,
      },
    });

    await new Promise((resolve) => setTimeout(resolve, 50));

    if (onSaved) {
      await onSaved();
    }

    Toast.success(
      intl.formatMessage({ id: "toast.scene_with_similars_updated" })
    );
  }

  const onOrganizedClick = async () => {
    try {
      setOrganizedLoading(true);
      await updateScene({
        variables: {
          input: {
            id: scene.id,
            organized: !scene.organized,
          },
        },
      });
    } catch (e) {
      Toast.error(e);
    } finally {
      setOrganizedLoading(false);
    }
  };

  function onClickMarker(marker: GQL.SceneMarkerDataFragment) {
    setTimestamp(marker.seconds, true);
  }

  async function onRescan() {
    await mutateMetadataScan({
      paths: [objectPath(scene)],
      rescan: true,
    });

    Toast.success(
      intl.formatMessage(
        { id: "toast.rescanning_entity" },
        {
          count: 1,
          singularEntity: intl
            .formatMessage({ id: "scene" })
            .toLocaleLowerCase(),
        }
      )
    );
  }

  async function onRescanSimilarity() {
    try {
      // Trigger similarity recalculation by updating a similarity-affecting field
      // This will trigger the similarity job in the backend
      await updateScene({
        variables: {
          input: {
            id: scene.id,
            // Update tag_ids to trigger similarity recalculation
            // We'll set the same tag_ids to trigger the update
            tag_ids: scene.tags.map((tag) => tag.id),
          },
        },
      });

      Toast.success(
        intl.formatMessage(
          { id: "toast.rescanning_similarity" },
          {
            entity: intl.formatMessage({ id: "scene" }).toLocaleLowerCase(),
          }
        )
      );
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onGenerateScreenshot(at?: number) {
    await generateScreenshot({
      variables: {
        id: scene.id,
        at,
      },
    });
    Toast.success(intl.formatMessage({ id: "toast.generating_screenshot" }));
  }

  async function onSaveFilteredScreenshot() {
    if (isSavingFilteredScreenshot) {
      return;
    }

    try {
      setIsSavingFilteredScreenshot(true);
      const captured = await captureFilteredSceneScreenshot(scene);
      const at = captured.at ?? getPlayerPosition() ?? undefined;
      await saveFilteredScreenshot({
        variables: {
          input: {
            id: scene.id,
            image: captured.dataUrl,
            at,
          },
        },
      });
      Toast.success(
        intl.formatMessage({ id: "toast.saved_filtered_screenshot" })
      );
    } catch (e) {
      Toast.error(e);
    } finally {
      setIsSavingFilteredScreenshot(false);
    }
  }

  function onConvertToMP4() {
    setShowConvertToMP4Confirm(true);
  }

  async function confirmConvertToMP4() {
    try {
      const result = await convertToMP4({
        variables: {
          id: scene.id,
        },
      });

      if (result.data?.sceneConvertToMP4) {
        Toast.success(
          intl.formatMessage(
            { id: "actions.convert_to_mp4_started" },
            { jobId: result.data.sceneConvertToMP4 }
          )
        );
      }
      setShowConvertToMP4Confirm(false);
    } catch (e) {
      Toast.error(e);
      setShowConvertToMP4Confirm(false);
    }
  }

  function onConvertHLSToMP4() {
    setShowConvertHLSToMP4Confirm(true);
  }

  async function confirmConvertHLSToMP4() {
    try {
      const result = await convertHLSToMP4({
        variables: {
          id: scene.id,
        },
      });

      if (result.data?.sceneConvertHLSToMP4) {
        Toast.success(
          intl.formatMessage(
            { id: "actions.convert_hls_to_mp4_started" },
            { jobId: result.data.sceneConvertHLSToMP4 }
          )
        );
      }
      setShowConvertHLSToMP4Confirm(false);
    } catch (e) {
      Toast.error(e);
      setShowConvertHLSToMP4Confirm(false);
    }
  }

  function onDeleteDialogClosed(deleted: boolean) {
    setIsDeleteAlertOpen(false);
    if (deleted) {
      onDelete();
    }
  }

  function onMergeIntoOtherScene() {
    setIsMergeIntoDialogOpen(true);
  }

  function onMergeFromOtherScene() {
    setIsMergeFromDialogOpen(true);
  }

  async function onSetBroken() {
    try {
      await setBroken({
        variables: {
          id: scene.id,
        },
      });
      Toast.success(intl.formatMessage({ id: "toast.scene_set_broken" }));
      forceRefreshSceneData();
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onSetNotBroken() {
    try {
      await setNotBroken({
        variables: {
          id: scene.id,
        },
      });
      Toast.success(intl.formatMessage({ id: "toast.scene_set_not_broken" }));
      forceRefreshSceneData();
    } catch (e) {
      Toast.error(e);
    }
  }

  function onMergeDialogClosed(mergedID?: string) {
    setIsMergeIntoDialogOpen(false);
    setIsMergeFromDialogOpen(false);
    if (mergedID) {
      // For merge into: redirect to the destination scene (mergedID)
      // For merge from: redirect to the current scene (since current scene becomes the destination)
      if (isMergeIntoDialogOpen) {
        // Merge into: redirect to destination scene
        window.location.href = `/scenes/${mergedID}`;
      } else {
        // Merge from: refresh current scene since it becomes the destination
        window.location.reload();
      }
    }
  }

  function maybeRenderDeleteDialog() {
    if (isDeleteAlertOpen) {
      return (
        <DeleteScenesDialog
          selected={[{ ...scene, pinned: false }]}
          onClose={onDeleteDialogClosed}
        />
      );
    }
  }

  function maybeRenderSceneGenerateDialog() {
    if (isGenerateDialogOpen) {
      return (
        <GenerateDialog
          selectedIds={[scene.id]}
          onClose={() => {
            setIsGenerateDialogOpen(false);
          }}
          type="scene"
        />
      );
    }
  }

  function maybeRenderMergeIntoDialog() {
    if (isMergeIntoDialogOpen) {
      return (
        <SceneMergeModal
          scenes={[]}
          presetSource={[{ id: scene.id, title: objectTitle(scene) }]}
          presetDestination={[]}
          onClose={onMergeDialogClosed}
          show
        />
      );
    }
  }

  function maybeRenderMergeFromDialog() {
    if (isMergeFromDialogOpen) {
      return (
        <SceneMergeModal
          scenes={[]}
          presetSource={[]}
          presetDestination={[{ id: scene.id, title: objectTitle(scene) }]}
          onClose={onMergeDialogClosed}
          show
        />
      );
    }
  }

  function maybeRenderReduceResolutionDialog() {
    if (showReduceResolutionModal) {
      return (
        <ReduceResolutionModal
          scene={scene}
          onClose={() => setShowReduceResolutionModal(false)}
        />
      );
    }
  }

  function maybeRenderTrimVideoDialog() {
    if (showTrimVideoModal) {
      return (
        <TrimVideoModal
          scene={scene}
          onClose={() => setShowTrimVideoModal(false)}
        />
      );
    }
  }

  function maybeRenderRegenerateSpritesDialog() {
    if (showRegenerateSpritesModal) {
      return (
        <RegenerateSpritesModal
          sceneId={scene.id}
          show={showRegenerateSpritesModal}
          onClose={() => setShowRegenerateSpritesModal(false)}
          onSuccess={() => {
            // Refresh scene data after successful regeneration
            forceRefreshSceneData();
          }}
        />
      );
    }
  }

  function maybeRenderConvertToMP4ConfirmDialog() {
    if (showConvertToMP4Confirm) {
      const originalFormat =
        scene.files.length > 0
          ? scene.files[0].format?.toUpperCase() || "UNKNOWN"
          : "UNKNOWN";

      // Calculate temp path based on generatedPath (same logic as in config.go)
      const generatedPath = configuration?.general?.generatedPath || "";
      const tempPath = generatedPath
        ? generatedPath.substring(0, generatedPath.lastIndexOf("/")) + "/temp"
        : "./temp";

      return (
        <ModalComponent
          show
          icon={faVideo}
          header={intl.formatMessage({ id: "actions.convert_to_mp4" })}
          accept={{
            variant: "danger",
            onClick: confirmConvertToMP4,
            text: intl.formatMessage({ id: "actions.confirm" }),
          }}
          cancel={{
            onClick: () => setShowConvertToMP4Confirm(false),
            text: intl.formatMessage({ id: "actions.cancel" }),
            variant: "secondary",
          }}
        >
          <Alert variant="warning">
            <strong>Warning:</strong>{" "}
            <FormattedMessage id="dialogs.convert_to_mp4.warning_text" />
          </Alert>
          <p>
            <FormattedMessage
              id="dialogs.convert_to_mp4.info"
              values={{ originalFormat }}
            />
          </p>
          <p>
            <strong>
              <FormattedMessage id="dialogs.convert_to_mp4.temp_path_label" />
            </strong>
            <br />
            <a
              href={`file://${tempPath}`}
              target="_blank"
              rel="noopener noreferrer"
              style={{ fontWeight: "bold", textDecoration: "underline" }}
            >
              {tempPath}
            </a>
          </p>
        </ModalComponent>
      );
    }
  }

  function maybeRenderConvertHLSToMP4ConfirmDialog() {
    if (showConvertHLSToMP4Confirm) {
      // Calculate temp path based on generatedPath (same logic as in config.go)
      const generatedPath = configuration?.general?.generatedPath || "";
      const tempPath = generatedPath
        ? generatedPath.substring(0, generatedPath.lastIndexOf("/")) + "/temp"
        : "./temp";

      return (
        <ModalComponent
          show
          icon={faVideo}
          header={intl.formatMessage({ id: "actions.convert_hls_to_mp4" })}
          accept={{
            variant: "danger",
            onClick: confirmConvertHLSToMP4,
            text: intl.formatMessage({ id: "actions.confirm" }),
          }}
          cancel={{
            onClick: () => setShowConvertHLSToMP4Confirm(false),
            text: intl.formatMessage({ id: "actions.cancel" }),
            variant: "secondary",
          }}
        >
          <Alert variant="warning">
            <strong>Warning:</strong>{" "}
            <FormattedMessage id="dialogs.convert_hls_to_mp4.warning_text" />
          </Alert>
          <p>
            <FormattedMessage id="dialogs.convert_hls_to_mp4.info" />
          </p>
          <p>
            <strong>
              <FormattedMessage id="dialogs.convert_hls_to_mp4.temp_path_label" />
            </strong>
            <br />
            <a
              href={`file://${tempPath}`}
              target="_blank"
              rel="noopener noreferrer"
              style={{ fontWeight: "bold", textDecoration: "underline" }}
            >
              {tempPath}
            </a>
          </p>
        </ModalComponent>
      );
    }
  }

  const renderOperations = () => {
    // Check if conversion section has any items
    const hasConversionOptions =
      scene.files.length > 0 &&
      (scene.files[0]?.video_codec !== "h264" ||
        scene.files[0]?.format !== "mp4" ||
        (scene.is_broken && !isHLSVideo(scene)) ||
        isHLSVideo(scene) ||
        scene.is_broken ||
        true); // reduce resolution is always available if there are files

    return (
      <Dropdown>
        <Dropdown.Toggle
          variant="secondary"
          id="operation-menu"
          className="minimal"
          title={intl.formatMessage({ id: "operations" })}
        >
          <Icon icon={faEllipsisV} />
        </Dropdown.Toggle>
        <Dropdown.Menu
          className="bg-secondary text-white"
          style={{ maxHeight: "80vh", overflowY: "auto" }}
        >
          {!!scene.files.length && (
            <Dropdown.Item
              key="rescan"
              className="bg-secondary text-white d-flex align-items-center"
              onClick={() => onRescan()}
            >
              <Icon icon={faSync} className="mr-2" />
              <FormattedMessage id="actions.rescan" />
            </Dropdown.Item>
          )}
          {!!scene.files.length && (
            <Dropdown.Item
              key="scan-threats"
              className="bg-secondary text-white d-flex align-items-center"
              onClick={async () => {
                const fileId = scene.files[0].id;
                try {
                  setScanningThreats(true);
                  await scanVideoFileThreats({ variables: { fileId } });
                  Toast.success(intl.formatMessage({ id: "toast.scan_started" }));
                  if (onSaved) {
                    setTimeout(() => onSaved(), 15000);
                  }
                } catch (e) {
                  Toast.error(e);
                } finally {
                  setScanningThreats(false);
                }
              }}
              disabled={scanningThreats}
            >
              <Icon icon={faShieldAlt} className="mr-2" />
              <FormattedMessage id="actions.scan_for_threats" defaultMessage="Scan for threats" />
            </Dropdown.Item>
          )}
          <Dropdown.Item
            key="rescan-similarity"
            className="bg-secondary text-white d-flex align-items-center"
            onClick={() => onRescanSimilarity()}
          >
            <Icon icon={faSearch} className="mr-2" />
            <FormattedMessage id="actions.rescan_similarity" />
          </Dropdown.Item>
          <Dropdown.Divider style={{ borderTopColor: "#52616d" }} />
          <Dropdown.Item
            key="generate"
            className="bg-secondary text-white d-flex align-items-center"
            onClick={() => setIsGenerateDialogOpen(true)}
          >
            <Icon icon={faCog} className="mr-2" />
            <FormattedMessage id="actions.generate" />
          </Dropdown.Item>
          <Dropdown.Item
            key="generate-screenshot"
            className="bg-secondary text-white d-flex align-items-center"
            onClick={() => onGenerateScreenshot(getPlayerPosition())}
          >
            <Icon icon={faCamera} className="mr-2" />
            <FormattedMessage id="actions.generate_thumb_from_current" />
          </Dropdown.Item>
          {scene.files.length > 0 && (
            <Dropdown.Item
              key="save-filtered-screenshot"
              className="bg-secondary text-white d-flex align-items-center"
              onClick={() => onSaveFilteredScreenshot()}
              disabled={isSavingFilteredScreenshot}
            >
              <Icon icon={faPhotoVideo} className="mr-2" />
              <FormattedMessage id="actions.make_screen_from_current" />
            </Dropdown.Item>
          )}
          <Dropdown.Item
            key="generate-default"
            className="bg-secondary text-white d-flex align-items-center"
            onClick={() => onGenerateScreenshot()}
          >
            <Icon icon={faImage} className="mr-2" />
            <FormattedMessage id="actions.generate_thumb_default" />
          </Dropdown.Item>
          {scene.files.length > 0 && (
            <Dropdown.Item
              key="regenerate-sprites"
              className="bg-secondary text-white d-flex align-items-center"
              onClick={() => setShowRegenerateSpritesModal(true)}
            >
              <Icon icon={faImages} className="mr-2" />
              <FormattedMessage id="actions.regenerate_sprites" />
            </Dropdown.Item>
          )}
          {hasConversionOptions && (
            <Dropdown.Divider style={{ borderTopColor: "#52616d" }} />
          )}
          {scene.files.length > 0 &&
            (scene.files[0]?.video_codec !== "h264" ||
              scene.files[0]?.format !== "mp4" ||
              (scene.is_broken && !isHLSVideo(scene))) && (
              <Dropdown.Item
                key="convert-to-mp4"
                className="bg-secondary text-white d-flex align-items-center"
                onClick={() => onConvertToMP4()}
              >
                <Icon icon={faVideo} className="mr-2" />
                <FormattedMessage id="actions.convert_to_mp4" />
              </Dropdown.Item>
            )}
          {scene.files.length > 0 && (isHLSVideo(scene) || scene.is_broken) && (
            <Dropdown.Item
              key="convert-hls-to-mp4"
              className="bg-secondary text-white d-flex align-items-center"
              onClick={() => onConvertHLSToMP4()}
            >
              <Icon icon={faVideo} className="mr-2" />
              <FormattedMessage id="actions.convert_hls_to_mp4" />
            </Dropdown.Item>
          )}
          {scene.files.length > 0 && (
            <Dropdown.Item
              key="reduce-resolution"
              className="bg-secondary text-white d-flex align-items-center"
              onClick={() => setShowReduceResolutionModal(true)}
            >
              <Icon icon={faCompressAlt} className="mr-2" />
              <FormattedMessage id="actions.reduce_resolution" />
            </Dropdown.Item>
          )}
          {scene.files.length > 0 &&
            (scene.start_time !== null || scene.end_time !== null) && (
              <Dropdown.Item
                key="trim-video"
                className="bg-secondary text-white d-flex align-items-center"
                onClick={() => setShowTrimVideoModal(true)}
              >
                <Icon icon={faCut} className="mr-2" />
                <FormattedMessage id="actions.trim_video" />
              </Dropdown.Item>
            )}
          {scene.files.length > 0 &&
            scene.start_time === null &&
            scene.end_time === null && (
              <OverlayTrigger
                overlay={
                  <Tooltip id="trim-video-disabled-tooltip">
                    <FormattedMessage id="dialogs.trim_video.disabled_tooltip" />
                  </Tooltip>
                }
                placement="bottom"
              >
                <span style={{ cursor: "not-allowed" }}>
                  <Dropdown.Item
                    key="trim-video-disabled"
                    className="bg-secondary text-white d-flex align-items-center"
                    style={{ opacity: 0.5, cursor: "not-allowed" }}
                    disabled
                  >
                    <Icon icon={faCut} className="mr-2" />
                    <FormattedMessage id="actions.trim_video" />
                  </Dropdown.Item>
                </span>
              </OverlayTrigger>
            )}
          {boxes.length > 0 && (
            <Dropdown.Divider style={{ borderTopColor: "#52616d" }} />
          )}
          {boxes.length > 0 && (
            <Dropdown.Item
              key="submit"
              className="bg-secondary text-white d-flex align-items-center"
              onClick={() => setShowDraftModal(true)}
            >
              <Icon icon={faUpload} className="mr-2" />
              <FormattedMessage id="actions.submit_stash_box" />
            </Dropdown.Item>
          )}
          <Dropdown.Divider style={{ borderTopColor: "#52616d" }} />
          {!scene.is_broken && (
            <Dropdown.Item
              key="set-broken"
              className="bg-secondary text-white d-flex align-items-center"
              onClick={() => onSetBroken()}
            >
              <Icon icon={faExclamationTriangle} className="mr-2" />
              <FormattedMessage id="actions.set_broken" />
            </Dropdown.Item>
          )}
          {!scene.is_not_broken && (
            <Dropdown.Item
              key="set-not-broken"
              className="bg-secondary text-white d-flex align-items-center"
              onClick={() => onSetNotBroken()}
            >
              <Icon icon={faCheckCircle} className="mr-2" />
              <FormattedMessage id="actions.set_not_broken" />
            </Dropdown.Item>
          )}
          <Dropdown.Divider style={{ borderTopColor: "#52616d" }} />
          <Dropdown.Item
            key="merge-into-other"
            className="bg-secondary text-white d-flex align-items-center"
            onClick={() => onMergeIntoOtherScene()}
          >
            <Icon icon={faExchangeAlt} className="mr-2" />
            <FormattedMessage id="actions.merge_into_other_scene" />
          </Dropdown.Item>
          <Dropdown.Item
            key="merge-from-other"
            className="bg-secondary text-white d-flex align-items-center"
            onClick={() => onMergeFromOtherScene()}
          >
            <Icon icon={faExchangeAlt} className="mr-2" />
            <FormattedMessage id="actions.merge_from_other_scene" />
          </Dropdown.Item>
          <Dropdown.Divider style={{ borderTopColor: "#52616d" }} />
          <Dropdown.Item
            key="delete-scene"
            className="bg-secondary text-white d-flex align-items-center"
            onClick={() => setIsDeleteAlertOpen(true)}
          >
            <Icon icon={faTrash} className="mr-2" />
            <FormattedMessage
              id="actions.delete"
              values={{ entityType: intl.formatMessage({ id: "scene" }) }}
            />
          </Dropdown.Item>
        </Dropdown.Menu>
      </Dropdown>
    );
  };

  const renderTabs = () => (
    <Tab.Container
      activeKey={activeTabKey}
      onSelect={(k) => k && setActiveTabKey(k)}
    >
      <div>
        <Nav variant="tabs" className="mr-auto">
          <ScenePageTabs {...props}>
            <Nav.Item>
              <Nav.Link eventKey="scene-details-panel">
                <Icon icon={faInfoCircle} className="mr-1" />
                <FormattedMessage id="details" />
              </Nav.Link>
            </Nav.Item>
            {scene.groups.length > 0 ? (
              <Nav.Item>
                <Nav.Link eventKey="scene-group-panel">
                  <Icon icon={faLayerGroup} className="mr-1" />
                  <FormattedMessage
                    id="countables.groups"
                    values={{ count: scene.groups.length }}
                  />
                </Nav.Link>
              </Nav.Item>
            ) : (
              ""
            )}
            {scene.galleries.length >= 1 ? (
              <Nav.Item>
                <Nav.Link eventKey="scene-galleries-panel">
                  <Icon icon={faImages} className="mr-1" />
                  <FormattedMessage
                    id="countables.galleries"
                    values={{ count: scene.galleries.length }}
                  />
                </Nav.Link>
              </Nav.Item>
            ) : undefined}
            <Nav.Item>
              <Nav.Link eventKey="scene-video-filter-panel">
                <span className="fa-layers fa-fw mr-1 preferences-tab-icon position-relative">
                  <Icon icon={faPlay} />
                  <span className="preferences-tab-wrench-bg">
                    <Icon icon={faWrench} transform="shrink-6" />
                  </span>
                </span>
                <FormattedMessage id="effect_filters.name" />
              </Nav.Link>
            </Nav.Item>
            <Nav.Item>
              <Nav.Link eventKey="scene-history-panel">
                <Icon icon={faHistory} className="mr-1" />
                <FormattedMessage id="history" />
              </Nav.Link>
            </Nav.Item>
            <Nav.Item>
              <Nav.Link eventKey="scene-edit-panel">
                <Icon icon={faEdit} className="mr-1" />
                <FormattedMessage id="actions.edit" />
              </Nav.Link>
            </Nav.Item>
          </ScenePageTabs>
        </Nav>
      </div>

      <Tab.Content>
        <ScenePageTabContent {...props}>
          <Tab.Pane eventKey="scene-details-panel">
            <SceneDetailPanel scene={scene} />
          </Tab.Pane>
          <Tab.Pane eventKey="scene-group-panel">
            <SceneGroupPanel scene={scene} />
          </Tab.Pane>
          {scene.galleries.length >= 1 && (
            <Tab.Pane eventKey="scene-galleries-panel">
              <SceneGalleriesPanel galleries={scene.galleries} />
              {scene.galleries.length === 1 && (
                <GalleryViewer galleryId={scene.galleries[0].id} />
              )}
            </Tab.Pane>
          )}
          <Tab.Pane eventKey="scene-video-filter-panel">
            <SceneVideoFilterPanel scene={scene} />
          </Tab.Pane>
          <Tab.Pane eventKey="scene-edit-panel" mountOnEnter>
            <SceneEditPanel
              key={scene.id}
              isVisible={activeTabKey === "scene-edit-panel"}
              scene={scene}
              onSubmit={onSave}
              onDelete={() => setIsDeleteAlertOpen(true)}
              onTagsChange={(tags) => {
                setEditingTags(tags);
              }}
            />
          </Tab.Pane>
          <Tab.Pane eventKey="scene-history-panel">
            <SceneHistoryPanel scene={scene} />
          </Tab.Pane>
        </ScenePageTabContent>
      </Tab.Content>
    </Tab.Container>
  );

  function getCollapseButtonIcon() {
    return collapsed ? faChevronRight : faChevronLeft;
  }

  const title = objectTitle(scene);

  const file = useMemo(
    () => (scene.files.length > 0 ? scene.files[0] : undefined),
    [scene]
  );

  return (
    <>
      <Helmet>
        <title>{title}</title>
      </Helmet>
      {maybeRenderSceneGenerateDialog()}
      {maybeRenderDeleteDialog()}
      {maybeRenderMergeIntoDialog()}
      {maybeRenderMergeFromDialog()}
      {maybeRenderReduceResolutionDialog()}
      {maybeRenderTrimVideoDialog()}
      {maybeRenderRegenerateSpritesDialog()}
      {maybeRenderConvertToMP4ConfirmDialog()}
      {maybeRenderConvertHLSToMP4ConfirmDialog()}
      <div
        className={`scene-tabs order-xl-first order-last ${
          collapsed ? "collapsed" : ""
        }`}
      >
        <SceneDataUpdateNotification
          visible={showDataUpdateNotification}
          onRefresh={forceRefreshSceneData}
        />
        <div>
          <div className="scene-header-container">
            {scene.studio && (
              <h1 className="text-center scene-studio-image">
                <Link to={`/studios/${scene.studio.id}`}>
                  <img
                    src={scene.studio.image_path ?? ""}
                    alt={`${scene.studio.name} logo`}
                    className="studio-logo"
                  />
                </Link>
              </h1>
            )}
            <div className="scene-header">
              {scene.force_hls ? (
                <HLSBadge />
              ) : scene.is_broken && !scene.is_not_broken ? (
                <BrokenBadge />
              ) : (
                !scene.is_broken &&
                scene.is_probably_broken &&
                !scene.is_not_broken && <ProbablyBrokenBadge />
              )}
              <h3 className={cx({ "no-studio": !scene.studio })}>{title}</h3>
            </div>
          </div>

          <div className="scene-subheader">
            {!!scene.date && (
              <span className="date mr-3" data-value={scene.date}>
                {TextUtils.formatDate(intl, scene.date, true, scene.date_display)}
              </span>
            )}
            <VideoFrameRateResolution
              width={file?.width}
              height={file?.height}
              frameRate={file?.frame_rate}
            />
            <div className="ml-auto">
              <TagRequirementsIndicator
                tags={allSceneTags}
                colorPresets={colorPresets}
              />
            </div>
          </div>

          <div className="scene-toolbar">
            <div className="scene-toolbar-row">
              <span className="scene-toolbar-group">
                <RatingSystem
                  value={scene.rating100}
                  onSetRating={setRating}
                  clickToRate
                  withoutContext
                />
              </span>
            </div>
            <div className="scene-toolbar-row">
              <span className="scene-toolbar-group">
                <span>
                  <ExternalPlayerButton scene={scene} />
                </span>
                <span>
                  <ViewCountButton
                    value={scene.play_count ?? 0}
                    onIncrement={() => incrementPlayCount()}
                    onValueClicked={() =>
                      setActiveTabKey("scene-history-panel")
                    }
                  />
                </span>
                <span>
                  <OCounterButton
                    value={scene.o_counter ?? 0}
                    onIncrement={() => onIncrementOClick()}
                  />
                </span>
                <span>
                  <OMGCounterButton
                    value={scene.omgCounter ?? 0}
                    onIncrement={() => onIncrementOMGClick()}
                    onValueClicked={() =>
                      setActiveTabKey("scene-history-panel")
                    }
                  />
                </span>
                <span>
                  <OrganizedButton
                    loading={organizedLoading}
                    organized={scene.organized}
                    onClick={onOrganizedClick}
                  />
                </span>
              </span>
              <span className="scene-toolbar-group">
                <span>{renderOperations()}</span>
              </span>
            </div>
          </div>
        </div>
        {renderTabs()}
      </div>
      <div className="scene-divider d-none d-xl-block">
        <Button onClick={() => setCollapsed(!collapsed)}>
          <Icon className="fa-fw" icon={getCollapseButtonIcon()} />
        </Button>
      </div>
      <SubmitStashBoxDraft
        type="scene"
        boxes={boxes}
        entity={scene}
        show={showDraftModal}
        onHide={() => setShowDraftModal(false)}
      />
      <SceneCountAttributeModal
        show={showOAttributeModal}
        onHide={() => setShowOAttributeModal(false)}
        type="o"
        performers={scenePerformers}
        onConfirm={handleOAttributeConfirm}
      />
      <SceneCountAttributeModal
        show={showOmgAttributeModal}
        onHide={() => setShowOmgAttributeModal(false)}
        type="omg"
        performers={scenePerformers}
        onConfirm={handleOmgAttributeConfirm}
      />
    </>
  );
});

const SceneLoader: React.FC<RouteComponentProps<ISceneParams>> = ({
  location,
  history,
  match,
}) => {
  const { id } = match.params;
  const { configuration } = useContext(ConfigurationContext);
  const { data, loading, error, refetch } = useFindScene(id);
  const [updateScene] = useSceneUpdate();
  const onVolumeChangeRef = useRef<
    (level: number, muted: boolean) => void
  >(() => {});

  // Use data directly from Apollo instead of useState
  const scene = data?.findScene;

  useEffect(() => {
    onVolumeChangeRef.current = (level: number, muted: boolean) => {
      if (!scene) return;
      updateScene({
        variables: {
          input: {
            id: scene.id,
            video_filters: {
              ...(scene.video_filters ?? {}),
              volume_level: level,
              volume_muted: muted,
            } as GQL.VideoFiltersInput,
          },
        },
      });
    };
  }, [scene, updateScene]);
  const onVolumeChangeStable = useCallback((level: number, muted: boolean) => {
    onVolumeChangeRef.current(level, muted);
  }, []);

  const queryParams = useMemo(
    () => new URLSearchParams(location.search),
    [location.search]
  );
  const sceneQueue = useMemo(
    () => SceneQueue.fromQueryParameters(queryParams),
    [queryParams]
  );
  const queryContinue = useMemo(() => {
    let cont = queryParams.get("continue");
    if (cont) {
      return cont === "true";
    } else {
      return !!configuration?.interface.continuePlaylistDefault;
    }
  }, [configuration?.interface.continuePlaylistDefault, queryParams]);

  const [queueScenes, setQueueScenes] = useState<QueuedScene[]>([]);
  const [rightSidebarPanel, setRightSidebarPanel] = useState<
    "queue" | "markers" | "fileinfo" | null
  >(null);

  // Preload right-sidebar panels so first open doesn't suspend and flicker the page
  useEffect(() => {
    void import("./QueueViewer");
    void import("./SceneMarkersPanel");
    void import("./SceneFileInfoPanel");
  }, []);

  const [collapsed, setCollapsed] = useState(false);
  const [viewedScenes, setViewedScenes] = useState<Set<string>>(new Set());

  const handleMarkSceneViewed = useCallback((sceneId: string) => {
    setViewedScenes((prev) => new Set(prev).add(sceneId));
  }, []);
  const [continuePlaylist, setContinuePlaylist] = useState(queryContinue);
  const [hideScrubber, setHideScrubber] = useState(
    !(configuration?.interface.showScrubber ?? true)
  );
  const [markerPlaylist, setMarkerPlaylist] = useState<
    GQL.SceneMarkerDataFragment[] | undefined
  >();
  const [playingTagId, setPlayingTagId] = useState<string | undefined>();
  const [initialPlaylistIndex, setInitialPlaylistIndex] = useState<number>(0);

  const _setTimestamp =
    useRef<(value: number, programmatic?: boolean) => void>();
  const initialTimestamp = useMemo(() => {
    const t = queryParams.get("t");
    if (!t) return 0;

    const n = Number(t);
    if (Number.isNaN(n)) return 0;
    return n;
  }, [queryParams]);

  const [queueTotal, setQueueTotal] = useState(0);
  const [queueStart, setQueueStart] = useState(1);

  const autoplay = queryParams.get("autoplay") === "true";
  const autoPlayOnSelected =
    configuration?.interface.autostartVideoOnPlaySelected ?? false;

  const currentQueueIndex = useMemo(
    () => queueScenes.findIndex((s) => s.id === id),
    [queueScenes, id]
  );

  function getSetTimestamp(
    fn: (value: number, programmatic?: boolean) => void
  ) {
    _setTimestamp.current = fn;
  }

  function setTimestamp(value: number, programmatic?: boolean) {
    if (_setTimestamp.current) {
      _setTimestamp.current(value, programmatic);
    }
  }

  function onPlayMarkers(markers: GQL.SceneMarkerDataFragment[]) {
    const sortedMarkers = [...markers].sort((a, b) => a.seconds - b.seconds);
    if (sortedMarkers.length === 0) return;

    const newTagId = sortedMarkers[0].primary_tag.id;
    if (playingTagId === newTagId) {
      // Already playing this tag, do nothing
      return;
    }

    // If switching from "Play All" to a specific tag, always start from the first marker
    if (playingTagId === "__ALL__") {
      setMarkerPlaylist(sortedMarkers);
      setPlayingTagId(newTagId);
      setInitialPlaylistIndex(0);
      setTimestamp(sortedMarkers[0].seconds, true);
      return;
    }

    const getMarkerEndTime = (
      marker: GQL.SceneMarkerDataFragment,
      index: number
    ) => {
      if (marker.end_seconds) {
        return marker.end_seconds;
      }
      if (index < sortedMarkers.length - 1) {
        return sortedMarkers[index + 1].seconds;
      }
      return marker.seconds + 5; // Default duration for last point marker
    };

    const currentTime = getPlayerPosition();

    if (currentTime === undefined) {
      return;
    }

    // Check if we are already inside one of the markers for this tag
    const currentMarkerIndex = sortedMarkers.findIndex((marker, index) => {
      const endTime = getMarkerEndTime(marker, index);
      return currentTime >= marker.seconds && currentTime < endTime;
    });

    if (currentMarkerIndex !== -1) {
      // We are inside a marker, just start the playlist mode without seeking
      setMarkerPlaylist(sortedMarkers);
      setPlayingTagId(sortedMarkers[currentMarkerIndex].primary_tag.id);
      setInitialPlaylistIndex(currentMarkerIndex);
    } else {
      // We are not in a marker, find the next one to play
      let nextMarkerToPlay = sortedMarkers.find(
        (m) => m.seconds >= currentTime
      );
      let startIndex = sortedMarkers.findIndex((m) => m.seconds >= currentTime);

      if (!nextMarkerToPlay) {
        // Current time is after all markers, loop back to the first one
        nextMarkerToPlay = sortedMarkers[0];
        startIndex = 0;
      }

      setMarkerPlaylist(sortedMarkers);
      setPlayingTagId(nextMarkerToPlay.primary_tag.id);
      setInitialPlaylistIndex(startIndex);
      setTimestamp(nextMarkerToPlay.seconds, true);
    }
  }

  const onPlayAllMarkers = (allMarkers: GQL.SceneMarkerDataFragment[]) => {
    const markers = allMarkers.filter((m) => m.end_seconds != null);
    if (markers.length === 0) return;

    // Sort by start time
    const sortedMarkers = [...markers].sort((a, b) => a.seconds - b.seconds);

    // Merge overlapping intervals
    const mergedMarkers: GQL.SceneMarkerDataFragment[] = [];
    if (sortedMarkers.length > 0) {
      let currentMerge = { ...sortedMarkers[0] };

      for (let i = 1; i < sortedMarkers.length; i++) {
        const nextMarker = sortedMarkers[i];
        if (nextMarker.seconds <= currentMerge.end_seconds!) {
          // Overlap or contiguous, extend the current merge
          currentMerge.end_seconds = Math.max(
            currentMerge.end_seconds!,
            nextMarker.end_seconds!
          );
        } else {
          // No overlap, push the current merge and start a new one
          mergedMarkers.push(currentMerge);
          currentMerge = { ...nextMarker };
        }
      }
      mergedMarkers.push(currentMerge);
    }

    const currentTime = getPlayerPosition();

    if (currentTime === undefined) {
      return;
    }

    let nextMarkerToPlay = mergedMarkers.find((m) => m.seconds >= currentTime);
    let startIndex = mergedMarkers.findIndex((m) => m.seconds >= currentTime);

    if (!nextMarkerToPlay) {
      nextMarkerToPlay = mergedMarkers[0];
      startIndex = 0;
    }

    setMarkerPlaylist(mergedMarkers);
    setPlayingTagId("__ALL__");
    setInitialPlaylistIndex(startIndex === -1 ? 0 : startIndex);
    setTimestamp(nextMarkerToPlay.seconds, true);
  };

  const onStopMarkers = () => {
    setMarkerPlaylist(undefined);
    setPlayingTagId(undefined);
  };

  // set up hotkeys
  useEffect(() => {
    Mousetrap.bind(".", () => setHideScrubber((value) => !value));

    return () => {
      Mousetrap.unbind(".");
    };
  }, []);

  async function getQueueFilterScenes(filter: ListFilterModel) {
    const query = await queryFindScenes(filter);
    const { scenes, count } = query.data.findScenes;
    setQueueScenes(scenes);
    setQueueTotal(count);
    setQueueStart((filter.currentPage - 1) * filter.itemsPerPage + 1);
  }

  async function getQueueScenes(sceneIDs: number[]) {
    const query = await queryFindScenesByID(sceneIDs);
    const { scenes, count } = query.data.findScenes;
    setQueueScenes(scenes);
    setQueueTotal(count);
    setQueueStart(1);
  }

  useEffect(() => {
    if (sceneQueue.query) {
      getQueueFilterScenes(sceneQueue.query);
    } else if (sceneQueue.sceneIDs) {
      getQueueScenes(sceneQueue.sceneIDs);
    }
  }, [sceneQueue]);

  async function onQueueLessScenes() {
    if (!sceneQueue.query || queueStart <= 1) {
      return;
    }

    const filterCopy = sceneQueue.query.clone();
    const newStart = queueStart - filterCopy.itemsPerPage;
    filterCopy.currentPage = Math.ceil(newStart / filterCopy.itemsPerPage);
    const query = await queryFindScenes(filterCopy);
    const { scenes } = query.data.findScenes;

    // prepend scenes to scene list
    const newScenes = (scenes as QueuedScene[]).concat(queueScenes);
    setQueueScenes(newScenes);
    setQueueStart(newStart);

    return scenes;
  }

  const queueHasMoreScenes = useMemo(() => {
    return queueStart + queueScenes.length - 1 < queueTotal;
  }, [queueStart, queueScenes, queueTotal]);

  async function onQueueMoreScenes() {
    if (!sceneQueue.query || !queueHasMoreScenes) {
      return;
    }

    const filterCopy = sceneQueue.query.clone();
    const newStart = queueStart + queueScenes.length;
    filterCopy.currentPage = Math.ceil(newStart / filterCopy.itemsPerPage);
    const query = await queryFindScenes(filterCopy);
    const { scenes } = query.data.findScenes;

    // append scenes to scene list
    const newScenes = queueScenes.concat(scenes);
    setQueueScenes(newScenes);
    // don't change queue start
    return scenes;
  }

  function loadScene(sceneID: string, autoPlay?: boolean, newPage?: number) {
    const sceneLink = sceneQueue.makeLink(sceneID, {
      newPage,
      autoPlay,
      continue: continuePlaylist,
    });
    history.replace(sceneLink);
  }

  async function queueNext(autoPlay: boolean) {
    if (currentQueueIndex === -1) return;

    if (currentQueueIndex < queueScenes.length - 1) {
      loadScene(queueScenes[currentQueueIndex + 1].id, autoPlay);
    } else {
      // if we're at the end of the queue, load more scenes
      if (currentQueueIndex === queueScenes.length - 1 && queueHasMoreScenes) {
        const loadedScenes = await onQueueMoreScenes();
        if (loadedScenes && loadedScenes.length > 0) {
          // set the page to the next page
          const newPage = (sceneQueue.query?.currentPage ?? 0) + 1;
          loadScene(loadedScenes[0].id, autoPlay, newPage);
        }
      }
    }
  }

  async function queuePrevious(autoPlay: boolean) {
    if (currentQueueIndex === -1) return;

    if (currentQueueIndex > 0) {
      loadScene(queueScenes[currentQueueIndex - 1].id, autoPlay);
    } else {
      // if we're at the beginning of the queue, load the previous page
      if (queueStart > 1) {
        const loadedScenes = await onQueueLessScenes();
        if (loadedScenes && loadedScenes.length > 0) {
          const newPage = (sceneQueue.query?.currentPage ?? 0) - 1;
          loadScene(
            loadedScenes[loadedScenes.length - 1].id,
            autoPlay,
            newPage
          );
        }
      }
    }
  }

  async function queueRandom(autoPlay: boolean) {
    if (sceneQueue.query) {
      const { query } = sceneQueue;
      const pages = Math.ceil(queueTotal / query.itemsPerPage);
      const page = Math.floor(Math.random() * pages) + 1;
      const index = Math.floor(
        Math.random() * Math.min(query.itemsPerPage, queueTotal)
      );
      const filterCopy = sceneQueue.query.clone();
      filterCopy.currentPage = page;
      const queryResults = await queryFindScenes(filterCopy);
      if (queryResults.data.findScenes.scenes.length > index) {
        const { id: sceneID } = queryResults.data.findScenes.scenes[index];
        // navigate to the image player page
        loadScene(sceneID, autoPlay, page);
      }
    } else if (queueTotal !== 0) {
      const index = Math.floor(Math.random() * queueTotal);
      loadScene(queueScenes[index].id, autoPlay);
    }
  }

  function onComplete() {
    // load the next scene if we're continuing
    if (continuePlaylist) {
      queueNext(true);
    }
  }

  function onDelete() {
    if (
      continuePlaylist &&
      currentQueueIndex >= 0 &&
      currentQueueIndex < queueScenes.length - 1
    ) {
      loadScene(queueScenes[currentQueueIndex + 1].id);
    } else {
      history.goBack();
    }
  }

  function getScenePage(sceneID: string) {
    if (!sceneQueue.query) return;

    // find the page that the scene is on
    const index = queueScenes.findIndex((s) => s.id === sceneID);

    if (index === -1) return;

    const perPage = sceneQueue.query.itemsPerPage;
    return Math.floor((index + queueStart - 1) / perPage) + 1;
  }

  function onQueueSceneClicked(sceneID: string) {
    loadScene(sceneID, autoPlayOnSelected, getScenePage(sceneID));
  }

  function onClickMarker(marker: GQL.SceneMarkerDataFragment) {
    setTimestamp(marker.seconds, true);
  }

  const intl = useIntl();

  if (!scene) {
    if (loading) return <LoadingIndicator />;
    if (error) return <ErrorMessage error={error.message} />;
    return <ErrorMessage error={`No scene found with id ${id}.`} />;
  }

  return (
    <div className="row">
      <ScenePage
        scene={scene}
        setTimestamp={setTimestamp}
        queueScenes={queueScenes}
        queueStart={queueStart}
        onDelete={onDelete}
        onQueueNext={() => queueNext(autoPlayOnSelected)}
        onQueuePrevious={() => queuePrevious(autoPlayOnSelected)}
        onQueueRandom={() => queueRandom(autoPlayOnSelected)}
        onQueueSceneClicked={onQueueSceneClicked}
        continuePlaylist={continuePlaylist}
        queueHasMoreScenes={queueHasMoreScenes}
        onQueueLessScenes={onQueueLessScenes}
        onQueueMoreScenes={onQueueMoreScenes}
        collapsed={collapsed}
        setCollapsed={setCollapsed}
        setContinuePlaylist={setContinuePlaylist}
        onPlayMarkers={onPlayMarkers}
        onStopMarkers={onStopMarkers}
        playingTagId={playingTagId}
        onPlayAllMarkers={onPlayAllMarkers}
        setRightSidebarPanel={setRightSidebarPanel}
        onSaved={async () => {
          // force refetch immediately after save to provide latest data to form
          await refetch();
        }}
      />
      <div className={`scene-player-container ${collapsed ? "expanded" : ""}`}>
        <ScenePlayer
          key="ScenePlayer"
          scene={scene}
          hideScrubberOverride={hideScrubber}
          autoplay={autoplay}
          permitLoop={!continuePlaylist}
          initialTimestamp={initialTimestamp}
          sendSetTimestamp={getSetTimestamp}
          onComplete={onComplete}
          onNext={() => queueNext(true)}
          onPrevious={() => queuePrevious(true)}
          onPlayScene={(sceneId) => loadScene(sceneId, true)}
          viewedScenes={viewedScenes}
          onMarkSceneViewed={handleMarkSceneViewed}
          markerPlaylist={markerPlaylist}
          initialPlaylistIndex={initialPlaylistIndex}
          onClearMarkerPlaylist={onStopMarkers}
          initialVolume={scene.video_filters?.volume_level ?? 1}
          initialMuted={scene.video_filters?.volume_muted ?? false}
          onVolumeChange={onVolumeChangeStable}
        />
      </div>
      <div className="scene-right-edge-panel">
        <div className="scene-right-ears">
          {queueScenes.length > 0 && (
            <button
              type="button"
              className={`scene-ear ${rightSidebarPanel === "queue" ? "active" : ""}`}
              title={intl.formatMessage({ id: "queue" })}
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                setRightSidebarPanel((p) => (p === "queue" ? null : "queue"));
              }}
            >
              <Icon icon={faListUl} />
            </button>
          )}
          <button
            type="button"
            className={`scene-ear ${rightSidebarPanel === "markers" ? "active" : ""}`}
            title={intl.formatMessage({ id: "markers" })}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              setRightSidebarPanel((p) =>
                p === "markers" ? null : "markers"
              );
            }}
          >
            <Icon icon={faMapMarkerAlt} />
          </button>
          <button
            type="button"
            className={`scene-ear ${rightSidebarPanel === "fileinfo" ? "active" : ""}`}
            title={intl.formatMessage({ id: "file_info" })}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              setRightSidebarPanel((p) =>
                p === "fileinfo" ? null : "fileinfo"
              );
            }}
          >
            <Icon icon={faFileAlt} />
          </button>
        </div>
        <div
          className={`scene-right-sidebar ${rightSidebarPanel ? "open" : ""}`}
        >
          {rightSidebarPanel && (
            <>
              <Button
                variant="link"
                className="scene-right-sidebar-close"
                onClick={() => setRightSidebarPanel(null)}
                title={intl.formatMessage({ id: "actions.close" })}
              >
                <Icon icon={faTimes} />
              </Button>
              <div className="scene-right-sidebar-content">
                <Suspense fallback={<LoadingIndicator />}>
                  {rightSidebarPanel === "queue" && (
                    <QueueViewer
                      scenes={queueScenes}
                      currentID={scene.id}
                      continue={continuePlaylist}
                      setContinue={setContinuePlaylist}
                      onSceneClicked={onQueueSceneClicked}
                      onNext={() => queueNext(autoPlayOnSelected)}
                      onPrevious={() => queuePrevious(autoPlayOnSelected)}
                      onRandom={() => queueRandom(autoPlayOnSelected)}
                      start={queueStart}
                      hasMoreScenes={queueHasMoreScenes}
                      onLessScenes={onQueueLessScenes}
                      onMoreScenes={onQueueMoreScenes}
                    />
                  )}
                  {rightSidebarPanel === "markers" && (
                    <SceneMarkersPanel
                      sceneId={scene.id}
                      onClickMarker={onClickMarker}
                      onPlayMarkers={onPlayMarkers}
                      onStopMarkers={onStopMarkers}
                      playingTagId={playingTagId}
                      onPlayAllMarkers={onPlayAllMarkers}
                      isVisible
                    />
                  )}
                  {rightSidebarPanel === "fileinfo" && (
                    <SceneFileInfoPanel
                      scene={scene}
                      onRefetch={async () => {
                        await refetch();
                      }}
                    />
                  )}
                </Suspense>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default SceneLoader;
