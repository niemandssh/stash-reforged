import {
  faEllipsisV,
  faPlus,
  faTrash,
} from "@fortawesome/free-solid-svg-icons";
import React from "react";
import { Link } from "react-router-dom";
import { Button, Dropdown } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { PerformerPopover } from "src/components/Performers/PerformerPopover";
import { AlertModal } from "src/components/Shared/Alert";
import { Counter } from "src/components/Shared/Counter";
import { DateInput } from "src/components/Shared/DateInput";
import { Icon } from "src/components/Shared/Icon";
import { ModalComponent } from "src/components/Shared/Modal";
import {
  useSceneDecrementO,
  useSceneDecrementPlayCount,
  useSceneIncrementO,
  useSceneIncrementPlayCount,
  useSceneResetO,
  useSceneResetOmg,
  useSceneResetPlayCount,
  useSceneResetActivity,
  useSceneAddOmg,
  useSceneDeleteOmg,
} from "src/core/StashService";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";
import { SceneCountAttributeModal } from "./SceneCountAttributeModal";
import { TextField } from "src/utils/field";
import TextUtils from "src/utils/text";
import { HistoryCopyPaste } from "./HistoryCopyPaste";

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
    <div className="scene-history">
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

type HistoryEntry = {
  timestamp: string;
  performer_ids: (string | number)[];
};

