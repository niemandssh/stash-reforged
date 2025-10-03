import React from "react";
import { Badge } from "react-bootstrap";
import { FormattedMessage } from "react-intl";
import { HoverPopover } from "./HoverPopover";
import { Tag, ColorPreset } from "src/core/generated-graphql";

interface ITagRequirementsIndicatorProps {
  tags: Tag[];
  colorPresets: ColorPreset[];
}

export const TagRequirementsIndicator: React.FC<ITagRequirementsIndicatorProps> = ({
  tags,
  colorPresets,
}) => {
  // Filter presets that have tag requirements description
  const presetsWithRequirements = colorPresets.filter(
    preset => preset.tag_requirements_description && preset.tag_requirements_description.trim() !== ""
  );

  // Separate required and optional presets
  const requiredPresets = presetsWithRequirements.filter(preset => preset.required_for_requirements ?? true);
  const optionalPresets = presetsWithRequirements.filter(preset => !preset.required_for_requirements);

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
        <div className="tag-requirements-list">
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
      placement="bottom"
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
