import React, {
  useEffect,
  useRef,
  useState,
  useCallback,
  useMemo,
} from "react";
import {
  defineMessages,
  FormattedMessage,
  MessageDescriptor,
  useIntl,
} from "react-intl";
import { Nav, Navbar, Button, Fade } from "react-bootstrap";
import { IconDefinition } from "@fortawesome/fontawesome-svg-core";
import { Link, NavLink, useLocation, useHistory } from "react-router-dom";
import Mousetrap from "mousetrap";

import SessionUtils from "src/utils/session";
import { Icon } from "src/components/Shared/Icon";
import { ConfigurationContext } from "src/hooks/Config";
import { ManualStateContext } from "./Help/context";
import { SettingsButton } from "./SettingsButton";
import { NotesModal } from "./Notes/NotesModal";
import { useNotes } from "src/hooks/useNotes";
import {
  faBars,
  faChartColumn,
  faFilm,
  faHistory,
  faClock,
  faImage,
  faImages,
  faMapMarkerAlt,
  faPlayCircle,
  faQuestionCircle,
  faRandom,
  faSignOutAlt,
  faStar,
  faStarHalfStroke,
  faTag,
  faTimes,
  faUser,
  faVideo,
  faStickyNote,
  faGamepad,
} from "@fortawesome/free-solid-svg-icons";
import { baseURL } from "src/core/createClient";
import { PatchComponent } from "src/patch";
import {
  getRandomBestScene,
  getRandomScene,
  getRandomUnclassifiedScene,
  getRandomUnorganisedScene,
  makeReviewSceneLink,
} from "src/utils/randomSceneSelection";
import { useHotkey } from "src/hooks/hotkeys";
import { getHotkey, HotkeyAction } from "src/core/config";

interface IMenuItem {
  name: string;
  message: MessageDescriptor;
  href: string;
  icon: IconDefinition;
  hotkeyAction: HotkeyAction;
  userCreatable?: boolean;
}
const messages = defineMessages({
  scenes: {
    id: "scenes",
    defaultMessage: "Scenes",
  },
  games: {
    id: "games",
    defaultMessage: "Games",
  },
  images: {
    id: "images",
    defaultMessage: "Images",
  },
  groups: {
    id: "groups",
    defaultMessage: "Groups",
  },
  markers: {
    id: "markers",
    defaultMessage: "Markers",
  },
  performers: {
    id: "performers",
    defaultMessage: "Performers",
  },
  studios: {
    id: "studios",
    defaultMessage: "Studios",
  },
  tags: {
    id: "tags",
    defaultMessage: "Tags",
  },
  primary_tag: {
    id: "primary_tag",
    defaultMessage: "Primary Tag",
  },
  galleries: {
    id: "galleries",
    defaultMessage: "Galleries",
  },
  sceneTagger: {
    id: "sceneTagger",
    defaultMessage: "Scene Tagger",
  },
  statistics: {
    id: "statistics",
    defaultMessage: "Statistics",
  },
  viewHistory: {
    id: "view_history",
    defaultMessage: "History",
  },
  watchLater: {
    id: "watch_later",
    defaultMessage: "Watch Later",
  },
});

const allMenuItems: IMenuItem[] = [
  {
    name: "scenes",
    message: messages.scenes,
    href: "/scenes",
    icon: faPlayCircle,
    hotkeyAction: "nav.goto.scenes",
  },
  {
    name: "games",
    message: messages.games,
    href: "/games",
    icon: faGamepad,
    hotkeyAction: "nav.goto.games",
    userCreatable: true,
  },
  {
    name: "images",
    message: messages.images,
    href: "/images",
    icon: faImage,
    hotkeyAction: "nav.goto.images",
  },
  {
    name: "groups",
    message: messages.groups,
    href: "/groups",
    icon: faFilm,
    hotkeyAction: "nav.goto.groups",
    userCreatable: true,
  },
  {
    name: "markers",
    message: messages.markers,
    href: "/scenes/markers",
    icon: faMapMarkerAlt,
    hotkeyAction: "nav.goto.markers",
  },
  {
    name: "galleries",
    message: messages.galleries,
    href: "/galleries",
    icon: faImages,
    hotkeyAction: "nav.goto.galleries",
    userCreatable: true,
  },
  {
    name: "performers",
    message: messages.performers,
    href: "/performers",
    icon: faUser,
    hotkeyAction: "nav.goto.performers",
    userCreatable: true,
  },
  {
    name: "studios",
    message: messages.studios,
    href: "/studios",
    icon: faVideo,
    hotkeyAction: "nav.goto.studios",
    userCreatable: true,
  },
  {
    name: "tags",
    message: messages.tags,
    href: "/tags",
    icon: faTag,
    hotkeyAction: "nav.goto.tags",
    userCreatable: true,
  },
  {
    name: "history",
    message: messages.viewHistory,
    href: "/history",
    icon: faHistory,
    hotkeyAction: "nav.goto.history",
  },
  {
    name: "watch-later",
    message: messages.watchLater,
    href: "/watch-later",
    icon: faClock,
    hotkeyAction: "nav.goto.watchLater",
  },
];

