import {
  faEllipsisV,
  faPlus,
  faTrash,
} from "@fortawesome/free-solid-svg-icons";
import React from "react";
import { Button, Dropdown } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { AlertModal } from "src/components/Shared/Alert";
import { Counter } from "src/components/Shared/Counter";
import { DateInput } from "src/components/Shared/DateInput";
import { Icon } from "src/components/Shared/Icon";
import { ModalComponent } from "src/components/Shared/Modal";
import {
  useGameAddO,
  useGameDeleteO,
  useGameResetO,
  useGameAddOmg,
  useGameDeleteOmg,
  useGameResetOmg,
  useGameIncrementView,
  useGameDeleteView,
  useGameResetViews,
} from "src/core/StashService";
import * as GQL from "src/core/generated-graphql";
import TextUtils from "src/utils/text";

const History: React.FC<{
  className?: string;
  history: string[];
  unknownDate?: string;
  onRemove: (date: string) => void;
  noneID: string;
}> = ({ className, history, unknownDate, noneID, onRemove }) => {
  const intl = useIntl();

  if (history.length === 0) {
    return (
      <div>
        <FormattedMessage id={noneID} />
      </div>
    );
  }

  function renderDate(date: string) {
    if (date === unknownDate) {
      return intl.formatMessage({ id: "unknown_date" });
    }

    return TextUtils.formatDateTime(intl, date);
  }

  return (
    <div className="gallery-history">
      <ul className={className}>
        {history.map((playdate, index) => (
          <li key={index}>
            <span>{renderDate(playdate)}</span>
            <Button
              className="remove-date-button"
              size="sm"
              variant="minimal"
              onClick={() => onRemove(playdate)}
              title={intl.formatMessage({ id: "actions.remove_date" })}
            >
              <Icon icon={faTrash} />
            </Button>
          </li>
        ))}
      </ul>
    </div>
  );
};

const HistoryMenu: React.FC<{
  hasHistory: boolean;
  onAddDate: () => void;
  onClearDates: () => void;
}> = ({ hasHistory, onAddDate, onClearDates }) => {
  const intl = useIntl();

  return (
    <Dropdown className="history-operations-dropdown">
      <Dropdown.Toggle
        variant="secondary"
        className="minimal"
        title={intl.formatMessage({ id: "operations" })}
      >
        <Icon icon={faEllipsisV} />
      </Dropdown.Toggle>
      <Dropdown.Menu className="bg-secondary text-white">
        <Dropdown.Item
          className="bg-secondary text-white"
          onClick={() => onAddDate()}
        >
          <FormattedMessage id="actions.add_manual_date" />
        </Dropdown.Item>
        {hasHistory && (
          <Dropdown.Item
            className="bg-secondary text-white"
            onClick={() => onClearDates()}
          >
            <FormattedMessage id="actions.clear_date_data" />
          </Dropdown.Item>
        )}
      </Dropdown.Menu>
    </Dropdown>
  );
};

const DatePickerModal: React.FC<{
  show: boolean;
  onClose: (t?: string) => void;
}> = ({ show, onClose }) => {
  const intl = useIntl();
  const [date, setDate] = React.useState<string>(
    TextUtils.dateTimeToString(new Date())
  );

  return (
    <ModalComponent
      show={show}
      header={<FormattedMessage id="actions.choose_date" />}
      accept={{
        onClick: () => onClose(date),
        text: intl.formatMessage({ id: "actions.confirm" }),
      }}
      cancel={{
        variant: "secondary",
        onClick: () => onClose(),
        text: intl.formatMessage({ id: "actions.cancel" }),
      }}
    >
      <div>
        <DateInput value={date} onValueChange={(d) => setDate(d)} isTime />
      </div>
    </ModalComponent>
  );
};

interface IGameHistoryProps {
  game: GQL.GameDataFragment;
}

