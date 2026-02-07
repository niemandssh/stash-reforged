import React, { useEffect, useMemo, useState } from "react";
import {
  OptionProps,
  components,
  MultiValueGenericProps,
  MultiValueProps,
  SingleValueProps,
  InputProps,
  GroupBase,
} from "react-select";
import cx from "classnames";

import * as GQL from "src/core/generated-graphql";
import {
  useTagCreate,
  queryFindTagsByIDForSelect,
  queryFindTagsForSelect,
} from "src/core/StashService";
import { ConfigurationContext } from "src/hooks/Config";
import { useIntl } from "react-intl";
import { defaultMaxOptionsShown } from "src/core/config";
import { ListFilterModel } from "src/models/list-filter/filter";
import {
  FilterSelectComponent,
  IFilterIDProps,
  IFilterProps,
  IFilterValueProps,
  Option as SelectOption,
} from "../Shared/FilterSelect";
import { useCompare } from "src/hooks/state";
import { TagPopover } from "./TagPopover";
import { Placement } from "react-bootstrap/esm/Overlay";
import { PatchComponent, PatchFunction } from "src/patch";
import {
  generateSearchVariants,
  translateRussianToEnglish,
  translateEnglishToRussian,
  isFuzzyMatch,
  levenshteinDistance,
} from "src/utils/keyboardLayout";

const getContrastColor = (backgroundColor: string): string => {
  if (!backgroundColor) return "#000000";

  let r = 0,
    g = 0,
    b = 0;

  if (backgroundColor.startsWith("#")) {
    const hex = backgroundColor.replace("#", "");
    if (hex.length === 3) {
      r = parseInt(hex[0] + hex[0], 16);
      g = parseInt(hex[1] + hex[1], 16);
      b = parseInt(hex[2] + hex[2], 16);
    } else if (hex.length === 6) {
      r = parseInt(hex.substr(0, 2), 16);
      g = parseInt(hex.substr(2, 2), 16);
      b = parseInt(hex.substr(4, 2), 16);
    }
  }

  const brightness = (r * 299 + g * 587 + b * 114) / 1000;
  return brightness > 128 ? "#000000" : "#ffffff";
};

export type SelectObject = {
  id: string;
  name?: string | null;
  title?: string | null;
};

export type Tag = Pick<
  GQL.Tag,
  | "id"
  | "name"
  | "sort_name"
  | "aliases"
  | "image_path"
  | "is_pose_tag"
  | "color"
>;
type Option = SelectOption<Tag>;

type FindTagsResult = Awaited<
  ReturnType<typeof queryFindTagsForSelect>
>["data"]["findTags"]["tags"];

function sortTagsByRelevance(input: string, tags: FindTagsResult) {
  if (!input) {
    return tags;
  }

  const inputLower = input.toLowerCase().trim();
  const inputWords = inputLower.split(/[\s\-_]+/).filter((w) => w.length > 0);

  // Calculate relevance score for each tag
  // Lower score = higher priority
  function getRelevanceScore(tag: (typeof tags)[0]): number {
    const nameLower = tag.name.toLowerCase();
    const nameWords = nameLower.split(/[\s\-_]+/).filter((w) => w.length > 0);

    // Exact match
    if (nameLower === inputLower) return 0;

    // Name starts with input (exact)
    if (nameLower.startsWith(inputLower)) return 10;

    // First word starts with input (exact)
    if (nameWords[0]?.startsWith(inputLower)) return 20;

    // Name starts with input (fuzzy)
    if (nameWords[0] && inputWords[0]) {
      const firstWordDist = levenshteinDistance(
        inputWords[0],
        nameWords[0].substring(0, inputWords[0].length),
        2
      );
      if (firstWordDist <= 2) {
        return 30 + firstWordDist;
      }
    }

    // Any word starts with input (exact)
    const wordStartIndex = nameWords.findIndex((w) => w.startsWith(inputLower));
    if (wordStartIndex !== -1) return 40 + wordStartIndex;

    // Name contains input
    const includeIndex = nameLower.indexOf(inputLower);
    if (includeIndex !== -1) return 50 + includeIndex;

    // Any word matches input (fuzzy)
    for (let i = 0; i < nameWords.length; i++) {
      const word = nameWords[i];
      if (inputWords[0]) {
        const dist = levenshteinDistance(inputWords[0], word, 2);
        if (dist <= 2) {
          return 60 + i * 10 + dist;
        }
        // Word starts with fuzzy input
        if (word.length >= inputWords[0].length) {
          const prefixDist = levenshteinDistance(
            inputWords[0],
            word.substring(0, inputWords[0].length),
            2
          );
          if (prefixDist <= 2) {
            return 70 + i * 10 + prefixDist;
          }
        }
      }
    }

    // Check aliases
    if (tag.aliases) {
      for (const alias of tag.aliases) {
        const aliasLower = alias.toLowerCase();
        if (aliasLower === inputLower) return 5;
        if (aliasLower.startsWith(inputLower)) return 15;
        if (aliasLower.includes(inputLower)) return 55;
      }
    }

    // Fuzzy match on full name
    if (isFuzzyMatch(input, tag.name)) {
      return 80;
    }

    // No match
    return 100;
  }

  return tags.slice().sort((a, b) => {
    const scoreA = getRelevanceScore(a);
    const scoreB = getRelevanceScore(b);
    if (scoreA !== scoreB) {
      return scoreA - scoreB;
    }
    return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
  });
}

