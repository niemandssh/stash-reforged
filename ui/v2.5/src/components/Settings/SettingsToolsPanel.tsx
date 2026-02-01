import React from "react";
import { Button } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { Link } from "react-router-dom";
import { Setting } from "./Inputs";
import { SettingSection } from "./SettingSection";
import { PatchContainerComponent } from "src/patch";
import { ExternalLink } from "../Shared/ExternalLink";
import { useScanAllScenesForThreats } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";

const SettingsToolsSection = PatchContainerComponent("SettingsToolsSection");

export const SettingsToolsPanel: React.FC = () => {
  const intl = useIntl();
  const Toast = useToast();
  const [scanAllScenesForThreats] = useScanAllScenesForThreats();

  async function onScanAllScenesForThreats() {
    try {
      const result = await scanAllScenesForThreats();
      const jobId = result.data?.scanAllScenesForThreats;
      Toast.success(
        intl.formatMessage(
          { id: "config.tasks.added_job_to_queue" },
          { operation_name: jobId ? `Scan all scenes for threats (job ${jobId})` : "Scan all scenes for threats" }
        )
      );
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <>
      <SettingSection headingID="config.tools.heading">
        <SettingsToolsSection>
          <Setting
            heading={
              <ExternalLink href="/playground">
                <Button>
                  <FormattedMessage id="config.tools.graphql_playground" />
                </Button>
              </ExternalLink>
            }
          />
        </SettingsToolsSection>
      </SettingSection>
      <SettingSection headingID="config.tools.scene_tools">
        <SettingsToolsSection>
          <Setting
            heading={
              <Link to="/sceneFilenameParser">
                <Button>
                  <FormattedMessage id="config.tools.scene_filename_parser.title" />
                </Button>
              </Link>
            }
          />

          <Setting
            heading={
              <Link to="/sceneDuplicateChecker">
                <Button>
                  <FormattedMessage id="config.tools.scene_duplicate_checker" />
                </Button>
              </Link>
            }
          />

          <Setting
            headingID="actions.scan_all_scenes_for_threats"
            subHeadingID="config.tools.scan_all_scenes_for_threats_desc"
          >
            <Button variant="secondary" onClick={onScanAllScenesForThreats}>
              <FormattedMessage id="actions.scan_all_scenes_for_threats" />
            </Button>
          </Setting>
        </SettingsToolsSection>
      </SettingSection>
    </>
  );
};