const HistoryWithAttribution: React.FC<{
  className?: string;
  entries: HistoryEntry[];
  performers: { id: string; name: string }[];
  unknownDate?: string;
  onRemove: (date: string) => void;
  noneID: string;
}> = ({ className, entries, performers, unknownDate, noneID, onRemove }) => {
  const intl = useIntl();

  if (entries.length === 0) {
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

  function renderAttribution(entry: HistoryEntry) {
    if (!entry.performer_ids || entry.performer_ids.length === 0) {
      return (
        <span>
          {intl.formatMessage({
            id: "history.whole_scene",
            defaultMessage: "Whole scene",
          })}
        </span>
      );
    }
    const resolved = entry.performer_ids
      .map((id) => {
        const pid = String(id);
        const p = performers.find((p) => p.id === pid);
        return p ? { id: pid, name: p.name } : null;
      })
      .filter(Boolean) as { id: string; name: string }[];
    return (
      <>
        {resolved.map((p, i) => (
          <React.Fragment key={p.id}>
            {i > 0 && ", "}
            <PerformerPopover id={p.id} placement="top">
              <Link
                to={`/performers/${p.id}`}
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted"
              >
                {p.name}
              </Link>
            </PerformerPopover>
          </React.Fragment>
        ))}
      </>
    );
  }

  return (
    <div className="scene-history">
      <ul className={className}>
        {entries.map((entry, index) => (
          <li key={index}>
            <span>{renderDate(entry.timestamp)}</span>
            <span className="ml-2 text-muted small">
              {renderAttribution(entry)}
            </span>
            <Button
              className="remove-date-button"
              size="sm"
              variant="minimal"
              onClick={() => onRemove(entry.timestamp)}
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
  showResetResumeDuration: boolean;
  onAddDate: () => void;
  onClearDates: () => void;
  resetResume: () => void;
  resetDuration: () => void;
}> = ({
  hasHistory,
  showResetResumeDuration,
  onAddDate,
  onClearDates,
  resetResume,
  resetDuration,
}) => {
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
        {showResetResumeDuration && (
          <Dropdown.Item
            className="bg-secondary text-white"
            onClick={() => resetResume()}
          >
            <FormattedMessage id="actions.reset_resume_time" />
          </Dropdown.Item>
        )}
        {showResetResumeDuration && (
          <Dropdown.Item
            className="bg-secondary text-white"
            onClick={() => resetDuration()}
          >
            <FormattedMessage id="actions.reset_play_duration" />
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

interface ISceneHistoryProps {
  scene: GQL.SceneDataFragment;
}

export const SceneHistoryPanel: React.FC<ISceneHistoryProps> = ({ scene }) => {
  const intl = useIntl();
  const Toast = useToast();

  const [dialogs, setDialogs] = React.useState({
    playHistory: false,
    oHistory: false,
    omgHistory: false,
    addPlay: false,
    addO: false,
    addOMG: false,
  });
  const [showOAttributeModal, setShowOAttributeModal] = React.useState(false);
  const [showOmgAttributeModal, setShowOmgAttributeModal] = React.useState(false);

  function setDialogPartial(partial: Partial<typeof dialogs>) {
    setDialogs({ ...dialogs, ...partial });
  }

  const [incrementPlayCount] = useSceneIncrementPlayCount();
  const [decrementPlayCount] = useSceneDecrementPlayCount();
  const [clearPlayCount] = useSceneResetPlayCount();
  const [incrementOCount] = useSceneIncrementO(scene.id);
  const [decrementOCount] = useSceneDecrementO(scene.id);
  const [resetO] = useSceneResetO(scene.id);
  const [resetOmg] = useSceneResetOmg(scene.id);
  const [addOmg] = useSceneAddOmg(scene.id);
  const [deleteOmg] = useSceneDeleteOmg(scene.id);
  const [resetResume] = useSceneResetActivity(scene.id, true, false);
  const [resetDuration] = useSceneResetActivity(scene.id, false, true);

  function dateStringToISOString(time: string) {
    const date = TextUtils.stringToFuzzyDateTime(time);
    if (!date) return null;
    return date.toISOString();
  }

  function handleAddPlayDate(time?: string) {
    incrementPlayCount({
      variables: {
        id: scene.id,
        times: time ? [time] : undefined,
      },
    });
  }

  function handleDeletePlayDate(time: string) {
    decrementPlayCount({
      variables: {
        id: scene.id,
        times: time ? [time] : undefined,
      },
    });
  }

  function handleClearPlayDates() {
    setDialogPartial({ playHistory: false });
    clearPlayCount({
      variables: {
        id: scene.id,
      },
    });
  }

  function handleAddODate(time?: string, performerIds?: string[] | null) {
    incrementOCount({
      variables: {
        id: scene.id,
        times: time ? [time] : undefined,
        performer_ids: performerIds ?? undefined,
      } as GQL.SceneAddOMutationVariables,
    });
  }

  function openAddOOrModal() {
    setShowOAttributeModal(true);
  }

  function handleAddODates(times: string[]) {
    if (times.length === 0) return;
    incrementOCount({
      variables: {
        id: scene.id,
        times: times,
      },
    });
  }

  function handleDeleteODate(time: string) {
    decrementOCount({
      variables: {
        id: scene.id,
        times: [time],
      },
    });
  }

  function handleClearODates() {
    setDialogPartial({ oHistory: false });
    resetO({
      variables: {
        id: scene.id,
      },
    });
  }

  function handleAddOMGDate(time?: string, performerIds?: string[] | null) {
    addOmg({
      variables: {
        id: scene.id,
        times: time ? [time] : undefined,
        performer_ids: performerIds ?? undefined,
      } as GQL.SceneAddOmgMutationVariables,
    });
  }

  function openAddOmgOrModal() {
    setShowOmgAttributeModal(true);
  }

  async function handleOAttributeConfirm(
    performerIds: string[] | null,
    date?: string
  ) {
    try {
      const timeIso = date ? dateStringToISOString(date) : undefined;
      handleAddODate(timeIso, performerIds);
      setShowOAttributeModal(false);
    } catch (e) {
      Toast.error(e);
    }
  }

  async function handleOmgAttributeConfirm(
    performerIds: string[] | null,
    date?: string
  ) {
    try {
      const timeIso = date ? dateStringToISOString(date) : undefined;
      handleAddOMGDate(timeIso, performerIds);
      setShowOmgAttributeModal(false);
    } catch (e) {
      Toast.error(e);
    }
  }

  function handleAddOMGDates(times: string[]) {
    if (times.length === 0) return;
    addOmg({
      variables: {
        id: scene.id,
        times: times,
      },
    });
  }

  function handleDeleteOMGDate(time: string) {
    deleteOmg({
      variables: {
        id: scene.id,
        times: [time],
      },
    });
  }

  function handleClearOMGDates() {
    setDialogPartial({ omgHistory: false });
    resetOmg({
      variables: {
        id: scene.id,
      },
    });
  }

  async function handleResetResume() {
    try {
      await resetResume({
        variables: {
          id: scene.id,
          reset_resume: true,
          reset_duration: false,
        },
      });

      Toast.success(
        intl.formatMessage({ id: "toast.scene_with_similars_updated" })
      );
    } catch (e) {
      Toast.error(e);
    }
  }

  async function handleResetDuration() {
    try {
      await resetDuration({
        variables: {
          id: scene.id,
          reset_resume: false,
          reset_duration: true,
        },
      });

      Toast.success(
        intl.formatMessage({ id: "toast.scene_with_similars_updated" })
      );
    } catch (e) {
      Toast.error(e);
    }
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
          text={intl.formatMessage({ id: "dialogs.clear_omg_history_confirm" })}
          confirmButtonText={intl.formatMessage({ id: "actions.clear" })}
          onConfirm={() => handleClearOMGDates()}
          onCancel={() => setDialogPartial({ omgHistory: false })}
        />
        {/* add conditions here so that date is generated correctly */}
        {dialogs.addPlay && (
          <DatePickerModal
            show
            onClose={(t) => {
              const tt = t ? dateStringToISOString(t) : null;
              if (tt) {
                handleAddPlayDate(tt);
              }
              setDialogPartial({ addPlay: false });
            }}
          />
        )}
      </>
    );
  }

  const playHistory = (scene.play_history ?? []).filter(
    (h) => h != null
  ) as string[];
  const oHistory = (scene.o_history ?? []).filter((h) => h != null) as string[];
  const omgHistory = (scene.omg_history ?? []).filter(
    (h) => h != null
  ) as string[];

  const scenePerformers = scene.performers ?? [];
  const performerNameMap = scenePerformers.map((p) => ({ id: p.id, name: p.name }));

  const oHistoryEntries: HistoryEntry[] =
    (scene as GQL.SceneDataFragment & {
      o_history_entries?: { timestamp: string; performer_ids: string[] }[];
    }).o_history_entries?.map((e) => ({
      timestamp: e.timestamp,
      performer_ids: e.performer_ids ?? [],
    })) ??
    oHistory.map((t) => ({ timestamp: t, performer_ids: [] as string[] }));

  const omgHistoryEntries: HistoryEntry[] =
    (scene as GQL.SceneDataFragment & {
      omg_history_entries?: { timestamp: string; performer_ids: string[] }[];
    }).omg_history_entries?.map((e) => ({
      timestamp: e.timestamp,
      performer_ids: e.performer_ids ?? [],
    })) ??
    omgHistory.map((t) => ({ timestamp: t, performer_ids: [] as string[] }));

  return (
    <div>
      {maybeRenderDialogs()}
      <SceneCountAttributeModal
        show={showOAttributeModal}
        onHide={() => setShowOAttributeModal(false)}
        type="o"
        performers={scenePerformers}
        includeDatePicker
        onConfirm={handleOAttributeConfirm}
      />
      <SceneCountAttributeModal
        show={showOmgAttributeModal}
        onHide={() => setShowOmgAttributeModal(false)}
        type="omg"
        performers={scenePerformers}
        includeDatePicker
        onConfirm={handleOmgAttributeConfirm}
      />
      <div className="play-history">
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
                showResetResumeDuration={true}
                onAddDate={() => setDialogPartial({ addPlay: true })}
                onClearDates={() => setDialogPartial({ playHistory: true })}
                resetResume={() => handleResetResume()}
                resetDuration={() => handleResetDuration()}
              />
            </span>
          </h5>
        </div>

        <History
          history={playHistory ?? []}
          noneID="playdate_recorded_no"
          unknownDate={scene.created_at}
          onRemove={(t) => handleDeletePlayDate(t)}
        />
        <dl className="details-list">
          <TextField
            id="media_info.play_duration"
            value={TextUtils.secondsToTimestamp(scene.play_duration ?? 0)}
          />
        </dl>
      </div>

      <div className="o-history">
        <div className="history-header">
          <h5>
            <span>
              <FormattedMessage id="o_history" />
              <Counter count={oHistoryEntries.length} hideZero />
            </span>
            <span>
              <HistoryCopyPaste
                history={oHistoryEntries.map((e) => e.timestamp)}
                onAddDates={handleAddODates}
                historyType="o"
              />
              <Button
                size="sm"
                variant="minimal"
                className="add-date-button"
                title={intl.formatMessage({ id: "actions.add_o" })}
                onClick={() => openAddOOrModal()}
              >
                <Icon icon={faPlus} />
              </Button>
              <HistoryMenu
                hasHistory={oHistoryEntries.length > 0}
                showResetResumeDuration={false}
                onAddDate={() => openAddOOrModal()}
                onClearDates={() => setDialogPartial({ oHistory: true })}
                resetResume={() => handleResetResume()}
                resetDuration={() => handleResetDuration()}
              />
            </span>
          </h5>
        </div>
        <HistoryWithAttribution
          entries={oHistoryEntries}
          performers={performerNameMap}
          noneID="odate_recorded_no"
          unknownDate={scene.created_at}
          onRemove={(t) => handleDeleteODate(t)}
        />
      </div>

      <div className="omg-history">
        <div className="history-header">
          <h5>
            <span>
              <FormattedMessage id="omg_history" />
              <Counter count={omgHistoryEntries.length} hideZero />
            </span>
            <span>
              <HistoryCopyPaste
                history={omgHistoryEntries.map((e) => e.timestamp)}
                onAddDates={handleAddOMGDates}
                historyType="omg"
              />
              <Button
                size="sm"
                variant="minimal"
                className="add-date-button"
                title={intl.formatMessage({ id: "actions.add_omg" })}
                onClick={() => openAddOmgOrModal()}
              >
                <Icon icon={faPlus} />
              </Button>
              <HistoryMenu
                hasHistory={omgHistoryEntries.length > 0}
                showResetResumeDuration={false}
                onAddDate={() => openAddOmgOrModal()}
                onClearDates={() => setDialogPartial({ omgHistory: true })}
                resetResume={() => handleResetResume()}
                resetDuration={() => handleResetDuration()}
              />
            </span>
          </h5>
        </div>
        <HistoryWithAttribution
          entries={omgHistoryEntries}
          performers={performerNameMap}
          noneID="omgdate_recorded_no"
          unknownDate={scene.created_at}
          onRemove={(t) => handleDeleteOMGDate(t)}
        />
      </div>
    </div>
  );
};

export default SceneHistoryPanel;
