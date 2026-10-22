import { useContext } from "react";
import { ConfigurationContext } from "./Config";
import { HotkeyAction, getHotkey } from "src/core/config";

// Resolves the effective key combo for a configurable hotkey action,
// falling back to the built-in default if the user hasn't overridden it.
export function useHotkey(action: HotkeyAction): string {
  const { configuration } = useContext(ConfigurationContext);
  return getHotkey(configuration?.ui.hotkeys, action);
}
