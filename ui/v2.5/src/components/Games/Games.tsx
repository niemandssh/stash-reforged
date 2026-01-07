import React from "react";
import { Helmet } from "react-helmet";
import { Route, Switch } from "react-router-dom";
import { useTitleProps } from "src/hooks/title";
import { View } from "../List/views";
import { GameList } from "./GameList";
import Game from "./GameDetails/Game";
import GameCreatePage from "./GameDetails/GameCreate";

const GamesListPage: React.FC = () => {
  return <GameList view={View.Games} />;
};

const GamesRoutes: React.FC = () => {
  const titleProps = useTitleProps({ id: "games" });
  return (
    <>
      <Helmet {...titleProps} />
      <Switch>
        <Route exact path="/games" component={GamesListPage} />
        <Route exact path="/games/new" component={GameCreatePage} />
        <Route path="/games/:id/:tab?" component={Game} />
      </Switch>
    </>
  );
};

export default GamesRoutes;
