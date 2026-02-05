import React from "react";
import { Button, OverlayTrigger, Tooltip } from "react-bootstrap";
import { useIntl } from "react-intl";
import { Icon } from "src/components/Shared/Icon";
import { faCopy, faPaste } from "@fortawesome/free-solid-svg-icons";
import { useToast } from "src/hooks/Toast";
import { Tag } from "./TagSelect";
import {
  useTagCreate,
  queryFindTagsForSelect,
} from "src/core/StashService";
import { ListFilterModel } from "src/models/list-filter/filter";
import * as GQL from "src/core/generated-graphql";

// Serialization format: tagName::tagColor (one per line)
// Example:
// Blonde::#ffcc00
// Blue Eyes::#0099ff
// No Color Tag

const TAG_SEPARATOR = "\n";
const TAG_COLOR_SEPARATOR = "::";

export function serializeTags(tags: Tag[]): string {
  return tags
    .map((tag) => {
      if (tag.color) {
        return `${tag.name}${TAG_COLOR_SEPARATOR}${tag.color}`;
      }
      return tag.name;
    })
    .join(TAG_SEPARATOR);
}

export interface ParsedTag {
  name: string;
  color?: string;
}

export function deserializeTags(text: string): ParsedTag[] {
  if (!text || !text.trim()) {
    return [];
  }

  return text
    .split(TAG_SEPARATOR)
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
    .map((line) => {
      const separatorIndex = line.lastIndexOf(TAG_COLOR_SEPARATOR);
      if (separatorIndex > 0) {
        const name = line.substring(0, separatorIndex);
        const color = line.substring(separatorIndex + TAG_COLOR_SEPARATOR.length);
        // Validate color format (hex color)
        if (/^#[0-9A-Fa-f]{3,6}$/.test(color)) {
          return { name, color };
        }
      }
      return { name: line };
    });
}

interface ITagCopyPasteProps {
  tags: Tag[];
  onSetTags: (tags: Tag[]) => void;
  className?: string;
}

export const TagCopyPaste: React.FC<ITagCopyPasteProps> = ({
  tags,
  onSetTags,
  className = "",
}) => {
  const intl = useIntl();
  const Toast = useToast();
  const [createTag] = useTagCreate();

  const handleCopy = async () => {
    try {
      const serialized = serializeTags(tags);
      await navigator.clipboard.writeText(serialized);
      Toast.success(
        intl.formatMessage(
          { id: "toast.tags_copied" },
          { count: tags.length }
        )
      );
    } catch (error) {
      Toast.error(
        intl.formatMessage({ id: "toast.clipboard_error" })
      );
    }
  };

  const handlePaste = async () => {
    try {
      const text = await navigator.clipboard.readText();
      const parsedTags = deserializeTags(text);

      if (parsedTags.length === 0) {
        Toast.error(
          intl.formatMessage({ id: "toast.no_tags_in_clipboard" })
        );
        return;
      }

      // Find existing tags by name
      const newTags: Tag[] = [];
      const tagsToCreate: ParsedTag[] = [];

      for (const parsedTag of parsedTags) {
        // Check if tag already exists in current selection
        const existingInSelection = tags.find(
          (t) => t.name.toLowerCase() === parsedTag.name.toLowerCase()
        );
        if (existingInSelection) {
          // Tag already selected, skip
          continue;
        }

        // Search for existing tag in database
        const filter = new ListFilterModel(GQL.FilterMode.Tags);
        filter.searchTerm = parsedTag.name;
        filter.currentPage = 1;
        filter.itemsPerPage = 100;

        const result = await queryFindTagsForSelect(filter);
        const foundTag = result.data.findTags.tags.find(
          (t) => t.name.toLowerCase() === parsedTag.name.toLowerCase()
        );

        if (foundTag) {
          newTags.push({
            id: foundTag.id,
            name: foundTag.name,
            sort_name: foundTag.sort_name,
            aliases: foundTag.aliases,
            image_path: foundTag.image_path,
            is_pose_tag: foundTag.is_pose_tag,
            color: foundTag.color,
          });
        } else {
          tagsToCreate.push(parsedTag);
        }
      }

      // Create new tags
      for (const tagToCreate of tagsToCreate) {
        try {
          const result = await createTag({
            variables: {
              input: {
                name: tagToCreate.name,
                color: tagToCreate.color,
              },
            },
          });

          if (result.data?.tagCreate) {
            newTags.push({
              id: result.data.tagCreate.id,
              name: result.data.tagCreate.name,
              aliases: [],
              is_pose_tag: false,
              color: tagToCreate.color,
            });
            Toast.success(
              intl.formatMessage(
                { id: "toast.created_entity" },
                {
                  entity: intl.formatMessage({ id: "tag" }).toLocaleLowerCase(),
                  entity_name: tagToCreate.name,
                }
              )
            );
          }
        } catch (error) {
          Toast.error(error);
        }
      }

      // Merge with existing tags
      if (newTags.length > 0) {
        const mergedTags = [...tags, ...newTags];
        // Remove duplicates
        const uniqueTags = mergedTags.filter(
          (tag, index, arr) => arr.findIndex((t) => t.id === tag.id) === index
        );
        onSetTags(uniqueTags);
        Toast.success(
          intl.formatMessage(
            { id: "toast.tags_pasted" },
            { count: newTags.length }
          )
        );
      } else {
        Toast.success(
          intl.formatMessage({ id: "toast.all_tags_already_present" })
        );
      }
    } catch (error) {
      Toast.error(
        intl.formatMessage({ id: "toast.clipboard_error" })
      );
    }
  };

  const copyTooltip = (
    <Tooltip id="copy-tags-tooltip">
      {intl.formatMessage({ id: "actions.copy_tags" })}
    </Tooltip>
  );

  const pasteTooltip = (
    <Tooltip id="paste-tags-tooltip">
      {intl.formatMessage({ id: "actions.paste_tags" })}
    </Tooltip>
  );

  return (
    <div className={`tag-copy-paste-buttons ${className}`}>
      <OverlayTrigger placement="top" overlay={copyTooltip}>
        <Button
          variant="link"
          size="sm"
          className="tag-copy-paste-btn"
          onClick={handleCopy}
          disabled={tags.length === 0}
        >
          <Icon icon={faCopy} />
        </Button>
      </OverlayTrigger>
      <OverlayTrigger placement="top" overlay={pasteTooltip}>
        <Button
          variant="link"
          size="sm"
          className="tag-copy-paste-btn"
          onClick={handlePaste}
        >
          <Icon icon={faPaste} />
        </Button>
      </OverlayTrigger>
    </div>
  );
};

