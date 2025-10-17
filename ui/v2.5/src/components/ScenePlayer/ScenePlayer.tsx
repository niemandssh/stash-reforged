import React, {
  KeyboardEvent,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import videojs, { VideoJsPlayer, VideoJsPlayerOptions } from "video.js";
import useScript from "src/hooks/useScript";
import "videojs-contrib-dash";
import "videojs-mobile-ui";
import "videojs-seek-buttons";
import { UAParser } from "ua-parser-js";
import "./live";
import "./PlaylistButtons";
import "./source-selector";
import "./persist-volume";
import MarkersPlugin, { type IMarker } from "./markers";
void MarkersPlugin;
import "./vtt-thumbnails";
import "./big-buttons";
import "./track-activity";
import "./vrmode";
import cx from "classnames";
import {
  useSceneSaveActivity,
  useSceneIncrementPlayCount,
} from "src/core/StashService";
import { useTrimContext } from "src/contexts/TrimContext";

import * as GQL from "src/core/generated-graphql";
import { ScenePlayerScrubber } from "./ScenePlayerScrubber";
import { ConfigurationContext } from "src/hooks/Config";
import { NextSceneOverlay } from "./NextSceneOverlay";
import { useNextScene } from "src/utils/nextSceneSelector";
import {
  ConnectionState,
  InteractiveContext,
} from "src/hooks/Interactive/context";
import { SceneInteractiveStatus } from "src/hooks/Interactive/status";
import { languageMap } from "src/utils/caption";
import { VIDEO_PLAYER_ID } from "./util";
import { useQuery } from "@apollo/client";

// @ts-ignore
import airplay from "@silvermine/videojs-airplay";
// @ts-ignore
import chromecast from "@silvermine/videojs-chromecast";
import abLoopPlugin from "videojs-abloop";
import ScreenUtils from "src/utils/screen";
import { PatchComponent } from "src/patch";

// register videojs plugins
try {
  airplay(videojs);
} catch (e) {
  console.warn('AirPlay plugin failed to load:', e);
}
chromecast(videojs);
abLoopPlugin(window, videojs);

function handleHotkeys(player: VideoJsPlayer, event: videojs.KeyboardEvent) {
  function seekStep(step: number) {
    const time = player.currentTime() + step;
    const duration = player.duration();
    if (time < 0) {
      player.currentTime(0);
    } else if (time < duration) {
      player.currentTime(time);
    } else {
      player.currentTime(duration);
    }
  }

  function seekPercent(percent: number) {
    const duration = player.duration();
    const time = duration * percent;
    player.currentTime(time);
  }

  function seekPercentRelative(percent: number) {
    const duration = player.duration();
    const currentTime = player.currentTime();
    const time = currentTime + duration * percent;
    if (time > duration) return;
    player.currentTime(time);
  }

  function toggleABLooping() {
    const opts = player.abLoopPlugin.getOptions();
    if (!opts.start) {
      opts.start = player.currentTime();
    } else if (!opts.end) {
      opts.end = player.currentTime();
      opts.enabled = true;
    } else {
      opts.start = 0;
      opts.end = 0;
      opts.enabled = false;
    }
    player.abLoopPlugin.setOptions(opts);
  }

  let seekFactor = 10;
  if (event.shiftKey) {
    seekFactor = 5;
  } else if (event.ctrlKey || event.altKey) {
    seekFactor = 60;
  }
  switch (event.which) {
    case 39: // right arrow
      seekStep(seekFactor);
      break;
    case 37: // left arrow
      seekStep(-seekFactor);
      break;
  }

  // toggle player looping with shift+l
  if (event.shiftKey && event.which === 76) {
    player.loop(!player.loop());
    return;
  }

  if (event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) {
    return;
  }

  const skipButtons = player.skipButtons();
  if (skipButtons) {
    // handle multimedia keys
    switch (event.key) {
      case "MediaTrackNext":
        if (!skipButtons.onNext) return;
        skipButtons.onNext();
        break;
      case "MediaTrackPrevious":
        if (!skipButtons.onPrevious) return;
        skipButtons.onPrevious();
        break;
      // MediaPlayPause handled by videojs
    }
  }

  switch (event.which) {
    case 32: // space
    case 13: // enter
      if (player.paused()) player.play();
      else player.pause();
      break;
    case 77: // m
      player.muted(!player.muted());
      break;
    case 70: // f
      if (player.isFullscreen()) player.exitFullscreen();
      else player.requestFullscreen();
      break;
    case 76: // l
      toggleABLooping();
      break;
    case 38: // up arrow
      player.volume(player.volume() + 0.1);
      break;
    case 40: // down arrow
      player.volume(player.volume() - 0.1);
      break;
    case 48: // 0
      player.currentTime(0);
      break;
    case 49: // 1
      seekPercent(0.1);
      break;
    case 50: // 2
      seekPercent(0.2);
      break;
    case 51: // 3
      seekPercent(0.3);
      break;
    case 52: // 4
      seekPercent(0.4);
      break;
    case 53: // 5
      seekPercent(0.5);
      break;
    case 54: // 6
      seekPercent(0.6);
      break;
    case 55: // 7
      seekPercent(0.7);
      break;
    case 56: // 8
      seekPercent(0.8);
      break;
    case 57: // 9
      seekPercent(0.9);
      break;
    case 221: // ]
      seekPercentRelative(0.1);
      break;
    case 219: // [
      seekPercentRelative(-0.1);
      break;
  }
}

type MarkerFragment = Pick<GQL.SceneMarker, "title" | "seconds"> & {
  primary_tag: Pick<GQL.Tag, "name">;
  tags: Array<Pick<GQL.Tag, "name">>;
};

function getMarkerTitle(marker: MarkerFragment) {
  if (marker.title) {
    return marker.title;
  }

  let ret = marker.primary_tag.name;
  if (marker.tags.length) {
    ret += `, ${marker.tags.map((t) => t.name).join(", ")}`;
  }

  return ret;
}

interface IScenePlayerProps {
  scene: GQL.SceneDataFragment;
  hideScrubberOverride: boolean;
  autoplay?: boolean;
  permitLoop?: boolean;
  initialTimestamp: number;
  sendSetTimestamp: (setTimestamp: (value: number) => void) => void;
  onComplete: () => void;
  onNext: () => void;
  onPrevious: () => void;
  onPlayScene?: (sceneId: string) => void;
  viewedScenes?: Set<string>;
  onMarkSceneViewed?: (sceneId: string) => void;
}

export const ScenePlayer: React.FC<IScenePlayerProps> = PatchComponent(
  "ScenePlayer",
  ({
    scene,
    hideScrubberOverride,
    autoplay,
    permitLoop = true,
    initialTimestamp: _initialTimestamp,
    sendSetTimestamp,
    onComplete,
    onNext,
    onPrevious,
    onPlayScene,
    viewedScenes = new Set(),
    onMarkSceneViewed,
  }) => {
    const { configuration } = useContext(ConfigurationContext);
    const interfaceConfig = configuration?.interface;
    const uiConfig = configuration?.ui;
    const videoRef = useRef<HTMLDivElement>(null);
    const [_player, setPlayer] = useState<VideoJsPlayer>();
    const sceneId = useRef<string>();
    const [sceneSaveActivity] = useSceneSaveActivity();
    const [sceneIncrementPlayCount] = useSceneIncrementPlayCount();

    const [time, setTime] = useState(0);
    const [ready, setReady] = useState(false);
    const [isInTrimmedSegment, setIsInTrimmedSegment] = useState(false);
    const { trimEnabled } = useTrimContext();
    const [pausedByTrim, setPausedByTrim] = useState(false);
    const previousTime = useRef<number>(0);
    const lastSeekTime = useRef<number>(0);
    const isSeeking = useRef<boolean>(false);

    const {
      interactive: interactiveClient,
      uploadScript,
      currentScript,
      initialised: interactiveInitialised,
      state: interactiveState,
    } = React.useContext(InteractiveContext);

    const [fullscreen, setFullscreen] = useState(false);
    const [showScrubber, setShowScrubber] = useState(false);
    const [showNextSceneOverlay, setShowNextSceneOverlay] = useState(false);

    const started = useRef(false);
    const auto = useRef(false);
    const interactiveReady = useRef(false);
    const minimumPlayPercent = uiConfig?.minimumPlayPercent ?? 0;

    // Query for similar scenes to determine next scene for autoplay
    // Get enough scenes to handle multiple batches when some are already viewed
    const { data: similarScenesData } = useQuery<GQL.FindSimilarScenesQuery, GQL.FindSimilarScenesQueryVariables>(
      GQL.FindSimilarScenesDocument,
      {
        variables: { id: scene.id, limit: 50 }, // Get more scenes to handle viewed scenes
        skip: !scene.id,
      }
    );

    const nextScene = useNextScene(scene.id, similarScenesData?.findScene?.similar_scenes, Array.from(viewedScenes as Set<string>));
    const trackActivity = uiConfig?.trackActivity ?? true;
    const vrTag = uiConfig?.vrTag ?? undefined;

    // Mark current scene as viewed when component mounts or scene changes
    useEffect(() => {
      if (scene.id && onMarkSceneViewed) {
        onMarkSceneViewed(scene.id);
      }
    }, [scene.id, onMarkSceneViewed]);

    useScript(
      "https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1",
      uiConfig?.enableChromecast
    );

    const file = useMemo(
      () => (scene.files.length > 0 ? scene.files[0] : undefined),
      [scene]
    );

    const maxLoopDuration = interfaceConfig?.maximumLoopDuration ?? 0;
    const looping = useMemo(
      () =>
        !!file?.duration &&
        permitLoop &&
        maxLoopDuration !== 0 &&
        file.duration < maxLoopDuration,
      [file, permitLoop, maxLoopDuration]
    );

    const getPlayer = useCallback(() => {
      if (!_player) return null;
      if (_player.isDisposed()) return null;
      return _player;
    }, [_player]);

    // Initialize previousTime when player is ready
    useEffect(() => {
      const player = getPlayer();
      if (player) {
        previousTime.current = player.currentTime();
      }
    }, [getPlayer]);

    useEffect(() => {
      if (hideScrubberOverride || fullscreen) {
        setShowScrubber(false);
        return;
      }

      const onResize = () => {
        const show = window.innerHeight >= 450 && !ScreenUtils.isMobile();
        setShowScrubber(show);
      };
      onResize();

      window.addEventListener("resize", onResize);

      return () => window.removeEventListener("resize", onResize);
    }, [hideScrubberOverride, fullscreen]);

    useEffect(() => {
      sendSetTimestamp((value: number) => {
        const player = getPlayer();
        if (player && value >= 0) {
          if (player.hasStarted() && player.paused()) {
            player.currentTime(value);
          } else {
            player.play()?.then(() => {
              player.currentTime(value);
            });
          }
        }
      });
    }, [sendSetTimestamp, getPlayer]);

    // Initialize VideoJS player
    useEffect(() => {
      const options: VideoJsPlayerOptions = {
        id: VIDEO_PLAYER_ID,
        controls: true,
        controlBar: {
          pictureInPictureToggle: false,
          volumePanel: {
            inline: false,
          },
          chaptersButton: false,
        },
        html5: {
          dash: {
            updateSettings: [
              {
                streaming: {
                  buffer: {
                    bufferTimeAtTopQuality: 30,
                    bufferTimeAtTopQualityLongForm: 30,
                  },
                  gaps: {
                    jumpGaps: false,
                    jumpLargeGaps: false,
                  },
                },
              },
            ],
          },
        },
        nativeControlsForTouch: false,
        playbackRates: [0.25, 0.5, 0.75, 1, 1.25, 1.5, 1.75, 2],
        inactivityTimeout: 2000,
        preload: "none",
        playsinline: true,
        techOrder: ["chromecast", "html5"],
        userActions: {
          hotkeys: function (this: VideoJsPlayer, event) {
            handleHotkeys(this, event);
          },
        },
        plugins: {
          airPlay: {},
          chromecast: {},
          vttThumbnails: {
            showTimestamp: true,
          },
          markers: {},
          sourceSelector: {},
          persistVolume: {},
          bigButtons: {},
          seekButtons: {
            forward: 10,
            back: 10,
          },
          skipButtons: {},
          trackActivity: {},
          vrMenu: {},
          abLoopPlugin: {
            start: 0,
            end: false,
            enabled: false,
            loopIfBeforeStart: true,
            loopIfAfterEnd: true,
            pauseAfterLooping: false,
            pauseBeforeLooping: false,
            createButtons: uiConfig?.showAbLoopControls ?? false,
          },
        },
      };

      const videoEl = document.createElement("video-js");
      videoEl.setAttribute("data-vjs-player", "true");
      videoEl.setAttribute("crossorigin", "anonymous");
      videoEl.classList.add("vjs-big-play-centered");
      videoRef.current!.appendChild(videoEl);

      const vjs = videojs(videoEl, options);

      /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
      const settings = (vjs as any).textTrackSettings;
      settings.setValues({
        backgroundColor: "#000",
        backgroundOpacity: "0.5",
      });
      settings.updateDisplay();

      vjs.focus();
      setPlayer(vjs);

      // Video player destructor
      return () => {
        vjs.dispose();
        videoEl.remove();
        setPlayer(undefined);

        // reset sceneId to force reload sources
        sceneId.current = undefined;
      };
      // empty deps - only init once
      // showAbLoopControls is necessary to re-init the player when the config changes
    }, [uiConfig?.showAbLoopControls]);

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;
      const skipButtons = player.skipButtons();
      skipButtons.setForwardHandler(onNext);
      skipButtons.setBackwardHandler(onPrevious);
    }, [getPlayer, onNext, onPrevious]);

    useEffect(() => {
      if (scene.interactive && interactiveInitialised) {
        interactiveReady.current = false;
        uploadScript(scene.paths.funscript || "").then(() => {
          interactiveReady.current = true;
        });
      }
    }, [
      uploadScript,
      interactiveInitialised,
      scene.interactive,
      scene.paths.funscript,
    ]);

    // play the script if video started before script upload finished
    useEffect(() => {
      if (interactiveState !== ConnectionState.Ready) return;
      const player = getPlayer();
      if (!player || player.paused()) return;
      interactiveClient.ensurePlaying(player.currentTime());
    }, [interactiveState, getPlayer, interactiveClient]);

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      const vrMenu = player.vrMenu();

      let showButton = false;

      if (vrTag) {
        showButton = scene.tags.some((tag) => vrTag === tag.name);
      }

      vrMenu.setShowButton(showButton);
    }, [getPlayer, scene, vrTag]);




    // Function to update Video.js progress bar trim styles
    const updateVideoJsProgressBarTrimStyles = useCallback((player: VideoJsPlayer) => {
      const progressHolder = player.el().querySelector('.vjs-progress-holder');
      if (!progressHolder) return;

      const duration = player.duration();
      const startTime = scene.start_time ?? 0;
      const endTime = scene.end_time ?? 0;

      if (duration <= 0) return;

      // Remove existing trim overlays
      const existingOverlays = progressHolder.querySelectorAll('.vjs-trim-overlay');
      existingOverlays.forEach((overlay: Element) => overlay.remove());

      // Add start trim overlay
      if (startTime > 0) {
        const startPercent = (startTime / duration) * 100;
        const startOverlay = document.createElement('div');
        startOverlay.className = 'vjs-trim-overlay vjs-trim-start';
        startOverlay.style.cssText = `
          position: absolute;
          top: 0;
          left: 0;
          width: ${startPercent}%;
          height: 100%;
          background-color: rgba(255, 0, 0, 0.3);
          pointer-events: none;
          z-index: 2;
        `;
        progressHolder.appendChild(startOverlay);
      }

      // Add end trim overlay
      if (endTime > 0 && endTime < duration) {
        const endPercent = (endTime / duration) * 100;
        const endOverlay = document.createElement('div');
        endOverlay.className = 'vjs-trim-overlay vjs-trim-end';
        endOverlay.style.cssText = `
          position: absolute;
          top: 0;
          left: ${endPercent}%;
          width: ${100 - endPercent}%;
          height: 100%;
          background-color: rgba(255, 0, 0, 0.3);
          pointer-events: none;
          z-index: 2;
        `;
        progressHolder.appendChild(endOverlay);
      }
    }, [scene.start_time, scene.end_time]);

    // Player event handlers
    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      function canplay(this: VideoJsPlayer) {
        // if we're seeking before starting, don't set the initial timestamp
        // when starting from the beginning, there is a small delay before the event
        // is triggered, so we can't just check if the time is 0
        if (this.currentTime() >= 0.1) {
          return;
        }
      }

      function playing(this: VideoJsPlayer) {
        // This still runs even if autoplay failed on Safari,
        // only set flag if actually playing
        if (!started.current && !this.paused()) {
          started.current = true;
        }
      }

      function loadstart(this: VideoJsPlayer) {
        setReady(true);
      }

      function fullscreenchange(this: VideoJsPlayer) {
        setFullscreen(this.isFullscreen());
      }

      player.on("canplay", canplay);
      player.on("playing", playing);
      player.on("loadstart", loadstart);
      player.on("fullscreenchange", fullscreenchange);

      return () => {
        player.off("canplay", canplay);
        player.off("playing", playing);
        player.off("loadstart", loadstart);
        player.off("fullscreenchange", fullscreenchange);
      };
    }, [getPlayer]);

    // delay before second play event after a play event to adjust for video player issues
    const DELAY_FOR_SECOND_PLAY_MS = 1000;
    const playingTimer = useRef<number>();

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      function playing(this: VideoJsPlayer) {
        if (scene.interactive && interactiveReady.current) {
          interactiveClient.play(this.currentTime());
          // trigger a second script play event to adjust for video player issues
          clearTimeout(playingTimer.current);
          playingTimer.current = setTimeout(() => {
            if (this.paused()) return;
            interactiveClient.play(this.currentTime());
          }, DELAY_FOR_SECOND_PLAY_MS);
        }

        // If trim is enabled, handle trimmed segments
        if (trimEnabled) {
          const currentTime = this.currentTime();
          const startTime = scene.start_time ?? 0;
          const endTime = scene.end_time ?? 0;
          const inTrimmedStart = startTime > 0 && currentTime < startTime;
          const inTrimmedEnd = endTime > 0 && currentTime > endTime;
          const inTrimmedSegment = inTrimmedStart || inTrimmedEnd;

          // If we were paused by trim restrictions OR if we're in a trimmed segment, jump to start_time
          if ((pausedByTrim || inTrimmedSegment) && startTime > 0) {
            this.currentTime(startTime);
            setPausedByTrim(false);
          }
        } else {
          // Reset pausedByTrim when trim is disabled
          if (pausedByTrim) {
            setPausedByTrim(false);
          }
        }
      }

      function pause(this: VideoJsPlayer) {
        interactiveClient.pause();
      }

      function seeking(this: VideoJsPlayer) {
        console.log('SEEKING called', this.currentTime());
        // Track that seeking has started
        isSeeking.current = true;
        lastSeekTime.current = this.currentTime();
      }

      function seeked(this: VideoJsPlayer) {
        // Seeking completed - check if seeking backward
        const currentTime = this.currentTime();
        console.log('SEEKED called', this.currentTime());
        if (showNextSceneOverlay && currentTime < lastSeekTime.current) {
          console.log('SEEKED BACKWARD:', { currentTime, lastSeekTime: lastSeekTime.current });
          setShowNextSceneOverlay(false);
        }
        isSeeking.current = false;
      }


      // Additional seek backward detection for progress bar interactions
      function onProgressInteraction(this: VideoJsPlayer) {
        // This fires when user interacts with progress bar (click, drag, etc.)
        const currentTime = this.currentTime();
        if (showNextSceneOverlay && currentTime < previousTime.current) {
            setShowNextSceneOverlay(false);
        }
        previousTime.current = currentTime;
      }


      function timeupdate(this: VideoJsPlayer) {
        const currentTime = this.currentTime();
        setTime(currentTime);

        // Check for seeking backward - close overlay on ANY backward movement
        if (showNextSceneOverlay && currentTime < previousTime.current) {
          console.log('TIMEUPDATE: Seeking backward detected', {
            currentTime,
            previousTime: previousTime.current,
            overlayState: showNextSceneOverlay,
            paused: this.paused()
          });
          setShowNextSceneOverlay(false);
        }
        previousTime.current = currentTime;

        // Update Video.js progress bar trim styles
        updateVideoJsProgressBarTrimStyles(this);

        // Check if we're in a trimmed segment
        const endTime = scene.end_time ?? 0;
        const startTime = scene.start_time ?? 0;
        const inTrimmedStart = startTime > 0 && currentTime < startTime;
        const inTrimmedEnd = endTime > 0 && currentTime > endTime;
        const inTrimmedSegment = inTrimmedStart || inTrimmedEnd;

        setIsInTrimmedSegment(inTrimmedSegment);

        // Auto-pause when reaching end_time, or show next scene overlay if available
        if (endTime && endTime > 0 && currentTime >= endTime) {
          console.log('ATTEMPTING TO SHOW OVERLAY:', {
            currentTime,
            endTime,
            overlayState: showNextSceneOverlay,
            nextScene: !!nextScene
          });
          if (nextScene) {
            this.pause();
            setShowNextSceneOverlay(true);
          } else if (trimEnabled) {
            this.pause();
            setPausedByTrim(true);
          }
        }
      }

      player.on("playing", playing);
      player.on("pause", pause);
      player.on("timeupdate", timeupdate);
      player.on("seeking", seeking);
      player.on("seeked", seeked);
      player.on("useractive", onProgressInteraction);

      return () => {
        player.off("playing", playing);
        player.off("pause", pause);
        player.off("timeupdate", timeupdate);
        player.off("seeking", seeking);
        player.off("seeked", seeked);
        player.off("useractive", onProgressInteraction);
        clearTimeout(playingTimer.current);
      };
    }, [getPlayer, interactiveClient, scene, trimEnabled, pausedByTrim, updateVideoJsProgressBarTrimStyles]);

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      // don't re-initialise the player unless the scene has changed
      if (!file || scene.id === sceneId.current) return;

      sceneId.current = scene.id;

      setReady(false);

      // reset on new scene
      player.trackActivity().reset();

      // always stop the interactive client on initialisation
      interactiveClient.pause();

      const isSafari = UAParser().browser.name?.includes("Safari");
      const isLandscape = file.height && file.width && file.width > file.height;
      const mobileUiOptions = {
        fullscreen: {
          enterOnRotate: true,
          exitOnRotate: true,
          lockOnRotate: true,
          lockToLandscapeOnEnter: uiConfig?.disableMobileMediaAutoRotateEnabled
            ? false
            : isLandscape,
        },
        touchControls: {
          disabled: true,
        },
      };
      if (!isSafari) {
        player.mobileUi(mobileUiOptions);
      }

      function isDirect(src: URL) {
        return (
          src.pathname.endsWith("/stream") ||
          src.pathname.endsWith("/stream.mpd") ||
          src.pathname.endsWith("/stream.m3u8")
        );
      }

      const { duration } = file;
      const sourceSelector = player.sourceSelector();
      sourceSelector.setSources(
        scene.sceneStreams
          .filter((stream) => {
            const src = new URL(stream.url);
            const isFileTranscode = !isDirect(src);

            return !(isFileTranscode && isSafari);
          })
          .map((stream) => {
            const src = new URL(stream.url);

            return {
              src: stream.url,
              type: stream.mime_type ?? undefined,
              label: stream.label ?? undefined,
              offset: !isDirect(src),
              duration,
            };
          })
      );

      function getDefaultLanguageCode() {
        let languageCode = window.navigator.language;

        if (languageCode.indexOf("-") !== -1) {
          languageCode = languageCode.split("-")[0];
        }

        if (languageCode.indexOf("_") !== -1) {
          languageCode = languageCode.split("_")[0];
        }

        return languageCode;
      }

      if (scene.captions && scene.captions.length > 0) {
        const languageCode = getDefaultLanguageCode();
        let hasDefault = false;

        for (let caption of scene.captions) {
          const lang = caption.language_code;
          let label = lang;
          if (languageMap.has(lang)) {
            label = languageMap.get(lang)!;
          }

          label = label + " (" + caption.caption_type + ")";
          const setAsDefault = !hasDefault && languageCode == lang;
          if (setAsDefault) {
            hasDefault = true;
          }
          sourceSelector.addTextTrack(
            {
              src: `${scene.paths.caption}?lang=${lang}&type=${caption.caption_type}`,
              kind: "captions",
              srclang: lang,
              label: label,
              default: setAsDefault,
            },
            false
          );
        }
      }

      auto.current =
        autoplay ||
        (interfaceConfig?.autostartVideo ?? false) ||
        _initialTimestamp > 0;

      const alwaysStartFromBeginning =
        uiConfig?.alwaysStartFromBeginning ?? false;
      const resumeTime = scene.resume_time ?? 0;
      const startTime = scene.start_time ?? 0;

      let startPosition = _initialTimestamp;
      if (
        !startPosition &&
        !alwaysStartFromBeginning &&
        resumeTime > 0
      ) {
        // Check if resume position is in a trimmed segment
        const endTime = scene.end_time ?? 0;
        const inTrimmedStart = startTime > 0 && resumeTime < startTime;
        const inTrimmedEnd = endTime > 0 && resumeTime > endTime;
        const inTrimmedSegment = inTrimmedStart || inTrimmedEnd;

        if (inTrimmedSegment) {
          // If resume position is in trimmed segment, start from start_time
          startPosition = startTime > 0 ? startTime : 0;
        } else {
          startPosition = resumeTime;
        }
      } else if (
        !startPosition &&
        !alwaysStartFromBeginning &&
        !resumeTime &&
        startTime > 0
      ) {
        // If no resume time but start_time is set, use start_time
        startPosition = startTime;
      }

      setTime(startPosition);

      player.load();
      player.focus();

      player.ready(() => {
        player.vttThumbnails().src(scene.paths.vtt ?? null);

        if (startPosition) {
          player.currentTime(startPosition);
        }

        // Initialize Video.js progress bar trim styles
        updateVideoJsProgressBarTrimStyles(player);

        // Also set trim styles after a short delay to ensure progress bar is rendered
        setTimeout(() => {
          updateVideoJsProgressBarTrimStyles(player);
        }, 100);

        // Set trim styles when metadata is loaded
        player.on('loadedmetadata', () => {
          updateVideoJsProgressBarTrimStyles(player);
        });

        // Set trim styles when video can play
        player.on('canplay', () => {
          updateVideoJsProgressBarTrimStyles(player);
        });
      });

      started.current = false;

      // Update Video.js progress bar trim styles when scene changes
      setTimeout(() => {
        updateVideoJsProgressBarTrimStyles(player);
      }, 200);
    }, [
      getPlayer,
      file,
      scene,
      interactiveClient,
      autoplay,
      interfaceConfig?.autostartVideo,
      uiConfig?.alwaysStartFromBeginning,
      uiConfig?.disableMobileMediaAutoRotateEnabled,
      _initialTimestamp,
      updateVideoJsProgressBarTrimStyles,
    ]);

    useEffect(() => {
      return () => {
        // stop the interactive client on unmount
        interactiveClient.pause();
      };
    }, [interactiveClient]);

    // Update Video.js progress bar trim styles when scene changes
    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      // Update immediately
      updateVideoJsProgressBarTrimStyles(player);

      // Also update after a delay to ensure progress bar is ready
      const timeoutId = setTimeout(() => {
        updateVideoJsProgressBarTrimStyles(player);
      }, 300);

      return () => clearTimeout(timeoutId);
    }, [getPlayer, scene, updateVideoJsProgressBarTrimStyles]);




    const loadMarkers = useCallback(() => {
      const player = getPlayer();
      if (!player) return;

      const markerData = scene.scene_markers.map((marker) => ({
        title: getMarkerTitle(marker),
        seconds: marker.seconds,
        end_seconds: marker.end_seconds ?? null,
        primaryTag: marker.primary_tag,
      }));

      const markers = player!.markers();

      const uniqueTagNames = markerData
        .map((marker) => marker.primaryTag.name)
        .filter((value, index, self) => self.indexOf(value) === index);

      // Wait for colors
      markers.findColors(uniqueTagNames);

      const showRangeTags =
        !ScreenUtils.isMobile() && (uiConfig?.showRangeMarkers ?? true);
      const timestampMarkers: IMarker[] = [];
      const rangeMarkers: IMarker[] = [];

      if (!showRangeTags) {
        for (const marker of markerData) {
          timestampMarkers.push(marker);
        }
      } else {
        for (const marker of markerData) {
          if (marker.end_seconds === null) {
            timestampMarkers.push(marker);
          } else {
            rangeMarkers.push(marker);
          }
        }
      }

      requestAnimationFrame(() => {
        markers.addDotMarkers(timestampMarkers);
        markers.addRangeMarkers(rangeMarkers);
      });
    }, [getPlayer, scene, uiConfig]);

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      if (scene.paths.screenshot) {
        player.poster(scene.paths.screenshot);
      } else {
        player.poster("");
      }

      // Define the event handler outside the useEffect
      const handleLoadMetadata = () => {
        loadMarkers();
      };

      // Ensure markers are added after player is fully ready and sources are loaded
      if (player.readyState() >= 1) {
        loadMarkers();
      } else {
        player.on("loadedmetadata", handleLoadMetadata);
      }

      return () => {
        player.off("loadedmetadata", handleLoadMetadata);
        const markers = player!.markers();
        markers.clearMarkers();
      };
    }, [getPlayer, scene, loadMarkers]);

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      async function saveActivity(resumeTime: number, playDuration: number) {
        if (!scene.id) return;

        await sceneSaveActivity({
          variables: {
            id: scene.id,
            playDuration,
            resume_time: resumeTime,
          },
        });
      }

      async function incrementPlayCount() {
        if (!scene.id) return;

        await sceneIncrementPlayCount({
          variables: {
            id: scene.id,
          },
        });
      }

      const activity = player.trackActivity();
      activity.saveActivity = saveActivity;
      activity.incrementPlayCount = incrementPlayCount;
      activity.minimumPlayPercent = minimumPlayPercent;
      activity.setEnabled(trackActivity);
    }, [
      getPlayer,
      scene,
      vrTag,
      trackActivity,
      minimumPlayPercent,
      sceneIncrementPlayCount,
      sceneSaveActivity,
    ]);

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      player.loop(looping);
      interactiveClient.setLooping(looping);
    }, [getPlayer, interactiveClient, looping]);

    useEffect(() => {
      const player = getPlayer();
      if (!player || !ready || !auto.current) {
        return;
      }

      // check if we're waiting for the interactive client
      if (
        scene.interactive &&
        interactiveClient.handyKey &&
        currentScript !== scene.paths.funscript
      ) {
        return;
      }

      player.play();
      auto.current = false;
    }, [getPlayer, scene, ready, interactiveClient, currentScript]);

    // Handle video end - show next scene overlay if available
    const handleVideoEnd = useCallback(() => {
      if (nextScene) {
        setShowNextSceneOverlay(true);
      } else {
        onComplete();
      }
    }, [nextScene, onComplete]);

    // Handle next scene overlay actions
    const handleNextScenePlay = useCallback(() => {
      setShowNextSceneOverlay(false);
      if (onPlayScene && nextScene) {
        onPlayScene(nextScene.id);
      } else {
        onNext();
      }
    }, [onPlayScene, nextScene, onNext]);

    const handleNextSceneSkip = useCallback(() => {
      setShowNextSceneOverlay(false);
      if (onPlayScene && nextScene) {
        onPlayScene(nextScene.id);
      } else {
        onNext();
      }
    }, [onPlayScene, nextScene, onNext]);

    const handleNextSceneCancel = useCallback(() => {
      setShowNextSceneOverlay(false);
      onComplete();
    }, [onComplete]);

    // Handle clicks on video player container for progress bar interactions
    const handleVideoPlayerClick = useCallback((event: React.MouseEvent<HTMLDivElement>) => {
      const target = event.target as HTMLElement;
      if (target.closest('.vjs-progress-control') || target.closest('.vjs-progress-holder')) {
        if (showNextSceneOverlay) {
            setShowNextSceneOverlay(false);
        }
      }
    }, [showNextSceneOverlay]);

    // Attach handler for onComplete event
    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      player.on("ended", handleVideoEnd);

      return () => player.off("ended");
    }, [getPlayer, handleVideoEnd]);

    function onScrubberScroll() {
      if (started.current) {
        getPlayer()?.pause();
      }
    }

    function onScrubberSeek(seconds: number) {
      // Reset pausedByTrim flag when manually seeking
      setPausedByTrim(false);

      // Close next scene overlay if seeking backward from end of video
      if (showNextSceneOverlay && seconds < time) {
            setShowNextSceneOverlay(false);
      }

      if (started.current) {
        getPlayer()?.currentTime(seconds);
      } else {
        setTime(seconds);
      }
    }



    // Override spacebar to always pause/play
    function onKeyDown(this: HTMLDivElement, event: KeyboardEvent) {
      const player = getPlayer();
      if (!player) return;

      if (event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) {
        return;
      }

      if (event.key == " ") {
        event.preventDefault();
        event.stopPropagation();
        if (player.paused()) {
          player.play();
        } else {
          player.pause();
        }
      }
    }

    const isPortrait =
      file && file.height && file.width && file.height > file.width;

    return (
      <div
        className={cx("VideoPlayer", {
          portrait: isPortrait,
          "no-file": !file,
        })}
        onKeyDownCapture={onKeyDown}
        onClick={handleVideoPlayerClick}
      >
        <div className="video-wrapper" ref={videoRef}>
          {showNextSceneOverlay && nextScene && (
            <NextSceneOverlay
              nextScene={nextScene}
              onPlay={handleNextScenePlay}
              onCancel={handleNextSceneCancel}
              onSkip={handleNextSceneSkip}
            />
          )}
        </div>
        {scene.interactive &&
          (interactiveState !== ConnectionState.Ready ||
            getPlayer()?.paused()) && <SceneInteractiveStatus />}
        {file && showScrubber && (
          <ScenePlayerScrubber
            file={file}
            scene={scene}
            time={time}
            onSeek={onScrubberSeek}
            onScroll={onScrubberScroll}
          />
        )}
        {isInTrimmedSegment && (
          <div className="trim-status-overlay">
            <span className="trim-status-text">Trimmed segment reached</span>
          </div>
        )}
      </div>
    );
  }
);

export default ScenePlayer;
