import React from "react";
import { Link } from "react-router-dom";
import { useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import NavUtils from "src/utils/navigation";
import TextUtils from "src/utils/text";
import { GridCard } from "../Shared/GridCard/GridCard";
import { CountryFlag } from "../Shared/CountryFlag";
import { SweatDrops } from "../Shared/SweatDrops";
import { HoverPopover } from "../Shared/HoverPopover";
import { Icon } from "../Shared/Icon";
import { TagLink } from "../Shared/TagLink";
import { Button, ButtonGroup } from "react-bootstrap";
import {
  ModifierCriterion,
  CriterionValue,
} from "src/models/list-filter/criteria/criterion";
import { PopoverCountButton } from "../Shared/PopoverCountButton";
import GenderIcon from "./GenderIcon";
import { faTag } from "@fortawesome/free-solid-svg-icons";
import { RatingBanner } from "../Shared/RatingBanner";
import { usePerformerUpdate } from "src/core/StashService";
import { ILabeledId } from "src/models/list-filter/types";
import { FavoriteIcon } from "../Shared/FavoriteIcon";
import { DeathRibbon } from "../Shared/DeathRibbon";
import { PatchComponent } from "src/patch";

export interface IPerformerCardExtraCriteria {
  scenes?: ModifierCriterion<CriterionValue>[];
  images?: ModifierCriterion<CriterionValue>[];
  galleries?: ModifierCriterion<CriterionValue>[];
  groups?: ModifierCriterion<CriterionValue>[];
  performer?: ILabeledId;
}

interface IPerformerCardProps {
  performer: GQL.PerformerDataFragment;
  cardWidth?: number;
  ageFromDate?: string;
  selecting?: boolean;
  selected?: boolean;
  zoomIndex?: number;
  onSelectedChanged?: (selected: boolean, shiftKey: boolean) => void;
  extraCriteria?: IPerformerCardExtraCriteria;
  roleDescription?: string;
}

const PerformerCardPopovers: React.FC<IPerformerCardProps> = PatchComponent(
  "PerformerCard.Popovers",
  ({ performer, extraCriteria }) => {
    function maybeRenderScenesPopoverButton() {
      if (!performer.scene_count) return;

      return (
        <PopoverCountButton
          className="scene-count"
          type="scene"
          count={performer.scene_count}
          url={NavUtils.makePerformerScenesUrl(
            performer,
            extraCriteria?.performer,
            extraCriteria?.scenes
          )}
        />
      );
    }

    function maybeRenderImagesPopoverButton() {
      if (!performer.image_count) return;

      return (
        <PopoverCountButton
          className="image-count"
          type="image"
          count={performer.image_count}
          url={NavUtils.makePerformerImagesUrl(
            performer,
            extraCriteria?.performer,
            extraCriteria?.images
          )}
        />
      );
    }

    function maybeRenderGalleriesPopoverButton() {
      if (!performer.gallery_count) return;

      return (
        <PopoverCountButton
          className="gallery-count"
          type="gallery"
          count={performer.gallery_count}
          url={NavUtils.makePerformerGalleriesUrl(
            performer,
            extraCriteria?.performer,
            extraCriteria?.galleries
          )}
        />
      );
    }

    function maybeRenderOCounter() {
      if (!performer.o_counter) return;

      return (
        <div className="o-counter">
          <Button className="minimal">
            <span className="fa-icon">
              <SweatDrops />
            </span>
            <span>{performer.o_counter}</span>
          </Button>
        </div>
      );
    }

    function maybeRenderTagPopoverButton() {
      if (performer.tags.length <= 0) return;

      const popoverContent = performer.tags.map((tag) => (
        <TagLink key={tag.id} linkType="performer" tag={tag} />
      ));

      return (
        <HoverPopover placement="bottom" content={popoverContent}>
          <Button className="minimal tag-count">
            <Icon icon={faTag} />
            <span>{performer.tags.length}</span>
          </Button>
        </HoverPopover>
      );
    }

    function maybeRenderGroupsPopoverButton() {
      if (!performer.group_count) return;

      return (
        <PopoverCountButton
          className="group-count"
          type="group"
          count={performer.group_count}
          url={NavUtils.makePerformerGroupsUrl(
            performer,
            extraCriteria?.performer,
            extraCriteria?.groups
          )}
        />
      );
    }

    if (
      performer.scene_count ||
      performer.image_count ||
      performer.gallery_count ||
      performer.tags.length > 0 ||
      performer.o_counter ||
      performer.group_count
    ) {
      return (
        <>
          <hr />
          <ButtonGroup className="card-popovers">
            {maybeRenderScenesPopoverButton()}
            {maybeRenderGroupsPopoverButton()}
            {maybeRenderImagesPopoverButton()}
            {maybeRenderGalleriesPopoverButton()}
            {maybeRenderTagPopoverButton()}
            {maybeRenderOCounter()}
          </ButtonGroup>
        </>
      );
    }

    return null;
  }
);

const PerformerCardOverlays: React.FC<IPerformerCardProps> = PatchComponent(
  "PerformerCard.Overlays",
  ({ performer }) => {
    const [updatePerformer] = usePerformerUpdate();
    const intl = useIntl();

    function onToggleFavorite(v: boolean) {
      if (performer.id) {
        updatePerformer({
          variables: {
            input: {
              id: performer.id,
              favorite: v,
            },
          },
        });
      }
    }

    function maybeRenderRatingBanner() {
      if (!performer.rating100) {
        return;
      }
      return <RatingBanner rating={performer.rating100} />;
    }

    function maybeRenderFlag() {
      if (performer.country) {
        return (
          <Link to={NavUtils.makePerformersCountryUrl(performer)}>
            <CountryFlag
              className="performer-card__country-flag"
              country={performer.country}
              includeOverlay
            />
            <span className="performer-card__country-string">
              {performer.country}
            </span>
          </Link>
        );
      }
    }

    function maybeRenderCurrentAgeChip() {
      // Current age (or age at death) - shown as white chip at bottom
      const currentAge = TextUtils.age(
        performer.birthdate,
        performer.death_date
      );

      if (currentAge <= 0) {
        return null;
      }

      // Always show chip for deceased performers
      // For living performers, always show current age chip when age > 0
      const isDead = !!performer.death_date;
      let chipText = "";

      if (isDead) {
        // Show "Dead at X yo" for deceased performers
        const deadAtString = intl.formatMessage({
          id: "dead_at",
          defaultMessage: "Dead at",
        });
        const ageShortString = intl.formatMessage({
          id: "years_old_short",
          defaultMessage: "yo",
        });
        chipText = `${deadAtString} ${currentAge} ${ageShortString}`;
      } else {
        // Show normal current age for living performers
        const ageL10String = intl.formatMessage({
          id: "years_old",
          defaultMessage: "years old",
        });

        chipText = intl.formatMessage(
          { id: "media_info.performer_card.age" },
          { age: currentAge, years_old: ageL10String }
        );
      }

      return <div className="performer-card__current-age-chip">{chipText}</div>;
    }

    return (
      <>
        <FavoriteIcon
          favorite={performer.favorite}
          onToggleFavorite={onToggleFavorite}
          size="2x"
          className="hide-not-favorite"
        />
        {maybeRenderRatingBanner()}
        {maybeRenderFlag()}
        {maybeRenderCurrentAgeChip()}
        {performer.death_date && <DeathRibbon size="large" />}
      </>
    );
  }
);

const PerformerCardDetails: React.FC<IPerformerCardProps> = PatchComponent(
  "PerformerCard.Details",
  ({ performer, ageFromDate }) => {
    const intl = useIntl();

    // Only show age in details if we have ageFromDate (age at production)
    let ageString = "";
    if (ageFromDate) {
      const age = TextUtils.age(performer.birthdate, ageFromDate);

      if (age > 0) {
        const ageL10nId = "media_info.performer_card.age_context";
        const ageL10String = intl.formatMessage({
          id: "years_old",
          defaultMessage: "years old",
        });
        ageString = intl.formatMessage(
          { id: ageL10nId },
          { age, years_old: ageL10String }
        );
      }
    }

    return (
      <>
        {ageString ? (
          <div className="performer-card__age">{ageString}</div>
        ) : (
          ""
        )}
      </>
    );
  }
);

const PerformerCardImage: React.FC<IPerformerCardProps> = PatchComponent(
  "PerformerCard.Image",
  ({ performer }) => {
    return (
      <>
        <img
          loading="lazy"
          className={`performer-card-image ${
            performer.death_date ? "deceased" : ""
          }`}
          alt={performer.name ?? ""}
          src={performer.primary_image_path ?? performer.image_path ?? ""}
        />
      </>
    );
  }
);

const PerformerCardTitle: React.FC<IPerformerCardProps> = PatchComponent(
  "PerformerCard.Title",
  ({ performer, roleDescription }) => {
    return (
      <div className="performer-title-container">
        <span className="performer-name">{performer.name}</span>
        {performer.primary_tag && (
          <TagLink
            tag={performer.primary_tag}
            linkType="performer"
            className="performer-primary-tag"
          />
        )}
        {performer.disambiguation && (
          <span className="performer-disambiguation">
            {` (${performer.disambiguation})`}
          </span>
        )}
        {roleDescription && (
          <div className="performer-role-description">
            <strong>Role:</strong> {roleDescription}
          </div>
        )}
      </div>
    );
  }
);

export const PerformerCard: React.FC<IPerformerCardProps> = PatchComponent(
  "PerformerCard",
  (props) => {
    const {
      performer,
      cardWidth,
      selecting,
      selected,
      onSelectedChanged,
      zoomIndex,
    } = props;

    return (
      <GridCard
        className={`performer-card zoom-${zoomIndex}`}
        url={`/performers/${performer.id}`}
        width={cardWidth}
        pretitleIcon={
          <GenderIcon className="gender-icon" gender={performer.gender} />
        }
        title={<PerformerCardTitle {...props} />}
        image={<PerformerCardImage {...props} />}
        overlays={<PerformerCardOverlays {...props} />}
        details={<PerformerCardDetails {...props} />}
        popovers={<PerformerCardPopovers {...props} />}
        selected={selected}
        selecting={selecting}
        onSelectedChanged={onSelectedChanged}
      />
    );
  }
);
