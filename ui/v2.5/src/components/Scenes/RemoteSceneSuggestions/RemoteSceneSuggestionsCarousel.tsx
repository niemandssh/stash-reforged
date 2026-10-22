import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Button } from "react-bootstrap";
import { faSync } from "@fortawesome/free-solid-svg-icons";
import Slider from "@ant-design/react-slick";
import { useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { ConfigurationContext } from "src/hooks/Config";
import {
  mutateStartRemoteSceneSuggestions,
  mutateStopJob,
  queryRemoteSceneSuggestionResults,
} from "src/core/StashService";
import { getSlickSliderSettings } from "src/core/recommendations";
import { Icon } from "src/components/Shared/Icon";
import { RecommendationRow } from "../../FrontPage/RecommendationRow";
import RemoteSceneSuggestionCard from "./RemoteSceneSuggestionCard";
import {
  ConfiguredTagRef,
  getPreferredTagHighlightColor,
} from "./tagMatching";
import { JobFragment, useMonitorJob } from "src/utils/job";

export type RemoteSuggestionsContext =
  | {
      type: "tag";
      tag: Pick<GQL.TagDataFragment, "id" | "name" | "aliases">;
    }
  | {
      type: "performer";
      performer: Pick<GQL.PerformerDataFragment, "id" | "name" | "disambiguation">;
    };

type RemoteSceneSuggestionsCarouselProps = {
  context?: RemoteSuggestionsContext;
};

type ProviderKey = "pornhub" | "xvideos" | "xnxx" | "xhamster";

const DEFAULT_PROVIDERS: ProviderKey[] = [
  "pornhub",
  "xvideos",
  "xnxx",
  "xhamster",
];

const DEFAULT_LIMIT = 12;

const PROVIDERS: Record<
  ProviderKey,
  {
    label: string;
    urlPatterns: string[];
  }
> = {
  pornhub: {
    label: "PornHub",
    urlPatterns: ["pornhub.com", "pornhub.org"],
  },
  xvideos: {
    label: "XVideos",
    urlPatterns: ["xvideos.com", "xvideos.red"],
  },
  xnxx: {
    label: "Xnxx",
    urlPatterns: ["xnxx.com"],
  },
  xhamster: {
    label: "XHamster",
    urlPatterns: ["xhamster.com"],
  },
};

type RemoteSuggestion = {
  providerLabel: string;
  scene: GQL.ScrapedSceneDataFragment;
  priority: boolean;
  priorityColor: string | null;
};

function isTouchEnabled() {
  return "ontouchstart" in window || navigator.maxTouchPoints > 0;
}

function providerLabelForScene(scene: GQL.ScrapedSceneDataFragment): string {
  const url = (scene.urls?.[0] ?? scene.url ?? "").toLowerCase();
  for (const [key, config] of Object.entries(PROVIDERS) as [
    ProviderKey,
    (typeof PROVIDERS)[ProviderKey]
  ][]) {
    if (config.urlPatterns.some((pattern) => url.includes(pattern))) {
      return config.label;
    }
    void key;
  }
  return "Remote";
}

const RemoteSceneSuggestionsCarousel: React.FC<
  RemoteSceneSuggestionsCarouselProps
> = ({ context }) => {
  const intl = useIntl();
  const { configuration } = useContext(ConfigurationContext);

  const remoteEnabled = configuration?.ui?.remoteSceneSuggestionsEnabled ?? true;
  const minDurationMinutes =
    configuration?.ui?.remoteSceneSuggestionsMinDurationMinutes ?? 10;
  const suggestionsLimit =
    configuration?.ui?.remoteSceneSuggestionsLimit ?? DEFAULT_LIMIT;
  const enabledProviders = useMemo(
    () =>
      (configuration?.ui?.remoteSceneSuggestionsProviders as
        | ProviderKey[]
        | undefined) ?? DEFAULT_PROVIDERS,
    [configuration?.ui?.remoteSceneSuggestionsProviders]
  );

  const priorityTagIds = useMemo(
    () => configuration?.ui?.remoteSceneSuggestionsPriorityTagIds ?? [],
    [configuration?.ui?.remoteSceneSuggestionsPriorityTagIds]
  );

  const stopTagIds = useMemo(
    () => configuration?.ui?.remoteSceneSuggestionsStopTagIds ?? [],
    [configuration?.ui?.remoteSceneSuggestionsStopTagIds]
  );

  const configuredTagIds = useMemo(
    () => Array.from(new Set([...priorityTagIds, ...stopTagIds])),
    [priorityTagIds, stopTagIds]
  );

  const priorityTagIdSignature = useMemo(
    () => priorityTagIds.join(","),
    [priorityTagIds]
  );

  const stopTagIdSignature = useMemo(
    () => stopTagIds.join(","),
    [stopTagIds]
  );

  const { data: configuredTagsData } = GQL.useFindTagsForSelectQuery({
    variables: {
      ids: configuredTagIds,
      filter: { per_page: configuredTagIds.length || 1 },
    },
    skip: configuredTagIds.length === 0,
  });

  const priorityTagRefs = useMemo<ConfiguredTagRef[]>(() => {
    const tags = configuredTagsData?.findTags?.tags ?? [];
    return tags.filter((tag) => priorityTagIds.includes(tag.id));
  }, [configuredTagsData, priorityTagIds]);

  const priorityTagsById = useMemo(() => {
    const map = new Map<string, ConfiguredTagRef>();
    for (const tag of priorityTagRefs) {
      map.set(tag.id, tag);
    }
    return map;
  }, [priorityTagRefs]);

  const providerList = useMemo(
    () => enabledProviders.filter((p) => p in PROVIDERS),
    [enabledProviders]
  );

  const providerKeySignature = useMemo(
    () => providerList.join(","),
    [providerList]
  );

  const [loading, setLoading] = useState(false);
  const [suggestions, setSuggestions] = useState<RemoteSuggestion[]>([]);
  const [reloadKey, setReloadKey] = useState(0);
  const [jobId, setJobId] = useState<string | null>(null);
  const fetchingResultsRef = useRef(false);
  const activeJobIdRef = useRef<string | null>(null);

  const contextSignature = useMemo(() => {
    if (context?.type === "tag") {
      return `tag:${context.tag.id}`;
    }
    if (context?.type === "performer") {
      return `performer:${context.performer.id}`;
    }
    return "";
  }, [
    context?.type,
    context?.type === "tag" ? context.tag.id : undefined,
    context?.type === "performer" ? context.performer.id : undefined,
  ]);

  const contextTagId =
    context?.type === "tag" ? context.tag.id : undefined;
  const contextPerformerId =
    context?.type === "performer" ? context.performer.id : undefined;

  const isTouch = useMemo(() => isTouchEnabled(), []);

  const handleJobComplete = useCallback(
    async (job?: JobFragment) => {
      const completedJobId = activeJobIdRef.current;
      if (!completedJobId || fetchingResultsRef.current) {
        return;
      }

      if (
        job?.status === GQL.JobStatus.Failed ||
        job?.status === GQL.JobStatus.Cancelled
      ) {
        setSuggestions([]);
        setLoading(false);
        setJobId(null);
        activeJobIdRef.current = null;
        return;
      }

      fetchingResultsRef.current = true;
      try {
        const result = await queryRemoteSceneSuggestionResults(completedJobId);
        const scenes = result.data?.remoteSceneSuggestionResults ?? [];
        const mapped: RemoteSuggestion[] = scenes.map((scene) => {
          const priorityColor = getPreferredTagHighlightColor(
            priorityTagIds,
            priorityTagsById,
            scene.tags ?? [],
            { sceneTitle: scene.title, matchTitle: true }
          );
          return {
            scene,
            providerLabel: providerLabelForScene(scene),
            priority: !!priorityColor,
            priorityColor,
          };
        });

        const priority = mapped.filter((item) => item.priority);
        const regular = mapped.filter((item) => !item.priority);
        setSuggestions([...priority, ...regular]);
      } catch {
        setSuggestions([]);
      } finally {
        fetchingResultsRef.current = false;
        setLoading(false);
        setJobId(null);
        activeJobIdRef.current = null;
      }
    },
    [priorityTagIds, priorityTagsById]
  );

  useMonitorJob(jobId, handleJobComplete);

  useEffect(() => {
    let cancelled = false;

    async function run() {
      if (!remoteEnabled) {
        setSuggestions([]);
        return;
      }

      if (providerList.length === 0) {
        setSuggestions([]);
        return;
      }

      if (activeJobIdRef.current) {
        try {
          await mutateStopJob(activeJobIdRef.current);
        } catch {
          // ignore
        }
        activeJobIdRef.current = null;
        setJobId(null);
      }

      setLoading(true);
      setSuggestions([]);
      fetchingResultsRef.current = false;

      const input: GQL.RemoteSceneSuggestionsInput = {
        providers: providerList,
        min_duration_minutes: minDurationMinutes,
        limit: suggestionsLimit,
        priority_tag_ids: priorityTagIds,
        stop_tag_ids: stopTagIds,
      };

      if (contextTagId) {
        input.context_tag_id = contextTagId;
      } else if (contextPerformerId) {
        input.context_performer_id = contextPerformerId;
      }

      try {
        const result = await mutateStartRemoteSceneSuggestions(input);
        const startedJobId = result.data?.startRemoteSceneSuggestions;
        if (cancelled) {
          if (startedJobId) {
            try {
              await mutateStopJob(startedJobId);
            } catch {
              // ignore
            }
          }
          return;
        }
        if (!startedJobId) {
          setLoading(false);
          return;
        }
        activeJobIdRef.current = startedJobId;
        setJobId(startedJobId);
      } catch {
        if (!cancelled) {
          setLoading(false);
          setSuggestions([]);
        }
      }
    }

    run();

    return () => {
      cancelled = true;
      const currentJobId = activeJobIdRef.current;
      if (currentJobId) {
        mutateStopJob(currentJobId).catch(() => undefined);
        activeJobIdRef.current = null;
      }
    };
  }, [
    remoteEnabled,
    providerKeySignature,
    minDurationMinutes,
    suggestionsLimit,
    priorityTagIdSignature,
    stopTagIdSignature,
    contextSignature,
    contextTagId,
    contextPerformerId,
    reloadKey,
    providerList,
    priorityTagIds,
    stopTagIds,
  ]);

  const displaySuggestions = useMemo(() => {
    const mapped = suggestions.map((item) => {
      const priorityColor = getPreferredTagHighlightColor(
        priorityTagIds,
        priorityTagsById,
        item.scene.tags ?? [],
        { sceneTitle: item.scene.title, matchTitle: true }
      );
      return {
        ...item,
        priority: !!priorityColor,
        priorityColor,
      };
    });
    const priority = mapped.filter((item) => item.priority);
    const regular = mapped.filter((item) => !item.priority);
    return [...priority, ...regular];
  }, [suggestions, priorityTagIds, priorityTagsById]);

  const itemsToShow =
    displaySuggestions.length > 0
      ? displaySuggestions.length
      : Math.max(1, suggestionsLimit);

  const sliderSettings = useMemo(() => {
    const settings = getSlickSliderSettings(itemsToShow, isTouch);
    const withOneScroll = {
      slidesToScroll: 1,
      swipeToSlide: true,
      touchThreshold: 10,
      waitForAnimate: false,
      draggable: true,
      swipe: true,
    };

    return {
      ...settings,
      ...withOneScroll,
      responsive: settings.responsive.map((breakpoint) => ({
        ...breakpoint,
        settings: {
          ...breakpoint.settings,
          ...withOneScroll,
        },
      })),
    };
  }, [itemsToShow, isTouch]);

  if (!remoteEnabled) return null;
  if (providerList.length === 0) return null;
  if (!loading && displaySuggestions.length === 0) return null;

  const header = intl.formatMessage({
    id: "config.ui.remote_scene_suggestions.heading",
    defaultMessage: "Remote scene suggestions",
  });

  const getMoreLabel = intl.formatMessage({
    id: "config.ui.remote_scene_suggestions.get_more",
    defaultMessage: "Get more scenes",
  });

  return (
    <RecommendationRow
      className="remote-scene-suggestions"
      header={header}
      link={
        <Button
          variant="primary"
          size="sm"
          className="remote-scene-suggestions-refresh"
          disabled={loading}
          onClick={() => setReloadKey((k) => k + 1)}
        >
          <Icon icon={faSync} className="fa-fw" />
          <span>{getMoreLabel}</span>
        </Button>
      }
    >
      <Slider {...sliderSettings}>
        {loading
          ? [...Array(itemsToShow)].map((_, i) => (
              <div
                key={`remote_suggestion_skeleton_${i}`}
                className="remote-scene-suggestion-skeleton skeleton-card"
              />
            ))
          : displaySuggestions.map((s, i) => (
              <div
                key={`${s.scene.urls?.[0] ?? s.scene.url ?? i}`}
                className="remote-scene-suggestion-slide"
              >
                <RemoteSceneSuggestionCard
                  scene={s.scene}
                  providerLabel={s.providerLabel}
                  priority={s.priority}
                  priorityColor={s.priorityColor}
                />
              </div>
            ))}
      </Slider>
    </RecommendationRow>
  );
};

export default RemoteSceneSuggestionsCarousel;
