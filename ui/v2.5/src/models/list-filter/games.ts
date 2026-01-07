import {
  createDateCriterionOption,
  createMandatoryNumberCriterionOption,
  createMandatoryTimestampCriterionOption,
  createStringCriterionOption,
} from "./criteria/criterion";
import { OrganizedCriterionOption } from "./criteria/organized";
import { ListFilterOptions } from "./filter-options";
import { DisplayMode } from "./types";
import { RatingCriterionOption } from "./criteria/rating";
import { TagsCriterionOption } from "./criteria/tags";

const defaultSortBy = "title";

const sortByOptions = [
  "title",
  "date",
  "rating100",
  "o_counter",
  "omg_counter",
  "play_count",
].map(ListFilterOptions.createSortBy);

const displayModeOptions = [DisplayMode.Grid, DisplayMode.List];

const criterionOptions = [
  createStringCriterionOption("title"),
  createStringCriterionOption("details"),
  createStringCriterionOption("url"),
  createDateCriterionOption("date"),
  RatingCriterionOption,
  OrganizedCriterionOption,
  createMandatoryNumberCriterionOption("o_counter"),
  createMandatoryNumberCriterionOption("omg_counter"),
  createMandatoryNumberCriterionOption("play_count"),
  TagsCriterionOption,
  createMandatoryTimestampCriterionOption("created_at"),
  createMandatoryTimestampCriterionOption("updated_at"),
];

export const GameListFilterOptions = new ListFilterOptions(
  defaultSortBy,
  sortByOptions,
  displayModeOptions,
  criterionOptions
);
