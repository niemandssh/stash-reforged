import React, { useMemo } from "react";
import * as GQL from "src/core/generated-graphql";
import { FilteredSceneList } from "src/components/Scenes/SceneList";
import { usePerformerFilterHook } from "src/core/performers";
import { View } from "src/components/List/views";
import { PatchComponent } from "src/patch";

interface IPerformerDetailsProps {
  active: boolean;
  performer: GQL.PerformerDataFragment;
}

export const PerformerScenesPanel: React.FC<IPerformerDetailsProps> =
  PatchComponent("PerformerScenesPanel", ({ active, performer }) => {
    const filterHook = usePerformerFilterHook(performer);
    const remoteSuggestionsContext = useMemo(
      () =>
        ({
          type: "performer" as const,
          performer: {
            id: performer.id,
            name: performer.name,
            disambiguation: performer.disambiguation,
          },
        }),
      [performer.id, performer.name, performer.disambiguation]
    );
    return (
      <FilteredSceneList
        filterHook={filterHook}
        alterQuery={active}
        view={View.PerformerScenes}
        remoteSuggestionsContext={remoteSuggestionsContext}
      />
    );
  });
