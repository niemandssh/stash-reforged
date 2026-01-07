import React from "react";
import { Tab, Nav, Row, Col, Form, Button } from "react-bootstrap";
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
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "tasks";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="tasks" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "tasks" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.tasks" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=custom"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "custom";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="custom" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "custom" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.custom" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=library"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "library";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="library" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "library" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="library" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=interface"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "interface";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="interface" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "interface" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.interface" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=security"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "security";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="security" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "security" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.security" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=metadata-providers"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "metadata-providers";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="metadata-providers" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "metadata-providers" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.metadata_providers" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=services"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "services";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="services" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "services" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.services" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=system"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "system";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="system" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "system" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.system" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=plugins"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "plugins";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="plugins" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "plugins" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.plugins" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=logs"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "logs";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="logs" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "logs" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.logs" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=tools"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "tools";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="tools" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "tools" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.tools" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=changelog"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "changelog";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="changelog" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "changelog" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.changelog" />
                  </Button>
                </Nav.Link>
              </NavLink>
            </Nav.Item>
            <Nav.Item>
              <NavLink
                to="/settings?tab=about"
                className="nav-link"
                isActive={(match, location) => {
                  const params = new URLSearchParams(location.search);
                  return params.get("tab") === "about";
                }}
                activeClassName="active"
              >
                <Nav.Link eventKey="about" as="span">
                  <Button
                    className={`minimal w-100 text-left ${
                      tab === "about" ? "active" : ""
                    }`}
                  >
                    <FormattedMessage id="config.categories.about" />
                  </Button>
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
