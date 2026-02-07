import React from "react";
import cloneDeep from "lodash-es/cloneDeep";
import { useIntl } from "react-intl";
import { useHistory } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { ItemList, ItemListContext } from "../List/ItemList";
import { ListFilterModel } from "src/models/list-filter/filter";
import { DisplayMode } from "src/models/list-filter/types";
import { View } from "../List/views";
import { queryFindGames, useFindGames } from "src/core/StashService";
import { GameCardGrid } from "./GameCardGrid";
import { GameListTable } from "./GameListTable";
import { DeleteGamesDialog } from "./DeleteGamesDialog";

import "./styles.scss";

interface IProps {
  filterHook?: (filter: ListFilterModel) => ListFilterModel;
  view?: View;
}

function getItems(result: GQL.FindGamesQueryResult) {
  return result.data?.findGames?.games ?? [];
}

function getCount(result: GQL.FindGamesQueryResult) {
  return result.data?.findGames?.count ?? 0;
}

export const GameList: React.FC<IProps> = ({ filterHook, view }) => {
  const intl = useIntl();
  const history = useHistory();

  const filterMode = GQL.FilterMode.Games;

  async function viewRandom(
    result: GQL.FindGamesQueryResult,
    filter: ListFilterModel
  ) {
    const payload = result.data?.findGames;
    if (!payload || payload.count === 0) return;

    const index = Math.floor(Math.random() * payload.count);
    const filterCopy = cloneDeep(filter);
    filterCopy.itemsPerPage = 1;
    filterCopy.currentPage = index + 1;
    const singleResult = await queryFindGames(filterCopy);
    const game = (singleResult.data as any).findGames?.games?.[0];
    if (game) {
      history.push(`/games/${game.id}`);
    }
  }

  const otherOperations = [
    {
      text: intl.formatMessage({ id: "actions.view_random" }),
      onClick: viewRandom,
    },
  ];

  function renderContent(
    result: GQL.FindGamesQueryResult,
    filter: ListFilterModel,
    selectedIds: Set<string>,
    onSelectChange: (id: string, selected: boolean, shiftKey: boolean) => void
  ) {
    if (!result.data?.findGames) return null;

    if (filter.displayMode === DisplayMode.Grid) {
      return (
        <GameCardGrid
          games={result.data.findGames.games}
          selectedIds={selectedIds}
          zoomIndex={filter.zoomIndex}
          onSelectChange={onSelectChange}
        />
      );
    }

    return (
      <GameListTable
        games={result.data.findGames.games}
        selectedIds={selectedIds}
        onSelectChange={onSelectChange}
      />
    );
  }

  function renderDeleteDialog(
    selected: GQL.SlimGameDataFragment[],
    onClose: (confirmed: boolean) => void
  ) {
    return <DeleteGamesDialog selected={selected} onClose={onClose} />;
  }

  return (
    <ItemListContext
      filterMode={filterMode}
      useResult={useFindGames}
      getItems={getItems}
      getCount={getCount}
      filterHook={filterHook}
      view={view}
      selectable
    >
      <ItemList
        zoomable
        view={view}
        otherOperations={otherOperations}
        renderContent={renderContent}
        renderDeleteDialog={renderDeleteDialog}
      />
    </ItemListContext>
  );
};
