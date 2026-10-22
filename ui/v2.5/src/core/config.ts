import { IntlShape } from "react-intl";
import { ITypename } from "src/utils/data";
import { ImageWallOptions } from "src/utils/imageWall";
import { RatingSystemOptions } from "src/utils/rating";
import {
  FilterMode,
  SavedFilterDataFragment,
  SortDirectionEnum,
} from "./generated-graphql";
import { View } from "src/components/List/views";
import { ITaggerConfig } from "src/components/Tagger/constants";
import { DisplayMode } from "src/models/list-filter/types";

// NOTE: double capitals aren't converted correctly in the backend

export interface ISavedFilterRow extends ITypename {
  __typename: "SavedFilter";
  savedFilterId: number;
}

export interface IMessage {
  id: string;
  values: { [key: string]: string };
}

export interface ICustomFilter extends ITypename {
  __typename: "CustomFilter";
  message?: IMessage;
  title?: string;
  mode: FilterMode;
  sortBy: string;
  direction: SortDirectionEnum;
}

export type DefaultFilters = {
  [P in View]?: SavedFilterDataFragment;
};

export type FrontPageContent = ISavedFilterRow | ICustomFilter;

export const defaultMaxOptionsShown = 200;

// Identifiers for user-configurable keyboard shortcuts. Each maps to a
// Mousetrap-compatible key combo string (e.g. "alt+r", "del", "ctrl+shift+d").
// Actions that mean the same thing on multiple pages (e.g. "save the current
// edit form", "delete the current entity") share a single id rather than
// being duplicated per-page, so re-binding one hotkey updates every page it
// applies to.
export type HotkeyAction =
  // global navigation
  | "nav.openManual"
  | "nav.gotoSettings"
  | "nav.newItem"
  | "nav.goto.scenes"
  | "nav.goto.games"
  | "nav.goto.images"
  | "nav.goto.groups"
  | "nav.goto.markers"
  | "nav.goto.galleries"
  | "nav.goto.performers"
  | "nav.goto.studios"
  | "nav.goto.tags"
  | "nav.goto.history"
  | "nav.goto.watchLater"
  | "scene.firstReview"
  | "scene.review"
  | "scene.random"
  | "scene.randomBest"
  // rating
  | "rating.trigger"
  // list pages (scenes/galleries/images/performers/studios/tags/groups/markers)
  | "list.playRandom"
  | "list.focusSearch"
  | "list.closeSidebar"
  | "list.selectAll"
  | "list.selectNone"
  | "list.bulkEdit"
  | "list.bulkDelete"
  | "list.openFilterEditor"
  | "list.reshuffleRandom"
  | "list.pageNext"
  | "list.pagePrevious"
  | "list.pageForward10"
  | "list.pageBack10"
  | "list.pageFirst"
  | "list.pageLast"
  | "list.viewGrid"
  | "list.viewList"
  | "list.viewWall"
  | "list.viewSlideshow"
  | "list.viewWeb"
  | "list.zoomIn"
  | "list.zoomOut"
  // image slideshow
  | "slideshow.previous"
  | "slideshow.next"
  | "slideshow.playPause"
  // shared entity detail page actions
  | "detail.tabDetails"
  | "detail.tabEdit"
  | "detail.tabHistory"
  | "detail.tabFileInfo"
  | "detail.toggleEdit"
  | "detail.toggleFavorite"
  | "detail.toggleSidebar"
  | "detail.incrementO"
  | "detail.newSubItem"
  | "edit.save"
  | "entity.delete"
  | "scene.delete"
  // scene detail page specifics
  | "scene.toggleQueuePanel"
  | "scene.toggleMarkersPanel"
  | "scene.toggleFileInfoPanel"
  | "scene.queueNext"
  | "scene.queuePrevious"
  | "scene.queueRandom"
  | "scene.hideScrubber"
  // gallery detail page specifics
  | "gallery.tabChapters"
  // performer detail page specifics
  | "performer.tabScenes"
  | "performer.tabGalleries"
  | "performer.tabGroups";

