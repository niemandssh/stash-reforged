import React from "react";
import { useIntl } from "react-intl";
import { SettingSection } from "./SettingSection";
import { Setting } from "./Inputs";
import { HotkeyRecorder } from "./HotkeyRecorder";
import { useSettings } from "./context";
import { defaultHotkeys, getHotkey, HotkeyAction } from "src/core/config";
import { hotkeyGroups } from "./hotkeyDefs";

export const SettingsHotkeysPanel: React.FC = () => {
  const intl = useIntl();
  const { ui, saveUI } = useSettings();

  function setHotkey(action: HotkeyAction, combo: string) {
    saveUI({
      hotkeys: {
        ...ui.hotkeys,
        [action]: combo,
      },
    });
  }

  return (
    <div className="settings-hotkeys-panel">
      <div className="hotkeys-hint text-muted">
        {intl.formatMessage({ id: "config.hotkeys.esc_closes_modals" })}
      </div>
      {hotkeyGroups.map((group) => (
        <SettingSection key={group.categoryID} headingID={group.categoryID}>
          {group.items.map(({ action, labelID }) => (
            <Setting key={action} id={`hotkey-${action}`} headingID={labelID}>
              <HotkeyRecorder
                id={`hotkey-${action}-record`}
                value={getHotkey(ui.hotkeys, action)}
                defaultValue={defaultHotkeys[action]}
                onChange={(combo) => setHotkey(action, combo)}
              />
            </Setting>
          ))}
        </SettingSection>
      ))}
    </div>
  );
};