const tagSelectSort = PatchFunction("TagSelect.sort", sortTagsByRelevance);

export type TagSelectProps = IFilterProps &
  IFilterValueProps<Tag> & {
    hoverPlacement?: Placement;
    hoverPlacementLabel?: Placement;
    excludeIds?: string[];
    instanceId?: string;
  };

const TagCustomInput = (
  inputProps: InputProps<
    { value: string; object: Tag },
    boolean,
    GroupBase<{ value: string; object: Tag }>
  >
) => {
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if ((e.ctrlKey || e.metaKey) && e.code === "KeyZ") {
      e.stopPropagation();
    }
    inputProps.onKeyDown?.(e);
  };

  return <components.Input {...inputProps} onKeyDown={handleKeyDown} />;
};

const _TagSelect: React.FC<TagSelectProps> = (props) => {
  const [createTag] = useTagCreate();
  const [currentInputValue, setCurrentInputValue] = useState("");

  const { configuration } = React.useContext(ConfigurationContext);
  const intl = useIntl();
  const maxOptionsShown =
    configuration?.ui.maxOptionsShown ?? defaultMaxOptionsShown;
  const defaultCreatable = !(
    configuration?.interface.disableDropdownCreate.tag ?? false
  );

  const exclude = useMemo(() => props.excludeIds || [], [props.excludeIds]);

  const loadTags = React.useCallback(
    async (input: string): Promise<Option[]> => {
      const searchVariants = generateSearchVariants(input);
      const allResults = new Map<string, FindTagsResult[0]>();

      for (const searchTerm of searchVariants) {
        const filter = new ListFilterModel(GQL.FilterMode.Tags);
        filter.searchTerm = searchTerm;
        filter.currentPage = 1;
        filter.itemsPerPage = maxOptionsShown;
        filter.sortBy = "name";
        filter.sortDirection = GQL.SortDirectionEnum.Asc;
        const query = await queryFindTagsForSelect(filter);
        const tags = query.data.findTags.tags.filter((tag) => {
          return !exclude.includes(tag.id.toString());
        });

        tags.forEach((tag) => {
          allResults.set(tag.id, tag);
        });
      }

      // If no results and input is long enough, try fuzzy search
      // Search with first 2 characters and filter with fuzzy matching
      if (allResults.size === 0 && input.length >= 3) {
        const prefix = input.substring(0, 2);
        const prefixVariants = generateSearchVariants(prefix);

        for (const searchTerm of prefixVariants) {
          const filter = new ListFilterModel(GQL.FilterMode.Tags);
          filter.searchTerm = searchTerm;
          filter.currentPage = 1;
          filter.itemsPerPage = 100; // Load more for fuzzy filtering
          filter.sortBy = "name";
          filter.sortDirection = GQL.SortDirectionEnum.Asc;
          const query = await queryFindTagsForSelect(filter);
          const tags = query.data.findTags.tags.filter((tag) => {
            if (exclude.includes(tag.id.toString())) return false;
            // Apply fuzzy matching
            if (isFuzzyMatch(input, tag.name)) return true;
            // Check aliases
            if (tag.aliases?.some((a) => isFuzzyMatch(input, a))) return true;
            return false;
          });

          tags.forEach((tag) => {
            allResults.set(tag.id, tag);
          });
        }
      }

      const ret = Array.from(allResults.values());
      return tagSelectSort(input, ret).map((tag) => ({
        value: tag.id,
        object: tag,
      }));
    },
    [exclude, maxOptionsShown]
  );

  const TagOption: React.FC<OptionProps<Option, boolean>> = (optionProps) => {
    const { object } = optionProps.data;

    let { name } = object;

    const { inputValue } = optionProps.selectProps;
    let alias: string | undefined = "";
    if (!name.toLowerCase().includes(inputValue.toLowerCase())) {
      alias = object.aliases?.find((a) =>
        a.toLowerCase().includes(inputValue.toLowerCase())
      );
    }

    let thisOptionProps = {
      ...optionProps,
      children: (
        <TagPopover id={object.id} placement={props.hoverPlacement ?? "right"}>
          <span className="react-select-image-option d-flex align-items-center">
            {object.color && (
              <span
                className="tag-color-indicator tag-color-indicator-sm mx-2"
                style={{ backgroundColor: object.color }}
                title={`Цвет тега: ${object.color}`}
              />
            )}
            <span>{name}</span>
            {alias && <span className="alias">&nbsp;({alias})</span>}
          </span>
        </TagPopover>
      ),
    };

    return <components.Option {...thisOptionProps} />;
  };

  const TagMultiValue: React.FC<MultiValueProps<Option, boolean>> = (
    optionProps
  ) => {
    const { object } = optionProps.data;

    const isHighlighted = () => {
      if (!currentInputValue) return false;

      const input = currentInputValue.toLowerCase();
      const tagName = object.name.toLowerCase();

      // Direct match
      if (tagName.includes(input)) return true;

      // Keyboard layout translations
      const englishTranslation = translateRussianToEnglish(currentInputValue);
      const russianTranslation = translateEnglishToRussian(currentInputValue);

      if (
        tagName.includes(englishTranslation.toLowerCase()) ||
        tagName.includes(russianTranslation.toLowerCase())
      ) {
        return true;
      }

      // Fuzzy matching with typo tolerance (only tag name, not aliases)
      if (isFuzzyMatch(currentInputValue, object.name)) {
        return true;
      }

      // Fuzzy match with keyboard layout translations
      if (
        isFuzzyMatch(englishTranslation, object.name) ||
        isFuzzyMatch(russianTranslation, object.name)
      ) {
        return true;
      }

      return false;
    };

    const highlighted = isHighlighted();
    const highlightedClass = highlighted ? "highlighted-tag-chip" : "";

    // Don't apply color style when highlighted - let CSS handle it
    const colorStyle =
      !highlighted && object.color
        ? {
            backgroundColor: object.color,
            color: getContrastColor(object.color),
          }
        : undefined;

    let thisOptionProps = {
      ...optionProps,
      className: `${optionProps.className || ""} ${highlightedClass}`.trim(),
      style: colorStyle,
    };

    return (
      <components.MultiValue
        {...thisOptionProps}
        key={`multi-value-${object.id}`}
      />
    );
  };

  const TagMultiValueLabel: React.FC<
    MultiValueGenericProps<Option, boolean>
  > = (optionProps) => {
    const { object } = optionProps.data;

    const thisOptionProps = {
      ...optionProps,
      children: (
        <TagPopover
          id={object.id}
          placement={props.hoverPlacementLabel ?? "top"}
        >
          <span className="d-flex align-items-center">
            {object.color && (
              <span
                className="tag-color-indicator tag-color-indicator-sm mr-1"
                style={{ backgroundColor: object.color }}
                title={`Цвет тега: ${object.color}`}
              />
            )}
            <span>{object.name}</span>
          </span>
        </TagPopover>
      ),
    };

    return <components.MultiValueLabel {...thisOptionProps} />;
  };

  const TagMultiValueRemove: React.FC<
    MultiValueGenericProps<Option, boolean>
  > = (optionProps) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return <components.MultiValueRemove {...(optionProps as any)} />;
  };

  const TagValueLabel: React.FC<SingleValueProps<Option, boolean>> = (
    optionProps
  ) => {
    const { object } = optionProps.data;

    const thisOptionProps = {
      ...optionProps,
      children: <>{object.name}</>,
    };

    return <components.SingleValue {...thisOptionProps} />;
  };

  const onCreate = async (name: string) => {
    const result = await createTag({
      variables: { input: { name } },
    });
    return {
      value: result.data!.tagCreate!.id,
      item: result.data!.tagCreate!,
      message: "Created tag",
    };
  };

  const getNamedObject = (id: string, name: string) => {
    return {
      id,
      name,
      aliases: [],
      is_pose_tag: false,
    };
  };

  const isValidNewOption = React.useCallback(
    (inputValue: string, options: Tag[]) => {
      if (!inputValue) {
        return false;
      }

      // Check if input exactly matches any selected tag
      const exactMatchWithSelected = props.values?.some((selectedTag) => {
        const selectedName = selectedTag.name.toLowerCase();
        const inputLower = inputValue.toLowerCase();
        const normalizedInput = inputLower.replace(/-/g, " ");
        const normalizedSelected = selectedName.replace(/-/g, " ");

        return (
          selectedName === inputLower ||
          selectedName === normalizedInput ||
          normalizedSelected === inputLower ||
          normalizedSelected === normalizedInput
        );
      });

      // If input exactly matches a selected tag, don't allow creation
      if (exactMatchWithSelected) {
        return false;
      }

      // Check if any existing tag matches (including normalized versions)
      const normalizedInput = inputValue.replace(/-/g, " ");

      return !options.some((o) => {
        const tagName = o.name.toLowerCase();
        const normalizedTagName = tagName.replace(/-/g, " ");

        return (
          tagName === inputValue.toLowerCase() ||
          tagName === normalizedInput.toLowerCase() ||
          normalizedTagName === inputValue.toLowerCase() ||
          normalizedTagName === normalizedInput.toLowerCase() ||
          o.aliases?.some((a) => {
            const aliasName = a.toLowerCase();
            const normalizedAliasName = aliasName.replace(/-/g, " ");
            return (
              aliasName === inputValue.toLowerCase() ||
              aliasName === normalizedInput.toLowerCase() ||
              normalizedAliasName === inputValue.toLowerCase() ||
              normalizedAliasName === normalizedInput.toLowerCase()
            );
          })
        );
      });
    },
    [props.values]
  );

  // Wrap loadOptions to add a dummy "no match" option when no results found
  const loadTagsWithDummy = React.useCallback(
    async (input: string) => {
      const results = await loadTags(input);

      // If no input, return results as is
      if (!input) {
        return results;
      }

      // Check if input exactly matches any selected tag
      const exactMatchWithSelected = props.values?.some((selectedTag) => {
        const selectedName = selectedTag.name.toLowerCase();
        const inputLower = input.toLowerCase();
        const normalizedInput = inputLower.replace(/-/g, " ");
        const normalizedSelected = selectedName.replace(/-/g, " ");

        return (
          selectedName === inputLower ||
          selectedName === normalizedInput ||
          normalizedSelected === inputLower ||
          normalizedSelected === normalizedInput
        );
      });

      // If input exactly matches a selected tag, don't show anything
      if (exactMatchWithSelected) {
        return [];
      }

      // If there are search results, return them without dummy
      if (results.length > 0) {
        return results;
      }

      // If no search results but input exists, check if it's a superset of selected tags
      const isSupersetOfSelected = props.values?.some((selectedTag) => {
        const selectedName = selectedTag.name.toLowerCase();
        const inputLower = input.toLowerCase();
        return inputLower.includes(selectedName) && inputLower !== selectedName;
      });

      // If it's a superset of selected tags, show dummy
      if (isSupersetOfSelected) {
        const dummyOption: Option = {
          value: "__no_match__",
          object: {
            id: "__no_match__",
            name: `Not found. Click below button to create tag`,
            aliases: [],
            is_pose_tag: false,
          },
        };
        return [dummyOption];
      }

      // For completely new input with no results, show create option
      const dummyOption: Option = {
        value: "__no_match__",
        object: {
          id: "__no_match__",
          name: `No results found. Use Tab or click to create "${input}"`,
          aliases: [],
          is_pose_tag: false,
        },
      };
      return [dummyOption];
    },
    [props.values, loadTags]
  );

  // Custom Option that handles the dummy option
  const CustomTagOption: React.FC<OptionProps<Option, boolean>> = (
    optionProps
  ) => {
    const { object } = optionProps.data;
    const intlHook = useIntl();

    // If this is the dummy option, render with custom text but standard disabled styles
    if (object.id === "__no_match__") {
      return (
        <TagOption
          {...optionProps}
          data={{
            ...optionProps.data,
            object: {
              ...object,
              name: intlHook.formatMessage({
                id: "actions.create_tag_no_results",
              }),
            },
          }}
        />
      );
    }

    // Otherwise render normal TagOption
    return <TagOption {...optionProps} />;
  };

  // Wrap onSelect to filter out dummy options
  const handleSelect = React.useCallback(
    (items: Tag[]) => {
      // Filter out dummy options
      const realItems = items.filter((item) => item.id !== "__no_match__");
      if (props.onSelect) {
        props.onSelect(realItems);
      }
    },
    [props]
  );

  // Mark dummy option as disabled
  const isOptionDisabled = React.useCallback((option: Option) => {
    return option.object.id === "__no_match__";
  }, []);

  const selectProps = {
    ...props,
    onSelect: handleSelect,
    isOptionDisabled: isOptionDisabled,
    className: cx(
      "tag-select",
      {
        "tag-select-active": props.active,
      },
      props.className
    ),
    loadOptions: loadTagsWithDummy,
    getNamedObject: getNamedObject,
    isValidNewOption: isValidNewOption,
    components: {
      Option: CustomTagOption,
      MultiValue: TagMultiValue,
      MultiValueLabel: TagMultiValueLabel,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      MultiValueRemove: TagMultiValueRemove as any,
      SingleValue: TagValueLabel,
      Input: TagCustomInput,
    },
    isMulti: props.isMulti ?? false,
    creatable: props.creatable ?? defaultCreatable,
    onCreate: onCreate,
    onInputChange: (inputValue: string) =>
      setCurrentInputValue(inputValue || ""),
    placeholder:
      props.noSelectionString ??
      intl.formatMessage(
        { id: "actions.select_entity" },
        {
          entityType: intl.formatMessage({
            id: props.isMulti ? "tags" : "tag",
          }),
        }
      ),
    closeMenuOnSelect: !props.isMulti,
  };

  return <FilterSelectComponent<Tag, boolean> {...selectProps} />;
};

