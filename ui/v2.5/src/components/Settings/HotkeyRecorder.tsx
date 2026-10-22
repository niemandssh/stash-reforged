import React, { useEffect, useState } from "react";
import { Button } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";

const modifierKeys = ["Control", "Alt", "Shift", "Meta"];

const specialKeyLabels: Record<string, string> = {
  " ": "space",
  Escape: "esc",
  Enter: "enter",
  ArrowUp: "up",
  ArrowDown: "down",
  ArrowLeft: "left",
  ArrowRight: "right",
  Delete: "del",
  Backspace: "backspace",
  Tab: "tab",
};

function modifiersFromEvent(e: KeyboardEvent): string[] {
  const mods: string[] = [];
  if (e.ctrlKey) mods.push("ctrl");
  if (e.altKey) mods.push("alt");
  if (e.shiftKey) mods.push("shift");
  if (e.metaKey) mods.push("meta");
  return mods;
}

function keyToken(e: KeyboardEvent): string | null {
  if (modifierKeys.includes(e.key)) return null;
  return specialKeyLabels[e.key] ?? e.key.toLowerCase();
}

interface IHotkeyRecorder {
  id: string;
  value: string;
  defaultValue: string;
  onChange: (combo: string) => void;
}

// A key-combo recorder for the hotkeys settings panel, modelled after the
// keybinding editors in VSCode and most games: click Record, press the
// desired combo (updates live as modifiers/keys are held), Enter confirms
// the combo shown, Esc cancels and restores the previous value.
export const HotkeyRecorder: React.FC<IHotkeyRecorder> = ({
  id,
  value,
  defaultValue,
  onChange,
}) => {
  const intl = useIntl();
  const [recording, setRecording] = useState(false);
  const [pending, setPending] = useState<string>();

  useEffect(() => {
    if (!recording) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      e.preventDefault();
      e.stopPropagation();

      if (e.key === "Escape") {
        setRecording(false);
        setPending(undefined);
        return;
      }

      if (e.key === "Enter") {
        if (pending) {
          onChange(pending);
        }
        setRecording(false);
        setPending(undefined);
        return;
      }

      const token = keyToken(e);
      const mods = modifiersFromEvent(e);

      if (token === null) {
        // only modifiers held so far - show a live preview
        if (mods.length) setPending(mods.join("+"));
        return;
      }

      setPending([...mods, token].join("+"));
    };

    document.addEventListener("keydown", handleKeyDown, true);
    return () => document.removeEventListener("keydown", handleKeyDown, true);
  }, [recording, pending, onChange]);

  const displayValue = recording
    ? pending ??
      intl.formatMessage({ id: "config.hotkeys.press_keys" })
    : value;

  return (
    <div className="hotkey-recorder">
      <span className={`hotkey-value ${recording ? "recording" : ""}`}>
        {displayValue}
      </span>
      <Button
        id={id}
        size="sm"
        variant={recording ? "danger" : "secondary"}
        onClick={() => {
          setPending(undefined);
          setRecording(!recording);
        }}
      >
        {recording ? (
          <FormattedMessage id="actions.cancel" />
        ) : (
          <FormattedMessage id="config.hotkeys.record" />
        )}
      </Button>
      {value !== defaultValue && (
        <Button
          size="sm"
          variant="minimal"
          title={intl.formatMessage({ id: "config.hotkeys.reset_default" })}
          onClick={() => onChange(defaultValue)}
        >
          <FormattedMessage id="actions.clear" />
        </Button>
      )}
    </div>
  );
};
