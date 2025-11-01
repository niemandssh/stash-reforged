import { ModifierCriterionOption, ModifierCriterion } from "./criterion";
import { CriterionModifier } from "src/core/generated-graphql";
import { IntlShape } from "react-intl";

export class ColorPresetCriterionOption extends ModifierCriterionOption {
  constructor() {
        super({
          messageID: "color_preset.title",
          type: "color_preset" as const,
      modifierOptions: [
        CriterionModifier.Equals,
        CriterionModifier.NotEquals,
        CriterionModifier.IsNull,
        CriterionModifier.NotNull,
      ],
      defaultModifier: CriterionModifier.Equals,
      inputType: "text",
      makeCriterion: () => new ColorPresetCriterion(this),
    });
  }
}

export class ColorPresetCriterion extends ModifierCriterion<string> {
  constructor(type: ModifierCriterionOption) {
    super(type, "");
  }

  protected getLabelValue() {
    return this.value;
  }

  public isValid(): boolean {
    return (
      this.modifier === CriterionModifier.IsNull ||
      this.modifier === CriterionModifier.NotNull ||
      this.value.length > 0
    );
  }
}
