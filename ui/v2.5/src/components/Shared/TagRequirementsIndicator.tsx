import React, { useEffect, useRef } from "react";
import { Badge } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { HoverPopover } from "./HoverPopover";
import { Tag, ColorPreset } from "src/core/generated-graphql";

const getContrastColor = (backgroundColor: string): string => {
  if (!backgroundColor) return "#000000";

  let r = 0, g = 0, b = 0;

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

interface ITagRequirementsIndicatorProps {
  tags: Tag[];
  colorPresets: ColorPreset[];
}

export const TagRequirementsIndicator: React.FC<ITagRequirementsIndicatorProps> = ({
  tags,
  colorPresets,
}) => {
  const intl = useIntl();
  const listRef = useRef<HTMLDivElement>(null);

  // Filter presets that have tag requirements description
  const presetsWithRequirements = colorPresets.filter(
    preset => preset.tag_requirements_description && preset.tag_requirements_description.trim() !== ""
  );

  // Separate required and optional presets
  const requiredPresets = presetsWithRequirements.filter(preset => preset.required_for_requirements ?? true);
  const optionalPresets = presetsWithRequirements.filter(preset => !preset.required_for_requirements);

  // Calculate max chip width and apply to all chips
  useEffect(() => {
    if (listRef.current) {
      const chips = listRef.current.querySelectorAll('.tag-preset-name-block span');
      let maxWidth = 0;

      // Find maximum width
      chips.forEach((chip) => {
        const width = (chip as HTMLElement).offsetWidth;
        if (width > maxWidth) {
          maxWidth = width;
        }
      });

      // Apply maximum width to all chips
      if (maxWidth > 0) {
        chips.forEach((chip) => {
          (chip as HTMLElement).style.width = `${maxWidth}px`;
        });
      }
    }
  }, [presetsWithRequirements]); // Re-run when presets change

  // If no presets at all, don't show anything
  if (colorPresets.length === 0) {
    return null;
  }

  // If no presets with requirements, show a different indicator
  if (presetsWithRequirements.length === 0) {
    return (
      <Badge
        variant="secondary"
        className="tag-requirements-indicator ml-2"
        style={{ cursor: "pointer" }}
        title="No tag requirements configured"
      >
        ⚙️
      </Badge>
    );
  }

  // Create a map of colors to presets for quick lookup
  const colorToPreset = new Map<string, ColorPreset>();
  presetsWithRequirements.forEach(preset => {
    colorToPreset.set(preset.color.toLowerCase(), preset);
  });

  // Create a map of tag colors to tags
  const colorToTags = new Map<string, Tag[]>();
  tags.forEach(tag => {
    if (tag.color) {
      const colorKey = tag.color.toLowerCase();
      if (!colorToTags.has(colorKey)) {
        colorToTags.set(colorKey, []);
      }
      colorToTags.get(colorKey)!.push(tag);
    }
  });

  // Calculate filled categories for REQUIRED presets only
  const filledRequiredCategories = requiredPresets.filter(preset =>
    colorToTags.has(preset.color.toLowerCase())
  );

  const totalRequiredCategories = requiredPresets.length;
  const filledRequiredCount = filledRequiredCategories.length;

  // Calculate satisfaction levels
  const minRequired = Math.ceil(totalRequiredCategories * 0.75);
  const isFullySatisfied = totalRequiredCategories === 0 || filledRequiredCount === totalRequiredCategories;
  const isPartiallySatisfied = filledRequiredCount >= minRequired;
  
  // Determine badge variant
  let badgeVariant: "success" | "warning" | "danger" = "danger";
  if (isFullySatisfied) {
    badgeVariant = "success"; // Green - all filled
  } else if (isPartiallySatisfied) {
    badgeVariant = "warning"; // Orange - 75%+ filled
  }

  // Create popover content
  const popoverContent = (
      <div className="tag-requirements-popover">
        <div className="mb-2">
          <strong>
            <FormattedMessage id="tag_requirements.title" />
          </strong>
        </div>
        <div className="tag-requirements-list" ref={listRef}>
          {presetsWithRequirements
            .sort((a, b) => a.sort - b.sort)
            .map(preset => {
              const isFilled = colorToTags.has(preset.color.toLowerCase());
              const isRequired = preset.required_for_requirements ?? true;
              
              // Determine row background class
              let rowClass = "tag-requirement-item";
              if (isRequired) {
                rowClass += isFilled ? " required-filled" : " required-empty";
              } else {
                rowClass += isFilled ? " optional-filled" : " optional-empty";
              }
              
              return (
                <div key={preset.id} className={rowClass}>
                  <span className={`requirement-indicator ${
                    isRequired
                      ? (isFilled ? 'filled' : 'empty')
                      : (isFilled ? 'optional-filled' : 'optional-empty')
                  }`}>
                    {isRequired
                      ? (isFilled ? '✓' : '✗')
                      : '~'
                    }
                  </span>
                  <div className="tag-preset-name-block">
                    <span
                      style={{ backgroundColor: preset.color, color: getContrastColor(preset.color) }}
                      title={intl.formatMessage({ id: "color_preset.preset" }, { preset: preset.name })}
                    >
                      {preset.name}
                    </span>
                  </div>
                  <span className="requirement-description">
                    {preset.tag_requirements_description}
                  </span>
                </div>
              );
            })}
        </div>
        <div className="mt-2 text-muted small">
          <FormattedMessage
            id="tag_requirements.summary"
            values={{
              filled: filledRequiredCount,
              total: totalRequiredCategories,
              min: minRequired,
            }}
          />
          {optionalPresets.length > 0 && (
            <div className="mt-1">
              <FormattedMessage
                id="tag_requirements.optional_summary"
                values={{
                  count: optionalPresets.length,
                }}
              />
            </div>
          )}
        </div>
      </div>
  );

  return (
    <HoverPopover
      content={popoverContent}
      placement="right"
      offset={[10, 0]}
      enterDelay={300}
      leaveDelay={200}
    >
      <Badge
        variant={badgeVariant}
        className="tag-requirements-indicator ml-auto"
        style={{ cursor: "pointer" }}
      >
        {filledRequiredCount}/{totalRequiredCategories} {isPartiallySatisfied ? '✓' : '✗'}
      </Badge>
    </HoverPopover>
  );
};
