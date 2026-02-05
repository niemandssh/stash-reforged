import React from "react";
import { Button, OverlayTrigger, Tooltip } from "react-bootstrap";
import { useIntl } from "react-intl";
import { Icon } from "src/components/Shared/Icon";
import { faCopy, faPaste } from "@fortawesome/free-solid-svg-icons";
import { useToast } from "src/hooks/Toast";

const DATE_SEPARATOR = "\n";

export function serializeDates(dates: string[]): string {
  return dates.join(DATE_SEPARATOR);
}

export function deserializeDates(text: string): string[] {
  if (!text || !text.trim()) {
    return [];
  }

  return text
    .split(DATE_SEPARATOR)
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
    .filter((line) => {
      // Validate ISO date format
      const date = new Date(line);
      return !isNaN(date.getTime());
    });
}

interface IHistoryCopyPasteProps {
  history: string[];
  onAddDates: (dates: string[]) => void;
  historyType: "o" | "omg";
  className?: string;
}

export const HistoryCopyPaste: React.FC<IHistoryCopyPasteProps> = ({
  history,
  onAddDates,
  historyType,
  className = "",
}) => {
  const intl = useIntl();
  const Toast = useToast();

  const handleCopy = async () => {
    try {
      const serialized = serializeDates(history);
      await navigator.clipboard.writeText(serialized);
      Toast.success(
        intl.formatMessage(
          { id: "toast.history_dates_copied" },
          { count: history.length }
        )
      );
    } catch (error) {
      Toast.error(intl.formatMessage({ id: "toast.clipboard_error" }));
    }
  };

  const handlePaste = async () => {
    try {
      const text = await navigator.clipboard.readText();
      const parsedDates = deserializeDates(text);

      if (parsedDates.length === 0) {
        Toast.error(intl.formatMessage({ id: "toast.no_dates_in_clipboard" }));
        return;
      }

      // Filter out dates that already exist
      const existingDates = new Set(history);
      const newDates = parsedDates.filter((date) => !existingDates.has(date));

      if (newDates.length === 0) {
        Toast.success(
          intl.formatMessage({ id: "toast.all_dates_already_present" })
        );
        return;
      }

      onAddDates(newDates);
      Toast.success(
        intl.formatMessage(
          { id: "toast.history_dates_pasted" },
          { count: newDates.length }
        )
      );
    } catch (error) {
      Toast.error(intl.formatMessage({ id: "toast.clipboard_error" }));
    }
  };

  const copyTooltip = (
    <Tooltip id={`copy-${historyType}-history-tooltip`}>
      {intl.formatMessage({ id: "actions.copy_dates" })}
    </Tooltip>
  );

  const pasteTooltip = (
    <Tooltip id={`paste-${historyType}-history-tooltip`}>
      {intl.formatMessage({ id: "actions.paste_dates" })}
    </Tooltip>
  );

  return (
    <span className={`history-copy-paste-buttons ${className}`}>
      <OverlayTrigger placement="top" overlay={copyTooltip}>
        <Button
          variant="link"
          size="sm"
          className="history-copy-paste-btn"
          onClick={handleCopy}
          disabled={history.length === 0}
        >
          <Icon icon={faCopy} />
        </Button>
      </OverlayTrigger>
      <OverlayTrigger placement="top" overlay={pasteTooltip}>
        <Button
          variant="link"
          size="sm"
          className="history-copy-paste-btn"
          onClick={handlePaste}
        >
          <Icon icon={faPaste} />
        </Button>
      </OverlayTrigger>
    </span>
  );
};