// Version for PoseTagSelector that works with tag IDs
interface IPoseTagCopyPasteProps {
  selectedTagIds: string[];
  allPoseTags: Array<{ id: string; name: string; color?: string | null }>;
  onSelectionChange: (tagIds: string[]) => void;
  className?: string;
}

export const PoseTagCopyPaste: React.FC<IPoseTagCopyPasteProps> = ({
  selectedTagIds,
  allPoseTags,
  onSelectionChange,
  className = "",
}) => {
  const intl = useIntl();
  const Toast = useToast();

  const selectedTags = allPoseTags.filter((t) => selectedTagIds.includes(t.id));

  const handleCopy = async () => {
    try {
      const serialized = selectedTags
        .map((tag) => {
          if (tag.color) {
            return `${tag.name}${TAG_COLOR_SEPARATOR}${tag.color}`;
          }
          return tag.name;
        })
        .join(TAG_SEPARATOR);

      await navigator.clipboard.writeText(serialized);
      Toast.success(
        intl.formatMessage(
          { id: "toast.tags_copied" },
          { count: selectedTags.length }
        )
      );
    } catch (error) {
      Toast.error(
        intl.formatMessage({ id: "toast.clipboard_error" })
      );
    }
  };

  const handlePaste = async () => {
    try {
      const text = await navigator.clipboard.readText();
      const parsedTags = deserializeTags(text);

      if (parsedTags.length === 0) {
        Toast.error(
          intl.formatMessage({ id: "toast.no_tags_in_clipboard" })
        );
        return;
      }

      const newTagIds: string[] = [];

      for (const parsedTag of parsedTags) {
        // Find matching pose tag
        const matchingTag = allPoseTags.find(
          (t) => t.name.toLowerCase() === parsedTag.name.toLowerCase()
        );

        if (matchingTag && !selectedTagIds.includes(matchingTag.id)) {
          newTagIds.push(matchingTag.id);
        }
      }

      if (newTagIds.length > 0) {
        onSelectionChange([...selectedTagIds, ...newTagIds]);
        Toast.success(
          intl.formatMessage(
            { id: "toast.tags_pasted" },
            { count: newTagIds.length }
          )
        );
      } else {
        Toast.success(
          intl.formatMessage({ id: "toast.all_tags_already_present" })
        );
      }
    } catch (error) {
      Toast.error(
        intl.formatMessage({ id: "toast.clipboard_error" })
      );
    }
  };

  const copyTooltip = (
    <Tooltip id="copy-pose-tags-tooltip">
      {intl.formatMessage({ id: "actions.copy_tags" })}
    </Tooltip>
  );

  const pasteTooltip = (
    <Tooltip id="paste-pose-tags-tooltip">
      {intl.formatMessage({ id: "actions.paste_tags" })}
    </Tooltip>
  );

  return (
    <div className={`tag-copy-paste-buttons ${className}`}>
      <OverlayTrigger placement="top" overlay={copyTooltip}>
        <Button
          variant="link"
          size="sm"
          className="tag-copy-paste-btn"
          onClick={handleCopy}
          disabled={selectedTagIds.length === 0}
        >
          <Icon icon={faCopy} />
        </Button>
      </OverlayTrigger>
      <OverlayTrigger placement="top" overlay={pasteTooltip}>
        <Button
          variant="link"
          size="sm"
          className="tag-copy-paste-btn"
          onClick={handlePaste}
        >
          <Icon icon={faPaste} />
        </Button>
      </OverlayTrigger>
    </div>
  );
};