export const TagSelect = PatchComponent("TagSelect", _TagSelect);

const _TagIDSelect: React.FC<IFilterProps & IFilterIDProps<Tag>> = (props) => {
  const { ids, onSelect: onSelectValues } = props;

  const [values, setValues] = useState<Tag[]>([]);
  const idsChanged = useCompare(ids);

  function onSelect(items: Tag[]) {
    setValues(items);
    onSelectValues?.(items);
  }

  async function loadObjectsByID(idsToLoad: string[]): Promise<Tag[]> {
    const query = await queryFindTagsByIDForSelect(idsToLoad);
    const { tags: loadedTags } = query.data.findTags;

    return loadedTags;
  }

  useEffect(() => {
    if (!idsChanged) {
      return;
    }

    if (!ids || ids?.length === 0) {
      setValues([]);
      return;
    }

    // load the values if we have ids and they haven't been loaded yet
    const filteredValues = values.filter((v) => ids.includes(v.id.toString()));
    if (filteredValues.length === ids.length) {
      return;
    }

    const load = async () => {
      const items = await loadObjectsByID(ids);

      // #4684 - sort items by sort name/name
      const sortedItems = [...items];
      sortedItems.sort((a, b) => {
        const aName = a.sort_name || a.name;
        const bName = b.sort_name || b.name;

        if (aName && bName) {
          return aName.localeCompare(bName);
        }
        return 0;
      });

      setValues(sortedItems);
    };

    load();
  }, [ids, idsChanged, values]);

  return <TagSelect {...props} values={values} onSelect={onSelect} />;
};

export const TagIDSelect = PatchComponent("TagIDSelect", _TagIDSelect);
