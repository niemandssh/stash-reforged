import * as GQL from "src/core/generated-graphql";
import { getQueryClient } from "src/core/query-client";
import {
  TagsCriterion,
  TagsCriterionOption,
} from "src/models/list-filter/criteria/tags";
import { ListFilterModel } from "src/models/list-filter/filter";

export const useTagFilterHook = (
  tag: GQL.TagDataFragment,
  showSubTagContent?: boolean
) => {
  return (filter: ListFilterModel) => {
    const tagValue = { id: tag.id, label: tag.name };
    // if tag is already present, then we modify it, otherwise add
    let tagCriterion = filter.criteria.find((c) => {
      return c.criterionOption.type === "tags";
    }) as TagsCriterion | undefined;

    if (tagCriterion) {
      if (
        tagCriterion.modifier === GQL.CriterionModifier.IncludesAll ||
        tagCriterion.modifier === GQL.CriterionModifier.Includes
      ) {
        // add the tag if not present
        if (
          !tagCriterion.value.items.find((p) => {
            return p.id === tag.id;
          })
        ) {
          tagCriterion.value.items.push(tagValue);
        }
      } else {
        // overwrite
        tagCriterion.value.items = [tagValue];
      }

      tagCriterion.modifier = GQL.CriterionModifier.IncludesAll;
    } else {
      tagCriterion = new TagsCriterion(TagsCriterionOption);
      tagCriterion.value = {
        items: [tagValue],
        excluded: [],
        depth: showSubTagContent ? -1 : 0,
      };
      tagCriterion.modifier = GQL.CriterionModifier.IncludesAll;
      filter.criteria.push(tagCriterion);
    }

    return filter;
  };
};

interface ITagRelationTuple {
  parents: GQL.SlimTagDataFragment[];
  children: GQL.SlimTagDataFragment[];
}

export const tagRelationHook = (
  tag: GQL.SlimTagDataFragment | GQL.TagDataFragment,
  old: ITagRelationTuple,
  updated: ITagRelationTuple
) => {
  const queryClient = getQueryClient();

  // Invalidate tag queries to force refetch when tag relations change
  // This replaces Apollo cache manipulation with TanStack Query invalidation
  queryClient.invalidateQueries({
    queryKey: ["tags", tag.id],
  });

  // Invalidate queries for all affected tags (parents and children)
  const allAffectedTagIds = new Set<string>();
  old.parents.forEach((t) => allAffectedTagIds.add(t.id));
  old.children.forEach((t) => allAffectedTagIds.add(t.id));
  updated.parents.forEach((t) => allAffectedTagIds.add(t.id));
  updated.children.forEach((t) => allAffectedTagIds.add(t.id));

  allAffectedTagIds.forEach((tagId) => {
    queryClient.invalidateQueries({
      queryKey: ["tags", tagId],
    });
  });

  // Also invalidate the tags list query
  queryClient.invalidateQueries({
    queryKey: ["tags"],
  });
};
