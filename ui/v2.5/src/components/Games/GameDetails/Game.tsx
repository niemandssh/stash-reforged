import React, { useEffect, useMemo, useState } from "react";
import { RouteComponentProps, useHistory } from "react-router-dom";
import { Helmet } from "react-helmet";
import { Button, Dropdown, Nav, Tab } from "react-bootstrap";
import { FormattedDate, FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { useTitleProps } from "src/hooks/title";
import {
  useFindGame,
  useGameAddO,
  useGameAddOmg,
  useGameIncrementView,
  useGameUpdate,
} from "src/core/StashService";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { ErrorMessage } from "src/components/Shared/ErrorMessage";
import { Icon } from "src/components/Shared/Icon";
import {
  faEllipsisV,
  faChevronRight,
  faChevronLeft,
} from "@fortawesome/free-solid-svg-icons";
import {
  OCounterButton,
  OMGCounterButton,
  ViewCountButton,
} from "src/components/Shared/CountButton";
import { OrganizedButton } from "src/components/Scenes/SceneDetails/OrganizedButton";
import { GameDetailPanel } from "./GameDetailPanel";
import { GameHistoryPanel } from "./GameHistoryPanel";
import { GameEditPanel } from "./GameEditPanel";
import { useToast } from "src/hooks/Toast";
import { DeleteGamesDialog } from "../DeleteGamesDialog";
import { RatingSystem } from "src/components/Shared/Rating/RatingSystem";

interface IParams {
  id: string;
  tab?: string;
}

const tabKeyMap = {
  details: "game-details-panel",
  history: "game-history-panel",
  edit: "game-edit-panel",
} as const;

type RouteTab = keyof typeof tabKeyMap;

type TabKey = (typeof tabKeyMap)[RouteTab];

const routeTabs: ReadonlyArray<RouteTab> = ["details", "history", "edit"];

const tabKeyValues: ReadonlyArray<TabKey> = [
  tabKeyMap.details,
  tabKeyMap.history,
  tabKeyMap.edit,
];

const isTabKey = (value: string): value is TabKey =>
  (tabKeyValues as ReadonlyArray<string>).includes(value);

const isRouteTab = (value: string): value is RouteTab =>
  (routeTabs as ReadonlyArray<string>).includes(value);

const getTabKeyFromRoute = (routeTab?: string): TabKey =>
  routeTab && isRouteTab(routeTab) ? tabKeyMap[routeTab] : tabKeyMap.details;

const getRouteFromTabKey = (tabKey: TabKey): RouteTab => {
  const entry = Object.entries(tabKeyMap).find(([, value]) => value === tabKey);
  return (entry?.[0] as RouteTab) ?? "details";
};

interface IGamePageProps {
  game: GQL.GameDataFragment;
  initialTab?: string;
}

const GamePage: React.FC<IGamePageProps> = ({ game, initialTab }) => {
  const history = useHistory();
  const intl = useIntl();
  const Toast = useToast();
  const titleProps = useTitleProps("games");

  const [updateGame, { loading: updatingGame }] = useGameUpdate();
  const [addO] = useGameAddO();
  const [addOmg] = useGameAddOmg();
  const [incrementView] = useGameIncrementView();

  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const [activeTabKey, setActiveTabKey] = useState<TabKey>(
    getTabKeyFromRoute(initialTab)
  );

  useEffect(() => {
    setActiveTabKey(getTabKeyFromRoute(initialTab));
  }, [initialTab]);

  const pageTitle = useMemo(() => {
    const entity = intl.formatMessage({ id: "games" });
    if (!game.title) return entity;
    return `${game.title} - ${entity}`;
  }, [game.title, intl]);

  const helmetProps = {
    ...titleProps,
    title: pageTitle,
  };

  async function onSave(input: GQL.GameCreateInput) {
    await updateGame({
      variables: {
        input: {
          id: game.id,
          ...input,
        },
      },
    });
    Toast.success(
      intl.formatMessage(
        { id: "toast.updated_entity" },
        { entity: intl.formatMessage({ id: "game" }).toLocaleLowerCase() }
      )
    );
  }

  function onSelectTab(key: string | null) {
    if (!key || !isTabKey(key)) {
      return;
    }

    setActiveTabKey(key);
    const routeKey = getRouteFromTabKey(key);
    history.replace(`/games/${game.id}/${routeKey}`);
  }

  function goToHistoryTab() {
    const historyKey: TabKey = tabKeyMap.history;
    setActiveTabKey(historyKey);
    history.replace(`/games/${game.id}/history`);
  }

  async function handleIncrementO() {
    try {
      await addO({ variables: { id: game.id } });
    } catch (error) {
      Toast.error(error);
    }
  }

  async function handleIncrementOmg() {
    try {
      await addOmg({ variables: { id: game.id } });
    } catch (error) {
      Toast.error(error);
    }
  }

  async function handleIncrementView() {
    try {
      await incrementView({ variables: { id: game.id } });
    } catch (error) {
      Toast.error(error);
    }
  }

  async function toggleOrganized() {
    try {
      await updateGame({
        variables: {
          input: {
            id: game.id,
            organized: !game.organized,
          },
        },
      });
    } catch (error) {
      Toast.error(error);
    }
  }

  async function setRating(value: number | null) {
    try {
      await updateGame({
        variables: {
          input: {
            id: game.id,
            rating100: value,
          },
        },
      });
    } catch (error) {
      Toast.error(error);
    }
  }

  function maybeRenderDeleteDialog() {
    if (!isDeleteDialogOpen) return null;

    return (
      <DeleteGamesDialog
        selected={[game as GQL.SlimGameDataFragment]}
        onClose={(deleted) => {
          setIsDeleteDialogOpen(false);
          if (deleted) {
            history.push("/games");
          }
        }}
      />
    );
  }

  function renderOperations() {
    return (
      <Dropdown>
        <Dropdown.Toggle
          variant="secondary"
          id="game-operations-menu"
          className="minimal"
          title={intl.formatMessage({ id: "operations" })}
        >
          <Icon icon={faEllipsisV} />
        </Dropdown.Toggle>
        <Dropdown.Menu className="bg-secondary text-white">
          <Dropdown.Item
            className="bg-secondary text-white"
            onClick={() => setIsDeleteDialogOpen(true)}
          >
            <FormattedMessage
              id="actions.delete"
              values={{ entityType: intl.formatMessage({ id: "game" }) }}
            />
          </Dropdown.Item>
        </Dropdown.Menu>
      </Dropdown>
    );
  }

  function getCollapseButtonIcon() {
    return collapsed ? faChevronRight : faChevronLeft;
  }

  function renderCoverPanel() {
    return (
      <div className="game-cover-panel text-center">
        {game.image_path ? (
          <img
            src={game.image_path}
            alt={game.title ?? ""}
            className="img-fluid"
          />
        ) : (
          <p className="text-muted">
            <FormattedMessage id="none" />
          </p>
        )}
      </div>
    );
  }

  return (
    <div className="row">
      <Helmet {...helmetProps} />
      {maybeRenderDeleteDialog()}
      <div className={`gallery-tabs ${collapsed ? "collapsed" : ""}`}>
        <div>
          {renderCoverPanel()}
          <div className="gallery-header-container">
            <h3 className="gallery-header">{game.title}</h3>
          </div>
          <div className="gallery-subheader">
            {!!game.date && (
              <span className="date" data-value={game.date}>
                <FormattedDate value={game.date} format="long" timeZone="utc" />
              </span>
            )}
          </div>
          <div className="gallery-toolbar">
            <div className="gallery-toolbar-row">
              <span className="gallery-toolbar-group">
                <RatingSystem
                  value={game.rating100 ?? undefined}
                  onSetRating={setRating}
                  clickToRate
                  withoutContext
                />
              </span>
            </div>
            <div className="gallery-toolbar-row">
              <span className="gallery-toolbar-group">
                <ViewCountButton
                  value={game.play_count ?? 0}
                  onIncrement={handleIncrementView}
                  onValueClicked={goToHistoryTab}
                />
                <OCounterButton
                  value={game.o_counter ?? 0}
                  onIncrement={handleIncrementO}
                  onValueClicked={goToHistoryTab}
                />
                <OMGCounterButton
                  value={game.omgCounter ?? 0}
                  onIncrement={handleIncrementOmg}
                  onValueClicked={goToHistoryTab}
                />
                <OrganizedButton
                  organized={game.organized}
                  onClick={toggleOrganized}
                  loading={updatingGame}
                />
              </span>
              <span className="gallery-toolbar-group">
                {renderOperations()}
              </span>
            </div>
          </div>
        </div>
        <Tab.Container
          id="gallery-details-container"
          activeKey={activeTabKey}
          onSelect={onSelectTab}
        >
          <div>
            <Nav variant="tabs" className="mr-auto">
              <Nav.Item>
                <Nav.Link eventKey={tabKeyMap.details}>
                  <FormattedMessage id="details" />
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey={tabKeyMap.history}>
                  <FormattedMessage id="history" />
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey={tabKeyMap.edit}>
                  <FormattedMessage id="actions.edit" />
                </Nav.Link>
              </Nav.Item>
            </Nav>

            <Tab.Content>
              <Tab.Pane eventKey={tabKeyMap.details}>
                <GameDetailPanel game={game} />
              </Tab.Pane>
              <Tab.Pane eventKey={tabKeyMap.history}>
                <GameHistoryPanel game={game} />
              </Tab.Pane>
              <Tab.Pane eventKey={tabKeyMap.edit} mountOnEnter>
                <GameEditPanel
                  isVisible={activeTabKey === tabKeyMap.edit}
                  game={game}
                  onSubmit={onSave}
                  onDelete={() => setIsDeleteDialogOpen(true)}
                />
              </Tab.Pane>
            </Tab.Content>
          </div>
        </Tab.Container>
      </div>
      <div className="gallery-divider d-none d-xl-block">
        <Button onClick={() => setCollapsed(!collapsed)}>
          <Icon className="fa-fw" icon={getCollapseButtonIcon()} />
        </Button>
      </div>
      <div className={`gallery-container ${collapsed ? "expanded" : ""}`}></div>
    </div>
  );
};

const GameLoader: React.FC<RouteComponentProps<IParams>> = ({ match }) => {
  const { id, tab } = match.params;
  const { data, loading, error } = useFindGame(id);

  if (loading) return <LoadingIndicator />;
  if (error) return <ErrorMessage error={error.message} />;
  if (!data?.findGame) return <ErrorMessage error="Game not found" />;

  return <GamePage game={data.findGame} initialTab={tab} />;
};

export default GameLoader;
