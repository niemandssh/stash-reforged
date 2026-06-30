import React, { useEffect, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { ModalComponent } from "src/components/Shared/Modal";
import { useToast } from "src/hooks/Toast";
import {
  ensureSceneAIVisionPanels,
  queryFillSceneWithAI,
} from "src/core/StashService";
import { waitForJob } from "src/utils/waitForJob";
import { lazyComponent } from "src/utils/lazyComponent";
import { Tag } from "src/components/Tags/TagSelect";
import { Performer } from "src/components/Performers/PerformerSelect";
import { Studio } from "src/components/Studios/StudioSelect";
import { Group } from "src/components/Groups/GroupSelect";
import { applyAIFilledScene } from "./applyAIFilledScene";

const SceneScrapeDialog = lazyComponent(() => import("./SceneScrapeDialog"));

interface IFillSceneAIDialogProps {
  scene: GQL.SceneDataFragment;
  onClose: () => void;
  onApplied?: () => void;
}

export const FillSceneAIDialog: React.FC<IFillSceneAIDialogProps> = ({
  scene,
  onClose,
  onApplied,
}) => {
  const intl = useIntl();
  const Toast = useToast();
  const [loading, setLoading] = useState(true);
  const [loadingPhase, setLoadingPhase] = useState<"sprites" | "thinking">(
    "sprites"
  );
  const [error, setError] = useState<string | null>(null);
  const [aiResult, setAiResult] = useState<GQL.FillSceneWithAiQuery | null>(
    null
  );
  const [scrapedScene, setScrapedScene] = useState<GQL.ScrapedScene | null>(
    null
  );

  useEffect(() => {
    let cancelled = false;

    async function fetchAI() {
      try {
        setLoadingPhase("sprites");
        const jobID = await ensureSceneAIVisionPanels(scene.id);
        await waitForJob(jobID);

        if (cancelled) return;

        setLoadingPhase("thinking");
        const result = await queryFillSceneWithAI(scene.id);
        if (cancelled) return;
        setAiResult(result.data ?? null);
        setScrapedScene(result.data?.fillSceneWithAI?.scene ?? null);
      } catch (e) {
        if (!cancelled) {
          setError(String(e));
          Toast.error(e);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchAI();

    return () => {
      cancelled = true;
    };
  }, [scene.id, Toast]);

  useEffect(() => {
    if (!loading && (error || !aiResult?.fillSceneWithAI || !scrapedScene)) {
      onClose();
    }
  }, [loading, error, aiResult, scrapedScene, onClose]);

  if (!loading && aiResult?.fillSceneWithAI && scrapedScene) {
    const sceneStudio: Studio | null = scene.studio
      ? {
          id: scene.studio.id,
          name: scene.studio.name,
          aliases: [],
        }
      : null;

    const scenePerformers: Performer[] = (scene.performers ?? []).map((p) => ({
      id: p.id,
      name: p.name,
      alias_list: p.alias_list ?? [],
      small_role: false,
    }));

    const performerTagIds = new Set<string>();
    if (scene.performer_tag_ids) {
      scene.performer_tag_ids.forEach((pt) => {
        pt.tag_ids?.forEach((id) => performerTagIds.add(id));
      });
    }

    const sceneTags: Tag[] = (scene.tags ?? [])
      .filter((t) => !performerTagIds.has(t.id))
      .map((t) => ({
        id: t.id,
        name: t.name,
        sort_name: t.sort_name,
        aliases: t.aliases,
        image_path: t.image_path,
        is_pose_tag: t.is_pose_tag,
        color: t.color,
      }));

    const sceneGroups: Group[] = (scene.groups ?? []).map((g) => ({
      id: g.group.id,
      name: g.group.name,
    }));

    const currentScene: Partial<GQL.SceneUpdateInput> = {
      id: scene.id,
      title: scene.title ?? undefined,
      code: scene.code ?? undefined,
      details: scene.details ?? undefined,
      director: scene.director ?? undefined,
      urls: scene.urls,
      date: scene.date ?? undefined,
      shoot_date: scene.shoot_date ?? undefined,
    };

    async function onScrapeDialogClosed(
      selected?: GQL.ScrapedSceneDataFragment
    ) {
      if (selected && aiResult?.fillSceneWithAI) {
        try {
          await applyAIFilledScene(
            scene.id,
            selected,
            aiResult.fillSceneWithAI
          );
          Toast.success(
            <FormattedMessage id="toast.scene_ai_fill_applied" />
          );
          onApplied?.();
        } catch (e) {
          Toast.error(e);
        }
      }
      onClose();
    }

    return (
      <SceneScrapeDialog
        scene={currentScene}
        sceneStudio={sceneStudio}
        scenePerformers={scenePerformers}
        sceneTags={sceneTags}
        sceneGroups={sceneGroups}
        scraped={scrapedScene}
        onClose={(s) => onScrapeDialogClosed(s)}
      />
    );
  }

  if (!loading) {
    return null;
  }

  return (
    <ModalComponent
      show
      onHide={onClose}
      header={intl.formatMessage({ id: "actions.fill_scene_data_with_ai" })}
      accept={{
        text: intl.formatMessage({ id: "actions.cancel" }),
        onClick: onClose,
        variant: "secondary",
      }}
      isRunning
    >
      <div className="m-4 text-center">
        <LoadingIndicator
          inline
          message={
            <FormattedMessage
              id={
                loadingPhase === "sprites"
                  ? "dialogs.fill_scene_ai.generating_sprites"
                  : "dialogs.fill_scene_ai.thinking"
              }
            />
          }
        />
      </div>
    </ModalComponent>
  );
};
