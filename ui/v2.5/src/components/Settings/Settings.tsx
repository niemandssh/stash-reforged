import React from "react";
import { Tab, Nav, Row, Col, Form } from "react-bootstrap";
import { Redirect, useLocation, NavLink } from "react-router-dom";
import { FormattedMessage } from "react-intl";
import { Helmet } from "react-helmet";
import { useTitleProps } from "src/hooks/title";
import { SettingsAboutPanel } from "./SettingsAboutPanel";
import { SettingsConfigurationPanel } from "./SettingsSystemPanel";
import { SettingsInterfacePanel } from "./SettingsInterfacePanel/SettingsInterfacePanel";
import { SettingsLogsPanel } from "./SettingsLogsPanel";
import { SettingsTasksPanel } from "./Tasks/SettingsTasksPanel";
import { SettingsCustomPanel } from "./SettingsCustomPanel";
import { SettingsPluginsPanel } from "./SettingsPluginsPanel";
import { SettingsScrapingPanel } from "./SettingsScrapingPanel";
import { SettingsToolsPanel } from "./SettingsToolsPanel";
import { SettingsServicesPanel } from "./SettingsServicesPanel";
import { SettingsContext, useSettings } from "./context";
import { SettingsLibraryPanel } from "./SettingsLibraryPanel";
import { SettingsSecurityPanel } from "./SettingsSecurityPanel";
import Changelog from "../Changelog/Changelog";

const validTabs = [
  "tasks",
  "custom",
  "library",
  "interface",
  "security",
  "metadata-providers",
  "services",
  "system",
  "plugins",
  "logs",
  "tools",
  "changelog",
  "about",
] as const;
type TabKey = (typeof validTabs)[number];

const defaultTab: TabKey = "tasks";

function isTabKey(tab: string | null): tab is TabKey {
  return validTabs.includes(tab as TabKey);
}

const SettingTabs: React.FC<{ tab: TabKey }> = ({ tab }) => {
  const { advancedMode, setAdvancedMode } = useSettings();

  const titleProps = useTitleProps({ id: "settings" });

  return (
    <Tab.Container activeKey={tab} id="configuration-tabs">
      <Helmet {...titleProps} />
      <Row>
        <Col id="settings-menu-container" sm={3} md={3} xl={2}>
          <Nav variant="pills" className="flex-column">
            <Nav.Item>
              <NavLink
                to="/settings?tab=tasks"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="tasks" as="span">
                  <FormattedMessage id="config.categories.tasks" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=custom"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="custom" as="span">
                  <FormattedMessage id="config.categories.custom" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=library"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="library" as="span">
                  <FormattedMessage id="library" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=interface"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="interface" as="span">
                  <FormattedMessage id="config.categories.interface" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=security"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="security" as="span">
                  <FormattedMessage id="config.categories.security" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=metadata-providers"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="metadata-providers" as="span">
                  <FormattedMessage id="config.categories.metadata_providers" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=services"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="services" as="span">
                  <FormattedMessage id="config.categories.services" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=system"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="system" as="span">
                  <FormattedMessage id="config.categories.system" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=plugins"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="plugins" as="span">
                  <FormattedMessage id="config.categories.plugins" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=logs"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="logs" as="span">
                  <FormattedMessage id="config.categories.logs" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=tools"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="tools" as="span">
                  <FormattedMessage id="config.categories.tools" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=changelog"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="changelog" as="span">
                  <FormattedMessage id="config.categories.changelog" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=about"
                className="nav-link"
                activeClassName="active"
              >
                <Nav.Link eventKey="about" as="span">
                  <FormattedMessage id="config.categories.about" />
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <div className="advanced-switch">
                <Form.Label htmlFor="advanced-settings">
                  <FormattedMessage id="config.advanced_mode" />
                </Form.Label>
                <Form.Switch
                  id="advanced-settings"
                  checked={advancedMode}
                  onChange={() => setAdvancedMode(!advancedMode)}
                />
              </div>
            </Nav.Item>
            <hr className="d-sm-none" />
          </Nav>
        </Col>
        <Col
          id="settings-container"
          sm={{ offset: 3 }}
          md={{ offset: 3 }}
          xl={{ offset: 2 }}
        >
          <Tab.Content className="mx-auto">
            <Tab.Pane eventKey="library">
              <SettingsLibraryPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="interface">
              <SettingsInterfacePanel />
            </Tab.Pane>
            <Tab.Pane eventKey="security">
              <SettingsSecurityPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="tasks">
              <SettingsTasksPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="custom">
              <SettingsCustomPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="services" unmountOnExit>
              <SettingsServicesPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="tools" unmountOnExit>
              <SettingsToolsPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="metadata-providers" unmountOnExit>
              <SettingsScrapingPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="system">
              <SettingsConfigurationPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="plugins" unmountOnExit>
              <SettingsPluginsPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="logs" unmountOnExit>
              <SettingsLogsPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="changelog" unmountOnExit>
              <Changelog />
            </Tab.Pane>
            <Tab.Pane eventKey="about" unmountOnExit>
              <SettingsAboutPanel />
            </Tab.Pane>
          </Tab.Content>
        </Col>
      </Row>
    </Tab.Container>
  );
};

export const Settings: React.FC = () => {
  const location = useLocation();
  const tab = new URLSearchParams(location.search).get("tab");

  if (!isTabKey(tab)) {
    return (
      <Redirect
        to={{
          ...location,
          search: `tab=${defaultTab}`,
        }}
      />
    );
  }

  return (
    <SettingsContext>
      <SettingTabs tab={tab} />
    </SettingsContext>
  );
};

export default Settings;
