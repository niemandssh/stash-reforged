import { HotkeyAction } from "src/core/config";

export interface IHotkeyDef {
  action: HotkeyAction;
  labelID: string;
}

export interface IHotkeyGroup {
  categoryID: string;
  items: IHotkeyDef[];
}

// Catalogue of every user-configurable hotkey in the app, grouped for
// display in the Settings > Hot Keys tab. Actions shared by several pages
// (e.g. "edit.save", "entity.delete") are listed once - rebinding them here
// updates every page that uses them.
export const hotkeyGroups: IHotkeyGroup[] = [
  {
    categoryID: "config.hotkeys.groups.scenes",
    items: [
      {
        action: "scene.firstReview",
        labelID: "config.hotkeys.scene.first_review",
      },
      { action: "scene.review", labelID: "config.hotkeys.scene.review" },
      { action: "scene.random", labelID: "config.hotkeys.scene.random" },
      {
        action: "scene.randomBest",
        labelID: "config.hotkeys.scene.random_best",
      },
      { action: "scene.delete", labelID: "config.hotkeys.scene.delete" },
      {
        action: "scene.toggleQueuePanel",
        labelID: "config.hotkeys.scene.toggle_queue_panel",
      },
      {
        action: "scene.toggleMarkersPanel",
        labelID: "config.hotkeys.scene.toggle_markers_panel",
      },
      {
        action: "scene.toggleFileInfoPanel",
        labelID: "config.hotkeys.scene.toggle_file_info_panel",
      },
      { action: "scene.queueNext", labelID: "config.hotkeys.scene.queue_next" },
      {
        action: "scene.queuePrevious",
        labelID: "config.hotkeys.scene.queue_previous",
      },
      {
        action: "scene.queueRandom",
        labelID: "config.hotkeys.scene.queue_random",
      },
      {
        action: "scene.hideScrubber",
        labelID: "config.hotkeys.scene.hide_scrubber",
      },
    ],
  },
  {
    categoryID: "config.hotkeys.groups.entity_detail",
    items: [
      {
        action: "detail.tabDetails",
        labelID: "config.hotkeys.detail.tab_details",
      },
      { action: "detail.tabEdit", labelID: "config.hotkeys.detail.tab_edit" },
      {
        action: "detail.tabHistory",
        labelID: "config.hotkeys.detail.tab_history",
      },
      {
        action: "detail.tabFileInfo",
        labelID: "config.hotkeys.detail.tab_file_info",
      },
      {
        action: "detail.toggleEdit",
        labelID: "config.hotkeys.detail.toggle_edit",
      },
      {
        action: "detail.toggleFavorite",
        labelID: "config.hotkeys.detail.toggle_favorite",
      },
      {
        action: "detail.toggleSidebar",
        labelID: "config.hotkeys.detail.toggle_sidebar",
      },
      {
        action: "detail.incrementO",
        labelID: "config.hotkeys.detail.increment_o",
      },
      {
        action: "detail.newSubItem",
        labelID: "config.hotkeys.detail.new_sub_item",
      },
      { action: "edit.save", labelID: "config.hotkeys.edit.save" },
      { action: "entity.delete", labelID: "config.hotkeys.entity.delete" },
      {
        action: "gallery.tabChapters",
        labelID: "config.hotkeys.gallery.tab_chapters",
      },
      {
        action: "performer.tabScenes",
        labelID: "config.hotkeys.performer.tab_scenes",
      },
      {
        action: "performer.tabGalleries",
        labelID: "config.hotkeys.performer.tab_galleries",
      },
      {
        action: "performer.tabGroups",
        labelID: "config.hotkeys.performer.tab_groups",
      },
      { action: "rating.trigger", labelID: "config.hotkeys.rating.trigger" },
    ],
  },
  {
    categoryID: "config.hotkeys.groups.lists",
    items: [
      { action: "list.playRandom", labelID: "config.hotkeys.list.play_random" },
      {
        action: "list.focusSearch",
        labelID: "config.hotkeys.list.focus_search",
      },
      {
        action: "list.closeSidebar",
        labelID: "config.hotkeys.list.close_sidebar",
      },
      { action: "list.selectAll", labelID: "config.hotkeys.list.select_all" },
      { action: "list.selectNone", labelID: "config.hotkeys.list.select_none" },
      { action: "list.bulkEdit", labelID: "config.hotkeys.list.bulk_edit" },
      { action: "list.bulkDelete", labelID: "config.hotkeys.list.bulk_delete" },
      {
        action: "list.openFilterEditor",
        labelID: "config.hotkeys.list.open_filter_editor",
      },
      {
        action: "list.reshuffleRandom",
        labelID: "config.hotkeys.list.reshuffle_random",
      },
      { action: "list.pageNext", labelID: "config.hotkeys.list.page_next" },
      {
        action: "list.pagePrevious",
        labelID: "config.hotkeys.list.page_previous",
      },
      {
        action: "list.pageForward10",
        labelID: "config.hotkeys.list.page_forward_10",
      },
      {
        action: "list.pageBack10",
        labelID: "config.hotkeys.list.page_back_10",
      },
      { action: "list.pageFirst", labelID: "config.hotkeys.list.page_first" },
      { action: "list.pageLast", labelID: "config.hotkeys.list.page_last" },
      { action: "list.viewGrid", labelID: "config.hotkeys.list.view_grid" },
      { action: "list.viewList", labelID: "config.hotkeys.list.view_list" },
      { action: "list.viewWall", labelID: "config.hotkeys.list.view_wall" },
      {
        action: "list.viewSlideshow",
        labelID: "config.hotkeys.list.view_slideshow",
      },
      { action: "list.viewWeb", labelID: "config.hotkeys.list.view_web" },
      { action: "list.zoomIn", labelID: "config.hotkeys.list.zoom_in" },
      { action: "list.zoomOut", labelID: "config.hotkeys.list.zoom_out" },
    ],
  },
  {
    categoryID: "config.hotkeys.groups.slideshow",
    items: [
      {
        action: "slideshow.previous",
        labelID: "config.hotkeys.slideshow.previous",
      },
      { action: "slideshow.next", labelID: "config.hotkeys.slideshow.next" },
      {
        action: "slideshow.playPause",
        labelID: "config.hotkeys.slideshow.play_pause",
      },
    ],
  },
  {
    categoryID: "config.hotkeys.groups.navigation",
    items: [
      { action: "nav.openManual", labelID: "config.hotkeys.nav.open_manual" },
      {
        action: "nav.gotoSettings",
        labelID: "config.hotkeys.nav.goto_settings",
      },
      { action: "nav.newItem", labelID: "config.hotkeys.nav.new_item" },
      { action: "nav.goto.scenes", labelID: "config.hotkeys.nav.goto_scenes" },
      { action: "nav.goto.games", labelID: "config.hotkeys.nav.goto_games" },
      { action: "nav.goto.images", labelID: "config.hotkeys.nav.goto_images" },
      { action: "nav.goto.groups", labelID: "config.hotkeys.nav.goto_groups" },
      {
        action: "nav.goto.markers",
        labelID: "config.hotkeys.nav.goto_markers",
      },
      {
        action: "nav.goto.galleries",
        labelID: "config.hotkeys.nav.goto_galleries",
      },
      {
        action: "nav.goto.performers",
        labelID: "config.hotkeys.nav.goto_performers",
      },
      {
        action: "nav.goto.studios",
        labelID: "config.hotkeys.nav.goto_studios",
      },
      { action: "nav.goto.tags", labelID: "config.hotkeys.nav.goto_tags" },
      {
        action: "nav.goto.history",
        labelID: "config.hotkeys.nav.goto_history",
      },
      {
        action: "nav.goto.watchLater",
        labelID: "config.hotkeys.nav.goto_watch_later",
      },
    ],
  },
];