export const defaultHotkeys: Record<HotkeyAction, string> = {
  "nav.openManual": "?",
  "nav.gotoSettings": "g z",
  "nav.newItem": "n",
  "nav.goto.scenes": "g s",
  "nav.goto.games": "g a",
  "nav.goto.images": "g i",
  "nav.goto.groups": "g v",
  "nav.goto.markers": "g k",
  "nav.goto.galleries": "g l",
  "nav.goto.performers": "g p",
  "nav.goto.studios": "g u",
  "nav.goto.tags": "g t",
  "nav.goto.history": "g h",
  "nav.goto.watchLater": "g w",
  "scene.firstReview": "alt+e",
  "scene.review": "alt+r",
  "scene.random": "alt+w",
  "scene.randomBest": "alt+q",

  "rating.trigger": "r",

  "list.playRandom": "p r",
  "list.focusSearch": "/",
  "list.closeSidebar": "esc",
  "list.selectAll": "s a",
  "list.selectNone": "s n",
  "list.bulkEdit": "e",
  "list.bulkDelete": "d d",
  "list.openFilterEditor": "f",
  "list.reshuffleRandom": "r",
  "list.pageNext": "right",
  "list.pagePrevious": "left",
  "list.pageForward10": "shift+right",
  "list.pageBack10": "shift+left",
  "list.pageFirst": "ctrl+home",
  "list.pageLast": "ctrl+end",
  "list.viewGrid": "v g",
  "list.viewList": "v l",
  "list.viewWall": "v w",
  "list.viewSlideshow": "v s",
  "list.viewWeb": "v e",
  "list.zoomIn": "+",
  "list.zoomOut": "-",

  "slideshow.previous": "left",
  "slideshow.next": "right",
  "slideshow.playPause": "space",

  "detail.tabDetails": "a",
  "detail.tabEdit": "e",
  "detail.tabHistory": "h",
  "detail.tabFileInfo": "f",
  "detail.toggleEdit": "e",
  "detail.toggleFavorite": "f",
  "detail.toggleSidebar": ",",
  "detail.incrementO": "o",
  "detail.newSubItem": "n",
  "edit.save": "s s",
  "entity.delete": "d d",
  "scene.delete": "del",

  "scene.toggleQueuePanel": "q",
  "scene.toggleMarkersPanel": "k",
  "scene.toggleFileInfoPanel": "i",
  "scene.queueNext": "p n",
  "scene.queuePrevious": "p p",
  "scene.queueRandom": "p r",
  "scene.hideScrubber": ".",

  "gallery.tabChapters": "c",

  "performer.tabScenes": "c",
  "performer.tabGalleries": "g",
  "performer.tabGroups": "m",
};

export function getHotkey(
  hotkeys: Partial<Record<HotkeyAction, string>> | undefined,
  action: HotkeyAction
): string {
  return hotkeys?.[action] ?? defaultHotkeys[action];
}

export interface IUIConfig {
  // unknown to prevent direct access - use getFrontPageContent
  frontPageContent?: unknown;

  showChildTagContent?: boolean;
  showChildStudioContent?: boolean;
  showTagCardOnHover?: boolean;

  abbreviateCounters?: boolean;

  ratingSystemOptions?: RatingSystemOptions;

  // if true a background image will be display on header
  enableMovieBackgroundImage?: boolean;
  // if true a background image will be display on header
  enablePerformerBackgroundImage?: boolean;
  // if true a background image will be display on header
  enableStudioBackgroundImage?: boolean;
  // if true a background image will be display on header
  enableTagBackgroundImage?: boolean;
  // if true view expanded details compact
  compactExpandedDetails?: boolean;
  // if true show all content details by default
  showAllDetails?: boolean;

  // if true the chromecast option will enabled
  enableChromecast?: boolean;

  // if true the fullscreen mobile media auto-rotate option will be disabled
  disableMobileMediaAutoRotateEnabled?: boolean;

