import React, { useEffect, useState, useMemo } from "react";
import { Form } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { useFindColorPresets } from "src/core/StashService";
import { ColorPreset } from "src/core/generated-graphql";

interface IColorPresetFilterProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export const ColorPresetFilter: React.FC<IColorPresetFilterProps> = ({
  value,
  onChange,
  disabled = false,
}) => {
  const intl = useIntl();
  const { data: presetsData, loading } = useFindColorPresets();
  const [options, setOptions] = useState<
    Array<{ value: string; label: string }>
  >([]);

  const presets = useMemo(
    () => presetsData?.findColorPresets?.color_presets || [],
    [presetsData]
  );

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

  if (loading) {
    return (
      <Form.Control as="select" disabled value="">
        <option value="">
          <FormattedMessage id="loading.generic" />
        </option>
      </Form.Control>
    );
  }

  return (
    <Form.Control
      as="select"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
    >
      {options.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </Form.Control>
  );
};
