import React, { useMemo, useCallback } from "react";
import { Tabs, Tab } from "react-bootstrap";
import { useHistory, useLocation } from "react-router-dom";

import { GeneralStats } from "./GeneralStats";
import { OCountStats } from "./OCountStats";
import "./OCountStats.scss";

type TabKey = "general" | "o-count";

const validTabs: TabKey[] = ["general", "o-count"];

function isTabKey(tab: string): tab is TabKey {
  return validTabs.includes(tab as TabKey);
}

const StatsPage: React.FC = () => {
  const history = useHistory();
  const location = useLocation();

  const tabFromLocation = useMemo(() => {
    const params = new URLSearchParams(location.search);
    const tab = params.get("tab");
    return tab && isTabKey(tab) ? tab : "general";
  }, [location.search]);

  const setTabKey = useCallback(
    (newTabKey: string | null) => {
      if (!newTabKey) newTabKey = "general";
      if (newTabKey === tabFromLocation) return;

      if (validTabs.includes(newTabKey as TabKey)) {
        const params = new URLSearchParams(location.search);
        params.set("tab", newTabKey);
        history.replace(`${location.pathname}?${params.toString()}`);
      }
    },
    [tabFromLocation, history, location.pathname, location.search]
  );

  return (
    <div className="row">
      <div className="col-12">
        <Tabs
          id="stats-tabs"
          mountOnEnter
          unmountOnExit
          activeKey={tabFromLocation}
          onSelect={setTabKey}
          style={{ textAlign: "center" }}
        >
          <Tab eventKey="general" title="General">
            <GeneralStats />
          </Tab>
          <Tab eventKey="o-count" title="O-Count">
            <OCountStats />
          </Tab>
        </Tabs>
      </div>
    </div>
  );
};

export default StatsPage;
