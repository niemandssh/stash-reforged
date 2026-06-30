import { BooleanCriterion, BooleanCriterionOption } from "./criterion";

export const HasTrimTimesCriterionOption = new BooleanCriterionOption(
  "has_trim_times",
  "has_trim_times"
);

export class HasTrimTimesCriterion extends BooleanCriterion {}
