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
import { LinkContainer } from "react-router-bootstrap";
import { Link, NavLink, useLocation, useHistory } from "react-router-dom";
import Mousetrap from "mousetrap";
import * as GQL from "src/core/generated-graphql";

import SessionUtils from "src/utils/session";
import { Icon } from "src/components/Shared/Icon";
import { ConfigurationContext } from "src/hooks/Config";
import { ManualStateContext } from "./Help/context";
import { SettingsButton } from "./SettingsButton";
import {
  faBars,
  faChartColumn,
  faFilm,
  faHeart,
  faImage,
  faImages,
  faMapMarkerAlt,
  faPlayCircle,
  faQuestionCircle,
  faSignOutAlt,
  faStarHalfStroke,
  faTag,
  faTimes,
  faUser,
  faVideo,
} from "@fortawesome/free-solid-svg-icons";
import { baseURL } from "src/core/createClient";
import { PatchComponent } from "src/patch";
import { getClient } from "src/core/StashService";

interface IMenuItem {
  name: string;
  message: MessageDescriptor;
  href: string;
  icon: IconDefinition;
  hotkey: string;
  userCreatable?: boolean;
}
const messages = defineMessages({
  scenes: {
    id: "scenes",
    defaultMessage: "Scenes",
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
  galleries: {
    id: "galleries",
    defaultMessage: "Galleries",
  },
  sceneTagger: {
    id: "sceneTagger",
    defaultMessage: "Scene Tagger",
  },
  donate: {
    id: "donate",
    defaultMessage: "Donate",
  },
  statistics: {
    id: "statistics",
    defaultMessage: "Statistics",
  },
});

const allMenuItems: IMenuItem[] = [
  {
    name: "scenes",
    message: messages.scenes,
    href: "/scenes",
    icon: faPlayCircle,
    hotkey: "g s",
  },
  {
    name: "images",
    message: messages.images,
    href: "/images",
    icon: faImage,
    hotkey: "g i",
  },
  {
    name: "groups",
    message: messages.groups,
    href: "/groups",
    icon: faFilm,
    hotkey: "g v",
    userCreatable: true,
  },
  {
    name: "markers",
    message: messages.markers,
    href: "/scenes/markers",
    icon: faMapMarkerAlt,
    hotkey: "g k",
  },
  {
    name: "galleries",
    message: messages.galleries,
    href: "/galleries",
    icon: faImages,
    hotkey: "g l",
    userCreatable: true,
  },
  {
    name: "performers",
    message: messages.performers,
    href: "/performers",
    icon: faUser,
    hotkey: "g p",
    userCreatable: true,
  },
  {
    name: "studios",
    message: messages.studios,
    href: "/studios",
    icon: faVideo,
    hotkey: "g u",
    userCreatable: true,
  },
  {
    name: "tags",
    message: messages.tags,
    href: "/tags",
    icon: faTag,
    hotkey: "g t",
    userCreatable: true,
  },
];

const newPathsList = allMenuItems
  .filter((item) => item.userCreatable)
  .map((item) => item.href);

// Функция для получения случайной сцены без рейтинга и тегов
const getRandomUnratedScene = async (): Promise<string | null> => {
  try {
    const client = getClient();
    const result = await client.query<GQL.FindScenesQuery>({
      query: GQL.FindScenesDocument,
      variables: {
        filter: {
          per_page: 100, // Получаем до 100 сцен для выбора случайной
        },
        scene_filter: {
          rating100: {
            modifier: GQL.CriterionModifier.IsNull,
            value: 0,
          },
          tag_count: {
            modifier: GQL.CriterionModifier.Equals,
            value: 0,
          },
        },
      },
    });

    const scenes = result.data?.findScenes?.scenes || [];
    if (scenes.length === 0) {
      return null;
    }

    // Выбираем случайную сцену
    const randomIndex = Math.floor(Math.random() * scenes.length);
    const randomScene = scenes[randomIndex];
    return randomScene?.id || null;
  } catch (error) {
    console.error("Ошибка при получении случайной сцены:", error);
    return null;
  }
};

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
    const sceneId = await getRandomUnratedScene();
    if (sceneId) {
      history.push(`/scenes/${sceneId}`);
    } else {
      // Показываем уведомление, если нет сцен для рецензирования
      alert(intl.formatMessage({ 
        id: "no_scenes_to_review", 
        defaultMessage: "Нет сцен без рейтинга и тегов для рецензирования" 
      }));
    }
  }, [history, intl]);

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
    Mousetrap.bind("?", () => openManual());
    Mousetrap.bind("g z", () => goto("/settings"));

    menuItems.forEach((item) =>
      Mousetrap.bind(item.hotkey, () => goto(item.href))
    );

    if (newPath) {
      Mousetrap.bind("n", () => history.push(String(newPath)));
    }

    return () => {
      Mousetrap.unbind("?");
      Mousetrap.unbind("g z");
      menuItems.forEach((item) => Mousetrap.unbind(item.hotkey));

      if (newPath) {
        Mousetrap.unbind("n");
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
        <Nav.Link
          className="nav-utility"
          href="https://opencollective.com/stashapp"
          target="_blank"
          onClick={handleDismiss}
        >
          <Button
            className="minimal donate"
            title={intl.formatMessage({ id: "donate" })}
          >
            <Icon icon={faHeart} />
            <span className="d-none d-sm-inline">
              {intl.formatMessage(messages.donate)}
            </span>
          </Button>
        </Nav.Link>
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
        <NavLink
          className="nav-utility"
          exact
          to="/settings"
          onClick={handleDismiss}
        >
          <SettingsButton />
        </NavLink>
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
                {menuItems.map(({ href, icon, message }) => (
                  <Nav.Link
                    eventKey={href}
                    as="div"
                    key={href}
                    className="col-4 col-sm-3 col-md-2 col-lg-auto"
                  >
                    <LinkContainer activeClassName="active" exact to={href}>
                      <Button className="minimal p-4 p-xl-2 d-flex d-xl-inline-block flex-column justify-content-between align-items-center">
                        <Icon
                          {...{ icon }}
                          className="nav-menu-icon d-block d-xl-inline mb-2 mb-xl-0"
                        />
                        <span>{intl.formatMessage(message)}</span>
                      </Button>
                    </LinkContainer>
                  </Nav.Link>
                ))}
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
            <Button className="minimal brand-link d-inline-block">Stash</Button>
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
            onClick={handleReviewClick}
            title="Review unrated video or video without tags"
          >
            <Icon icon={faStarHalfStroke} className="mr-1" />
            Review
          </button>
          <Navbar.Toggle className="nav-menu-toggle ml-sm-2">
            <Icon icon={expanded ? faTimes : faBars} />
          </Navbar.Toggle>
        </Nav>
      </Navbar>
    </>
  );
};
