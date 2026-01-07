import React from "react";
import { Table, Form, Badge } from "react-bootstrap";
import { Link } from "react-router-dom";
import { useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import TextUtils from "src/utils/text";

interface IProps {
  games: GQL.SlimGameDataFragment[];
  selectedIds: Set<string>;
  onSelectChange: (id: string, selected: boolean, shiftKey: boolean) => void;
}

export const GameListTable: React.FC<IProps> = ({
  games,
  selectedIds,
  onSelectChange,
}) => {
  const intl = useIntl();

  function renderRow(game: GQL.SlimGameDataFragment) {
    return (
      <tr key={game.id}>
        <td className="align-middle">
          <Form.Check
            type="checkbox"
            checked={selectedIds.has(game.id)}
            onChange={(event) =>
              onSelectChange(
                game.id,
                event.currentTarget.checked,
                (event.nativeEvent as MouseEvent).shiftKey
              )
            }
          />
        </td>
        <td className="align-middle">
          <Link to={`/games/${game.id}`} className="ellips-data">
            {game.title}
          </Link>
        </td>
        <td className="align-middle">
          {game.date
            ? TextUtils.formatDate(intl, game.date)
            : TextUtils.formatDate(intl, game.created_at ?? undefined)}
        </td>
        <td className="align-middle">{game.rating100 ?? "-"}</td>
        <td className="align-middle">
          {game.organized ? (
            <Badge variant="success">
              {intl.formatMessage({ id: "organized" })}
            </Badge>
          ) : (
            <span className="text-muted">
              {intl.formatMessage({ id: "not_organized" })}
            </span>
          )}
        </td>
        <td className="align-middle">{game.o_counter}</td>
        <td className="align-middle">{game.omgCounter}</td>
        <td className="align-middle">{game.play_count}</td>
        <td className="align-middle">
          {TextUtils.formatDate(
            intl,
            game.updated_at ?? game.created_at ?? undefined
          )}
        </td>
      </tr>
    );
  }

  return (
    <Table bordered responsive hover size="sm" className="game-list-table">
      <thead>
        <tr>
          <th style={{ width: "2rem" }} />
          <th>{intl.formatMessage({ id: "title" })}</th>
          <th>{intl.formatMessage({ id: "date" })}</th>
          <th>{intl.formatMessage({ id: "rating" })}</th>
          <th>{intl.formatMessage({ id: "organized" })}</th>
          <th>{intl.formatMessage({ id: "o_counter" })}</th>
          <th>{intl.formatMessage({ id: "omg_counter" })}</th>
          <th>{intl.formatMessage({ id: "play_count" })}</th>
          <th>{intl.formatMessage({ id: "updated_at" })}</th>
        </tr>
      </thead>
      <tbody>{games.map(renderRow)}</tbody>
    </Table>
  );
};
