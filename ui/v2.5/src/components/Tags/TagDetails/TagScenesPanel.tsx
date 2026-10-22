import React, { useMemo } from "react";
import * as GQL from "src/core/generated-graphql";
import { FilteredSceneList } from "src/components/Scenes/SceneList";
import { useTagFilterHook } from "src/core/tags";
import { View } from "src/components/List/views";

interface ITagScenesPanel {
  active: boolean;
  tag: GQL.TagDataFragment;
  showSubTagContent?: boolean;
}

export const TagScenesPanel: React.FC<ITagScenesPanel> = ({
  active,
  tag,
  showSubTagContent,
}) => {
  const filterHook = useTagFilterHook(tag, showSubTagContent);
  const remoteSuggestionsContext = useMemo(
    () =>
      ({
        type: "tag" as const,
        tag: {
          id: tag.id,
          name: tag.name,
          aliases: tag.aliases,
        },
      }),
    [tag.id, tag.name, tag.aliases]
  );
  return (
    <FilteredSceneList
      filterHook={filterHook}
      alterQuery={active}
      view={View.TagScenes}
      remoteSuggestionsContext={remoteSuggestionsContext}
    />
  );
};
