import React, { useMemo } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useHistory, useLocation } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { useGameCreate } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";
import { GameEditPanel } from "./GameEditPanel";

const GameCreate: React.FC = () => {
  const history = useHistory();
  const intl = useIntl();
  const Toast = useToast();

  const location = useLocation();
  const query = useMemo(() => new URLSearchParams(location.search), [location]);
  const game = {
    title: query.get("q") ?? undefined,
  };

  const [createGame] = useGameCreate();

  async function onSave(input: GQL.GameCreateInput) {
    const result = await createGame({
      variables: { input },
    });
    if (result.data?.gameCreate) {
      history.push(`/games/${result.data.gameCreate.id}`);
      Toast.success(
        intl.formatMessage(
          { id: "toast.created_entity" },
          { entity: intl.formatMessage({ id: "game" }).toLocaleLowerCase() }
        )
      );
    }
  }

  return (
    <div className="row new-view">
      <div className="col-md-6">
        <h2>
          <FormattedMessage
            id="actions.create_entity"
            values={{ entityType: intl.formatMessage({ id: "game" }) }}
          />
        </h2>
        <GameEditPanel
          game={game}
          isVisible
          onSubmit={onSave}
          onDelete={() => {}}
        />
      </div>
    </div>
  );
};

export default GameCreate;