  // if true markers with end times will display with a horizontal bar in the scene player
  showRangeMarkers?: boolean;
  // if true continue scene will always play from the beginning
  alwaysStartFromBeginning?: boolean;
  // if true, resume position is only remembered for 5 minutes after last play
  // only applies when alwaysStartFromBeginning is false
  rememberPlayPositionFor5Minutes?: boolean;
  // if true enable activity tracking
  trackActivity?: boolean;
  // the minimum percentage of scene duration which a scene must be played
  // before the play count is incremented
  minimumPlayPercent?: number;

  showAbLoopControls?: boolean;

  // maximum number of items to shown in the dropdown list - defaults to 200
  // upper limit of 1000
  maxOptionsShown?: number;

  imageWallOptions?: ImageWallOptions;

  lastNoteSeen?: number;

  vrTag?: string;

  pinnedFilters?: Record<string, string[]>;
  tableColumns?: Record<string, string[]>;

  advancedMode?: boolean;

  taskDefaults?: Record<string, {}>;

  defaultFilters?: DefaultFilters;

  taggerConfig?: ITaggerConfig;

  // Записи пользователя
  notes?: string;

  // Настройки отображения для конкретных галерей
  galleryDisplayModes?: Record<string, DisplayMode>;

  // Remote scene suggestions (/scenes)
  remoteSceneSuggestionsEnabled?: boolean;
  remoteSceneSuggestionsMinDurationMinutes?: number;
  remoteSceneSuggestionsLimit?: number;
  remoteSceneSuggestionsProviders?: string[];
  remoteSceneSuggestionsPriorityTagIds?: string[];
  remoteSceneSuggestionsStopTagIds?: string[];

  /** Preferred tags for scene cards (ordered). First match paints the preview stripe. */
  scenePreferredTagIds?: string[];

  // user-configurable keyboard shortcuts - see HotkeyAction
  hotkeys?: Partial<Record<HotkeyAction, string>>;
}

export function getFrontPageContent(
  ui: IUIConfig | undefined
): FrontPageContent[] | undefined {
  return ui?.frontPageContent as FrontPageContent[] | undefined;
}

function recentlyReleased(
  intl: IntlShape,
  mode: FilterMode,
  objectsID: string
): ICustomFilter {
  return {
    __typename: "CustomFilter",
    message: {
      id: "recently_released_objects",
      values: { objects: intl.formatMessage({ id: objectsID }) },
    },
    mode,
    sortBy: "date",
    direction: SortDirectionEnum.Desc,
  };
}

function recentlyAdded(
  intl: IntlShape,
  mode: FilterMode,
  objectsID: string
): ICustomFilter {
  return {
    __typename: "CustomFilter",
    message: {
      id: "recently_added_objects",
      values: { objects: intl.formatMessage({ id: objectsID }) },
    },
    mode,
    sortBy: "created_at",
    direction: SortDirectionEnum.Desc,
  };
}

export function generateDefaultFrontPageContent(intl: IntlShape) {
  return [
    recentlyReleased(intl, FilterMode.Scenes, "scenes"),
    recentlyAdded(intl, FilterMode.Studios, "studios"),
    recentlyReleased(intl, FilterMode.Groups, "groups"),
    recentlyAdded(intl, FilterMode.Performers, "performers"),
    recentlyReleased(intl, FilterMode.Galleries, "galleries"),
  ];
}

export function generatePremadeFrontPageContent(intl: IntlShape) {
  return [
    recentlyReleased(intl, FilterMode.Scenes, "scenes"),
    recentlyAdded(intl, FilterMode.Scenes, "scenes"),
    recentlyReleased(intl, FilterMode.Galleries, "galleries"),
    recentlyAdded(intl, FilterMode.Galleries, "galleries"),
    recentlyAdded(intl, FilterMode.Images, "images"),
    recentlyReleased(intl, FilterMode.Groups, "groups"),
    recentlyAdded(intl, FilterMode.Groups, "groups"),
    recentlyAdded(intl, FilterMode.Studios, "studios"),
    recentlyAdded(intl, FilterMode.Performers, "performers"),
  ];
}
