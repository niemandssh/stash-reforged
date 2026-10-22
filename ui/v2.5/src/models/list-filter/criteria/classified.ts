import { BooleanCriterion, BooleanCriterionOption } from "./criterion";

export const ClassifiedCriterionOption = new BooleanCriterionOption(
  "classified",
  "classified",
  () => new ClassifiedCriterion()
);

export class ClassifiedCriterion extends BooleanCriterion {
  constructor() {
    super(ClassifiedCriterionOption);
  }
}
