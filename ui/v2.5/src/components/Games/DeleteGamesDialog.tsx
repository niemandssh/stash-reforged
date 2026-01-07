import React, { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { faTrashAlt } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { ModalComponent } from "../Shared/Modal";
import { useToast } from "src/hooks/Toast";
import { useGameDestroy } from "src/core/StashService";

interface IProps {
  selected: GQL.SlimGameDataFragment[];
  onClose: (confirmed: boolean) => void;
}

export const DeleteGamesDialog: React.FC<IProps> = ({ selected, onClose }) => {
  const intl = useIntl();
  const Toast = useToast();
  const [isDeleting, setIsDeleting] = useState(false);
  const [destroyGames] = useGameDestroy();

  const singularEntity = intl.formatMessage({ id: "game" });
  const pluralEntity = intl.formatMessage({ id: "games" });

  const header = intl.formatMessage(
    { id: "dialogs.delete_entity_title" },
    { count: selected.length, singularEntity, pluralEntity }
  );
  const message = intl.formatMessage(
    { id: "dialogs.delete_entity_desc" },
    { count: selected.length, singularEntity, pluralEntity }
  );
  const toastMessage = intl.formatMessage(
    { id: "toast.delete_past_tense" },
    { count: selected.length, singularEntity, pluralEntity }
  );

  async function onDelete() {
    setIsDeleting(true);
    try {
      await destroyGames({
        variables: {
          input: { ids: selected.map((game) => game.id) },
        },
      });
      Toast.success(toastMessage);
      onClose(true);
    } catch (error) {
      Toast.error(error);
      onClose(false);
    } finally {
      setIsDeleting(false);
    }
  }

  return (
    <ModalComponent
      show
      icon={faTrashAlt}
      header={header}
      accept={{
        variant: "danger",
        onClick: onDelete,
        text: intl.formatMessage({ id: "actions.delete" }),
      }}
      cancel={{
        onClick: () => onClose(false),
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
      }}
      isRunning={isDeleting}
    >
      <p>{message}</p>
      <ul>
        {selected.slice(0, 5).map((game) => (
          <li key={game.id}>{game.title}</li>
        ))}
        {selected.length > 5 && (
          <li>
            <FormattedMessage
              id="dialogs.delete_object_overflow"
              values={{
                count: selected.length - 5,
                singularEntity,
                pluralEntity,
              }}
            />
          </li>
        )}
      </ul>
    </ModalComponent>
  );
};
