import React, { useEffect, useState, useMemo } from "react";
import { Form } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { ModifierCriterion } from "src/models/list-filter/criteria/criterion";
import { CriterionModifier, ColorPreset } from "src/core/generated-graphql";
import { useFindColorPresets } from "src/core/StashService";

interface IColorPresetFilter {
  criterion: ModifierCriterion<string>;
  setCriterion: (c: ModifierCriterion<string>) => void;
}

export const ColorPresetFilter: React.FC<IColorPresetFilter> = ({
  criterion,
  setCriterion,
}) => {
  const intl = useIntl();
  const { data: presetsData, loading } = useFindColorPresets();
  const [options, setOptions] = useState<Array<{ value: string; label: string }>>([]);

  const presets = useMemo(() => presetsData?.findColorPresets?.color_presets || [], [presetsData]);

  useEffect(() => {
    const newOptions = [
      { value: "", label: intl.formatMessage({ id: "all" }) },
      { value: "null", label: intl.formatMessage({ id: "no_color" }) },
      ...presets.map((preset: ColorPreset) => ({
        value: preset.color,
        label: `${preset.name} (${preset.color})`,
      })),
    ];
    setOptions(newOptions);
  }, [presets, intl]);

  function onValueChange(event: React.ChangeEvent<HTMLSelectElement>) {
    const c = criterion.clone();
    c.value = event.target.value;
    setCriterion(c);
  }

  const isDisabled = 
    criterion.modifier === CriterionModifier.IsNull || 
    criterion.modifier === CriterionModifier.NotNull;

  return (
    <Form.Control
      as="select"
      value={criterion.value}
      onChange={onValueChange}
      disabled={isDisabled || loading}
      className="btn-secondary"
    >
      {loading ? (
        <option value="">
          <FormattedMessage id="loading.generic" />
        </option>
      ) : (
        options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))
      )}
    </Form.Control>
  );
};
