import React from "react";
import { Form } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { SettingSection } from "./SettingSection";
import { BooleanSetting, ModalSetting, SelectSetting, StringSetting } from "./Inputs";
import { useSettings } from "./context";

const VISION_PROVIDERS = [
  "groq",
  "deepinfra",
  "together",
  "openai",
  "custom",
] as const;

type VisionProvider = (typeof VISION_PROVIDERS)[number];

const VISION_PROVIDER_DEFAULTS: Record<
  Exclude<VisionProvider, "custom">,
  { url: string; model: string }
> = {
  groq: {
    url: "https://api.groq.com/openai/v1",
    model: "meta-llama/llama-4-scout-17b-16e-instruct",
  },
  deepinfra: {
    url: "https://api.deepinfra.com/v1/openai",
    model: "google/gemini-2.5-flash",
  },
  together: {
    url: "https://api.together.xyz/v1",
    model: "meta-llama/Llama-4-Scout-17B-16E-Instruct",
  },
  openai: {
    url: "https://api.openai.com/v1",
    model: "gpt-4o-mini",
  },
};

function maskSecretValue(
  value: string,
  startVisible = 7,
  endVisible = 4
): string {
  if (value.length <= startVisible + endVisible) {
    if (value.length <= 2) {
      return "*".repeat(value.length);
    }
    return (
      value.slice(0, 1) +
      "*".repeat(value.length - 2) +
      value.slice(-1)
    );
  }

  const hiddenLength = value.length - startVisible - endVisible;
  return (
    value.slice(0, startVisible) +
    "*".repeat(hiddenLength) +
    value.slice(-endVisible)
  );
}

function isVisionProvider(value: string | null | undefined): value is VisionProvider {
  return VISION_PROVIDERS.includes(value as VisionProvider);
}

export const SettingsAIPanel: React.FC = () => {
  const intl = useIntl();
  const { ai, loading, error, saveAI } = useSettings();

  if (loading || error) {
    return null;
  }

  const visionEnabled = ai.visionEnabled ?? false;
  const visionProvider: VisionProvider = isVisionProvider(ai.visionProvider)
    ? ai.visionProvider
    : "groq";
  const isCustomVision = visionProvider === "custom";
  const presetDefaults = isCustomVision
    ? null
    : VISION_PROVIDER_DEFAULTS[visionProvider];

  return (
    <SettingSection headingID="config.ai.heading">
      <BooleanSetting
        id="ai-scene-fill-enabled"
        headingID="config.ai.scene_fill_enabled"
        checked={ai.sceneFillEnabled ?? false}
        onChange={(v) => saveAI({ sceneFillEnabled: v })}
      />
      <ModalSetting<string>
        id="deepseek-api-key"
        headingID="config.ai.deepseek_api_key"
        subHeadingID="config.ai.deepseek_api_key_desc"
        value={ai.deepseekApiKey ?? undefined}
        onChange={(v) => saveAI({ deepseekApiKey: v })}
        renderField={(value, setValue) => (
          <Form.Control
            className="text-input"
            type="password"
            autoComplete="off"
            value={value ?? ""}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setValue(e.currentTarget.value)
            }
          />
        )}
        renderValue={(value) => (
          <span>{value ? maskSecretValue(value) : ""}</span>
        )}
      />
      <StringSetting
        id="deepseek-model"
        headingID="config.ai.deepseek_model"
        subHeadingID="config.ai.deepseek_model_desc"
        value={ai.deepseekModel ?? ""}
        onChange={(v) => saveAI({ deepseekModel: v })}
      />
      <BooleanSetting
        id="ai-vision-enabled"
        headingID="config.ai.vision_enabled"
        subHeadingID="config.ai.vision_enabled_desc"
        checked={visionEnabled}
        onChange={(v) => saveAI({ visionEnabled: v })}
      />
      {visionEnabled && (
        <>
          <SelectSetting
            id="vision-provider"
            headingID="config.ai.vision_provider"
            subHeadingID="config.ai.vision_provider_desc"
            value={visionProvider}
            onChange={(v) => {
              if (!isVisionProvider(v)) {
                return;
              }

              if (v === "custom") {
                saveAI({ visionProvider: v });
                return;
              }

              const defaults = VISION_PROVIDER_DEFAULTS[v];
              saveAI({
                visionProvider: v,
                visionApiUrl: defaults.url,
                visionModel: defaults.model,
              });
            }}
          >
            {VISION_PROVIDERS.map((provider) => (
              <option key={provider} value={provider}>
                {intl.formatMessage({
                  id: `config.ai.vision_provider_${provider}`,
                })}
              </option>
            ))}
          </SelectSetting>
          {!isCustomVision && presetDefaults && (
            <div className="setting">
              <div className="value">
                <div className="text-muted small">
                  <div>
                    <FormattedMessage id="config.ai.vision_api_url" />:{" "}
                    <code>{presetDefaults.url}</code>
                  </div>
                  <div>
                    <FormattedMessage id="config.ai.vision_model" />:{" "}
                    <code>{presetDefaults.model}</code>
                  </div>
                </div>
              </div>
            </div>
          )}
          <ModalSetting<string>
            id="vision-api-key"
            headingID="config.ai.vision_api_key"
            subHeadingID="config.ai.vision_api_key_desc"
            value={ai.visionApiKey ?? undefined}
            onChange={(v) => saveAI({ visionApiKey: v })}
            renderField={(value, setValue) => (
              <Form.Control
                className="text-input"
                type="password"
                autoComplete="off"
                value={value ?? ""}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                  setValue(e.currentTarget.value)
                }
              />
            )}
            renderValue={(value) => (
              <span>{value ? maskSecretValue(value) : ""}</span>
            )}
          />
          {isCustomVision && (
            <>
              <StringSetting
                id="vision-api-url"
                headingID="config.ai.vision_api_url"
                subHeadingID="config.ai.vision_api_url_desc"
                value={ai.visionApiUrl ?? ""}
                onChange={(v) => saveAI({ visionApiUrl: v })}
              />
              <StringSetting
                id="vision-model"
                headingID="config.ai.vision_model"
                subHeadingID="config.ai.vision_model_desc"
                value={ai.visionModel ?? ""}
                onChange={(v) => saveAI({ visionModel: v })}
              />
            </>
          )}
        </>
      )}
    </SettingSection>
  );
};