const newPathsList = allMenuItems
  .filter((item) => item.userCreatable)
  .map((item) => item.href);

const MainNavbarMenuItems = PatchComponent(
  "MainNavBar.MenuItems",
  (props: React.PropsWithChildren<{}>) => {
    return <Nav>{props.children}</Nav>;
  }
);

const MainNavbarUtilityItems = PatchComponent(
  "MainNavBar.UtilityItems",
  (props: React.PropsWithChildren<{}>) => {
    return <>{props.children}</>;
  }
);

export const MainNavbar: React.FC = () => {
  const history = useHistory();
  const location = useLocation();
  const { configuration, loading } = React.useContext(ConfigurationContext);
  const { openManual } = React.useContext(ManualStateContext);

  const [expanded, setExpanded] = useState(false);
  const [showNotesModal, setShowNotesModal] = useState(false);
  const { notes, saveNotes } = useNotes();

  // Show all menu items by default, unless config says otherwise
  const menuItems = useMemo(() => {
    let cfgMenuItems = configuration?.interface.menuItems;
    if (!cfgMenuItems) {
      return allMenuItems;
    }

    // translate old movies menu item to groups
    cfgMenuItems = cfgMenuItems.map((item) => {
      if (item === "movies") {
        return "groups";
      }
      return item;
    });

    return allMenuItems.filter((menuItem) =>
      cfgMenuItems!.includes(menuItem.name)
    );
  }, [configuration]);

  // react-bootstrap typing bug
  const navbarRef = useRef<HTMLElement | null>(null);
  const intl = useIntl();

  const maybeCollapse = useCallback(
    (event: Event) => {
      if (
        navbarRef.current &&
        event.target instanceof Node &&
        !navbarRef.current.contains(event.target)
      ) {
        setExpanded(false);
      }
    },
    [setExpanded]
  );

  useEffect(() => {
    if (expanded) {
      document.addEventListener("click", maybeCollapse);
      document.addEventListener("touchstart", maybeCollapse);
    }
    return () => {
      document.removeEventListener("click", maybeCollapse);
      document.removeEventListener("touchstart", maybeCollapse);
    };
  }, [expanded, maybeCollapse]);

  const goto = useCallback(
    (page: string) => {
      history.push(page);
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    },
    [history]
  );

  const handleReviewClick = useCallback(async () => {
    const sceneId = await getRandomUnorganisedScene();
    if (sceneId) {
      history.push(makeReviewSceneLink(sceneId));
    } else {
      alert(
        intl.formatMessage({
          id: "no_scenes_to_review",
          defaultMessage: "No unorganised scenes to review",
        })
      );
    }
  }, [history, intl]);

  const handleFirstReviewClick = useCallback(async () => {
    const sceneId = await getRandomUnclassifiedScene();
    if (sceneId) {
      history.push(makeReviewSceneLink(sceneId));
    } else {
      alert(
        intl.formatMessage({
          id: "no_scenes_to_first_review",
        })
      );
    }
  }, [history, intl]);

  const handleRandomClick = useCallback(async () => {
    const ratingThreshold =
      configuration?.interface?.randomRatingThreshold ?? 55;
    const sceneId = await getRandomScene(ratingThreshold);
    if (sceneId) {
      history.push(`/scenes/${sceneId}`);
    } else {
      alert(
        intl.formatMessage({
          id: "no_scenes_available",
          defaultMessage: `No scenes with rating >= ${ratingThreshold}`,
        })
      );
    }
  }, [history, intl, configuration]);

  const handleRandomBestClick = useCallback(async () => {
    const ratingThreshold =
      configuration?.interface?.randomBestRatingThreshold ?? 90;
    const sceneId = await getRandomBestScene(ratingThreshold);
    if (sceneId) {
      history.push(`/scenes/${sceneId}`);
    } else {
      alert(
        intl.formatMessage({
          id: "no_best_scenes_available",
          defaultMessage: `No scenes with rating >= ${ratingThreshold}`,
        })
      );
    }
  }, [history, intl, configuration]);

  const firstReviewHotkey = useHotkey("scene.firstReview");
  const reviewHotkey = useHotkey("scene.review");
  const randomHotkey = useHotkey("scene.random");
  const randomBestHotkey = useHotkey("scene.randomBest");
  const openManualHotkey = useHotkey("nav.openManual");
  const gotoSettingsHotkey = useHotkey("nav.gotoSettings");
  const newItemHotkey = useHotkey("nav.newItem");

  useEffect(() => {
    Mousetrap.bind(firstReviewHotkey, () => {
      handleFirstReviewClick();
    });
    Mousetrap.bind(reviewHotkey, () => {
      handleReviewClick();
    });
    Mousetrap.bind(randomHotkey, () => {
      handleRandomClick();
    });
    Mousetrap.bind(randomBestHotkey, () => {
      handleRandomBestClick();
    });

    return () => {
      Mousetrap.unbind(firstReviewHotkey);
      Mousetrap.unbind(reviewHotkey);
      Mousetrap.unbind(randomHotkey);
      Mousetrap.unbind(randomBestHotkey);
    };
  }, [
    firstReviewHotkey,
    reviewHotkey,
    randomHotkey,
    randomBestHotkey,
    handleFirstReviewClick,
    handleReviewClick,
    handleRandomClick,
    handleRandomBestClick,
  ]);

  const pathname = location.pathname.replace(/\/$/, "");
  let newPath = newPathsList.includes(pathname) ? `${pathname}/new` : null;
  if (newPath !== null) {
    let queryParam = new URLSearchParams(location.search).get("q");
    if (queryParam) {
      newPath += "?q=" + encodeURIComponent(queryParam);
    }
  }

  // set up hotkeys
  useEffect(() => {
    const menuItemHotkeys = menuItems.map((item) =>
      getHotkey(configuration?.ui.hotkeys, item.hotkeyAction)
    );

    Mousetrap.bind(openManualHotkey, () => openManual());
    Mousetrap.bind(gotoSettingsHotkey, () => goto("/settings"));

    menuItems.forEach((item, i) =>
      Mousetrap.bind(menuItemHotkeys[i], () => goto(item.href))
    );

    if (newPath) {
      Mousetrap.bind(newItemHotkey, () => history.push(String(newPath)));
    }

    return () => {
      Mousetrap.unbind(openManualHotkey);
      Mousetrap.unbind(gotoSettingsHotkey);
      menuItemHotkeys.forEach((key) => Mousetrap.unbind(key));

      if (newPath) {
        Mousetrap.unbind(newItemHotkey);
      }
    };
  });

  function maybeRenderLogout() {
    if (SessionUtils.isLoggedIn()) {
      return (
        <Button
          className="minimal logout-button d-flex align-items-center"
          href={`${baseURL}logout`}
          title={intl.formatMessage({ id: "actions.logout" })}
        >
          <Icon icon={faSignOutAlt} />
        </Button>
      );
    }
  }

  const handleDismiss = useCallback(() => setExpanded(false), [setExpanded]);

  function renderUtilityButtons() {
    return (
      <>
        <Button
          className="nav-utility minimal"
          onClick={() => setShowNotesModal(true)}
          title={intl.formatMessage({
            id: "notes.title",
            defaultMessage: "Notes",
          })}
        >
          <Icon icon={faStickyNote} />
        </Button>
        <NavLink
          className="nav-utility"
          exact
          to="/stats"
          onClick={handleDismiss}
        >
          <Button
            className="minimal d-flex align-items-center h-100"
            title={intl.formatMessage({ id: "statistics" })}
          >
            <Icon icon={faChartColumn} />
          </Button>
        </NavLink>
        <a
          className="nav-utility"
          href="/settings"
          target="_blank"
          rel="noopener noreferrer"
          onClick={handleDismiss}
        >
          <SettingsButton />
        </a>
        <Button
          className="nav-utility minimal"
          onClick={() => openManual()}
          title={intl.formatMessage({ id: "help" })}
        >
          <Icon icon={faQuestionCircle} />
        </Button>
        {maybeRenderLogout()}
      </>
    );
  }

  return (
    <>
      <Navbar
        collapseOnSelect
        fixed="top"
        variant="dark"
        bg="dark"
        className="top-nav"
        expand="xl"
        expanded={expanded}
        onToggle={setExpanded}
        ref={navbarRef}
      >
        <Navbar.Collapse className="bg-dark order-sm-1">
          <Fade in={!loading}>
            <>
              <MainNavbarMenuItems>
                {menuItems.map(({ href, icon, message }) => {
                  const isActive =
                    location.pathname === href ||
                    (href !== "/" && location.pathname.startsWith(href));
                  return (
                    <Nav.Link
                      eventKey={href}
                      as="div"
                      key={href}
                      className="col-4 col-sm-3 col-md-2 col-lg-auto"
                    >
                      <NavLink
                        to={href}
                        className="d-block"
                        style={{ textDecoration: "none" }}
                      >
                        <Button
                          className={`minimal p-4 p-xl-2 d-flex d-xl-inline-block flex-column justify-content-between align-items-center ${
                            isActive ? "active" : ""
                          }`}
                        >
                          <Icon
                            {...{ icon }}
                            className="nav-menu-icon d-block d-xl-inline mb-2 mb-xl-0"
                          />
                          <span>{intl.formatMessage(message)}</span>
                        </Button>
                      </NavLink>
                    </Nav.Link>
                  );
                })}
              </MainNavbarMenuItems>
              <Nav>
                <MainNavbarUtilityItems>
                  {renderUtilityButtons()}
                </MainNavbarUtilityItems>
              </Nav>
            </>
          </Fade>
        </Navbar.Collapse>

        <Navbar.Brand as="div" onClick={handleDismiss}>
          <Link to="/">
            <Button className="minimal brand-link d-inline-block">
              <div className="text-left" style={{ marginTop: "-6px" }}>
                Stash:{" "}
                <strong>
                  <i>Reforged</i>
                </strong>
                <small
                  style={{
                    fontSize: "0.75em",
                    lineHeight: "1",
                    display: "block",
                  }}
                >
                  by niemandssh
                </small>
              </div>
            </Button>
          </Link>
        </Navbar.Brand>

        <Nav className="navbar-buttons flex-row ml-auto order-xl-2">
          {!!newPath && (
            <div className="mr-2">
              <Link to={newPath}>
                <Button variant="primary">
                  <FormattedMessage id="new" defaultMessage="New" />
                </Button>
              </Link>
            </div>
          )}
          <MainNavbarUtilityItems>
            {renderUtilityButtons()}
          </MainNavbarUtilityItems>
          <button
            type="button"
            className="btn btn-primary review-btn ml-2"
            style={{ display: "inline-block", visibility: "visible" }}
            onClick={handleFirstReviewClick}
            title={intl.formatMessage(
              { id: "first_review_unclassified_video_title" },
              { hotkey: firstReviewHotkey }
            )}
          >
            <Icon icon={faStarHalfStroke} className="mr-1" />
            <FormattedMessage id="first_review" />
          </button>
          <button
            type="button"
            className="btn btn-primary review-btn ml-2"
            style={{ display: "inline-block", visibility: "visible" }}
            onClick={handleReviewClick}
            title={intl.formatMessage(
              {
                id: "review_unrated_video_title",
                defaultMessage:
                  "Review random unorganised video ({hotkey})",
              },
              { hotkey: reviewHotkey }
            )}
          >
            <Icon icon={faStarHalfStroke} className="mr-1" />
            Review
          </button>
          <button
            type="button"
            className="btn btn-primary random-btn ml-2"
            style={{ display: "inline-block", visibility: "visible" }}
            onClick={handleRandomClick}
            title={intl.formatMessage(
              {
                id: "random_scene_title",
                defaultMessage:
                  "Open random scene with rating >= {ratingThreshold} ({hotkey})",
              },
              {
                ratingThreshold:
                  configuration?.interface?.randomRatingThreshold ?? 55,
                hotkey: randomHotkey,
              }
            )}
          >
            <Icon icon={faRandom} className="mr-1" />
            Random
          </button>
          <button
            type="button"
            className="btn btn-primary random-best-btn ml-2"
            style={{ display: "inline-block", visibility: "visible" }}
            onClick={handleRandomBestClick}
            title={intl.formatMessage(
              {
                id: "random_best_scene_title",
                defaultMessage:
                  "Open random scene with rating >= {ratingThreshold} ({hotkey})",
              },
              {
                ratingThreshold:
                  configuration?.interface?.randomBestRatingThreshold ?? 90,
                hotkey: randomBestHotkey,
              }
            )}
          >
            <Icon icon={faStar} className="mr-1" />
            Random best
          </button>
          <Navbar.Toggle className="nav-menu-toggle ml-sm-2">
            <Icon icon={expanded ? faTimes : faBars} />
          </Navbar.Toggle>
        </Nav>
      </Navbar>

      <NotesModal
        show={showNotesModal}
        onHide={() => setShowNotesModal(false)}
        notes={notes}
        onSave={saveNotes}
      />
    </>
  );
};