export const GameHistoryPanel: React.FC<IGameHistoryProps> = ({ game }) => {
  const intl = useIntl();

  const [dialogs, setDialogs] = React.useState({
    playHistory: false,
    oHistory: false,
    omgHistory: false,
    addPlay: false,
    addO: false,
    addOmg: false,
  });

  function setDialogPartial(partial: Partial<typeof dialogs>) {
    setDialogs({ ...dialogs, ...partial });
  }

  const [incrementView] = useGameIncrementView();
  const [deleteView] = useGameDeleteView();
  const [resetViews] = useGameResetViews();
  const [incrementOCount] = useGameAddO();
  const [decrementOCount] = useGameDeleteO();
  const [resetO] = useGameResetO();
  const [incrementOmgCount] = useGameAddOmg();
  const [decrementOmgCount] = useGameDeleteOmg();
  const [resetOmg] = useGameResetOmg();

  function dateStringToISOString(time: string) {
    const date = TextUtils.stringToFuzzyDateTime(time);
    if (!date) return null;
    return date.toISOString();
  }

  function handleAddPlayDate() {
    incrementView({
      variables: {
        id: game.id,
      },
    });
  }

  function handleAddODate(time?: string) {
    incrementOCount({
      variables: {
        id: game.id,
        times: time ? [time] : undefined,
      },
    });
  }

  function handleAddOmgDate(time?: string) {
    incrementOmgCount({
      variables: {
        id: game.id,
        times: time ? [time] : undefined,
      },
    });
  }

  function handleDeletePlayDate(time: string) {
    deleteView({
      variables: {
        id: game.id,
        times: [time],
      },
    });
  }

  function handleDeleteODate(time: string) {
    decrementOCount({
      variables: {
        id: game.id,
        times: [time],
      },
    });
  }

  function handleDeleteOmgDate(time: string) {
    decrementOmgCount({
      variables: {
        id: game.id,
        times: [time],
      },
    });
  }

  function handleClearPlayDates() {
    setDialogPartial({ playHistory: false });
    resetViews({ variables: { id: game.id } });
  }

  function handleClearODates() {
    setDialogPartial({ oHistory: false });
    resetO({ variables: { id: game.id } });
  }

  function handleClearOmgDates() {
    setDialogPartial({ omgHistory: false });
    resetOmg({ variables: { id: game.id } });
  }

  function maybeRenderDialogs() {
    return (
      <>
        <AlertModal
          show={dialogs.playHistory}
          text={intl.formatMessage({
            id: "dialogs.clear_play_history_confirm",
          })}
          confirmButtonText={intl.formatMessage({ id: "actions.clear" })}
          onConfirm={() => handleClearPlayDates()}
          onCancel={() => setDialogPartial({ playHistory: false })}
        />
        <AlertModal
          show={dialogs.oHistory}
          text={intl.formatMessage({ id: "dialogs.clear_o_history_confirm" })}
          confirmButtonText={intl.formatMessage({ id: "actions.clear" })}
          onConfirm={() => handleClearODates()}
          onCancel={() => setDialogPartial({ oHistory: false })}
        />
        <AlertModal
          show={dialogs.omgHistory}
          text={intl.formatMessage({
            id: "dialogs.clear_omg_history_confirm",
          })}
          confirmButtonText={intl.formatMessage({ id: "actions.clear" })}
          onConfirm={() => handleClearOmgDates()}
          onCancel={() => setDialogPartial({ omgHistory: false })}
        />
        {dialogs.addPlay && (
          <DatePickerModal
            show
            onClose={(t) => {
              const tt = t ? dateStringToISOString(t) : null;
              if (tt) {
                handleAddPlayDate();
              }
              setDialogPartial({ addPlay: false });
            }}
          />
        )}
        {dialogs.addO && (
          <DatePickerModal
            show
            onClose={(t) => {
              const tt = t ? dateStringToISOString(t) : null;
              if (tt) {
                handleAddODate(tt);
              }
              setDialogPartial({ addO: false });
            }}
          />
        )}
        {dialogs.addOmg && (
          <DatePickerModal
            show
            onClose={(t) => {
              const tt = t ? dateStringToISOString(t) : null;
              if (tt) {
                handleAddOmgDate(tt);
              }
              setDialogPartial({ addOmg: false });
            }}
          />
        )}
      </>
    );
  }

  const playHistory = (game.view_history ?? []).filter(
    (h) => h != null
  ) as string[];
  const oHistory = (game.o_history ?? []).filter((h) => h != null) as string[];
  const omgHistory = (game.omg_history ?? []).filter(
    (h) => h != null
  ) as string[];

  return (
    <div>
      {maybeRenderDialogs()}

      <div className="play-history mb-3">
        <div className="history-header">
          <h5>
            <span>
              <FormattedMessage id="play_history" />
              <Counter count={playHistory.length} hideZero />
            </span>
            <span>
              <Button
                size="sm"
                variant="minimal"
                className="add-date-button"
                title={intl.formatMessage({ id: "actions.add_play" })}
                onClick={() => handleAddPlayDate()}
              >
                <Icon icon={faPlus} />
              </Button>
              <HistoryMenu
                hasHistory={playHistory.length > 0}
                onAddDate={() => setDialogPartial({ addPlay: true })}
                onClearDates={() => setDialogPartial({ playHistory: true })}
              />
            </span>
          </h5>
        </div>
        <History
          className="play-history"
          history={playHistory}
          onRemove={(date) => handleDeletePlayDate(date)}
          noneID="playdate_recorded_no"
        />
      </div>

      <div className="o-history mb-3">
        <div className="history-header">
          <h5>
            <span>
              <FormattedMessage id="o_history" />
              <Counter count={oHistory.length} hideZero />
            </span>
            <span>
              <Button
                size="sm"
                variant="minimal"
                className="add-date-button"
                title={intl.formatMessage({ id: "actions.add_o" })}
                onClick={() => handleAddODate()}
              >
                <Icon icon={faPlus} />
              </Button>
              <HistoryMenu
                hasHistory={oHistory.length > 0}
                onAddDate={() => setDialogPartial({ addO: true })}
                onClearDates={() => setDialogPartial({ oHistory: true })}
              />
            </span>
          </h5>
        </div>
        <History
          history={oHistory}
          noneID="odate_recorded_no"
          unknownDate={game.created_at}
          onRemove={(t) => handleDeleteODate(t)}
        />
      </div>

      <div className="omg-history">
        <div className="history-header">
          <h5>
            <span>
              <FormattedMessage id="omg_history" />
              <Counter count={omgHistory.length} hideZero />
            </span>
            <span>
              <Button
                size="sm"
                variant="minimal"
                className="add-date-button"
                title={intl.formatMessage({ id: "actions.add_omg" })}
                onClick={() => handleAddOmgDate()}
              >
                <Icon icon={faPlus} />
              </Button>
              <HistoryMenu
                hasHistory={omgHistory.length > 0}
                onAddDate={() => setDialogPartial({ addOmg: true })}
                onClearDates={() => setDialogPartial({ omgHistory: true })}
              />
            </span>
          </h5>
        </div>
        <History
          history={omgHistory}
          noneID="omgdate_recorded_no"
          unknownDate={game.created_at}
          onRemove={(t) => handleDeleteOmgDate(t)}
        />
      </div>
    </div>
  );
};

export default GameHistoryPanel;
