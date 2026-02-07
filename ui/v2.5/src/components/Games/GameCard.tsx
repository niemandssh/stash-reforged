import React from "react";
import { Button, ButtonGroup, OverlayTrigger, Tooltip } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import { GridCard } from "../Shared/GridCard/GridCard";
import TextUtils from "src/utils/text";
import { FormattedMessage, useIntl } from "react-intl";
import { TruncatedText } from "../Shared/TruncatedText";
import { Icon } from "../Shared/Icon";
import { SweatDrops } from "../Shared/SweatDrops";
import { OMGIcon } from "../Shared/OMGIcon";
import { faBox, faEye } from "@fortawesome/free-solid-svg-icons";
import cx from "classnames";
import { RatingBanner } from "../Shared/RatingBanner";
import { TagLink } from "../Shared/TagLink";

interface IProps {
  game: GQL.SlimGameDataFragment;
  cardWidth?: number;
  zoomIndex?: number;
  selecting?: boolean;
  selected?: boolean;
  onSelectedChanged?: (selected: boolean, shiftKey: boolean) => void;
}

const GameCardImage: React.FC<{ game: GQL.SlimGameDataFragment }> = ({
  game,
}) => {
  return (
    <>
      <div className="gallery-card-cover">
        {game.image_path ? (
          <img
            loading="lazy"
            className="gallery-card-image"
            alt={game.title ?? ""}
            src={game.image_path}
          />
        ) : (
          <div className="gallery-card-placeholder">
            <FormattedMessage id="cover_image" />
          </div>
        )}
      </div>
      <RatingBanner rating={game.rating100 ?? undefined} />
    </>
  );
};

const GameCardDetails: React.FC<{
  game: GQL.SlimGameDataFragment;
  intl: ReturnType<typeof useIntl>;
}> = ({ game, intl }) => {
  const date = game.date
    ? TextUtils.formatDate(intl, game.date)
    : TextUtils.formatDate(
        intl,
        game.updated_at ?? game.created_at ?? undefined
      );

  return (
    <div className="gallery-card__details">
      {date && <span className="gallery-card__date">{date}</span>}
      {game.details && (
        <TruncatedText
          className="gallery-card__description"
          text={game.details}
          lineCount={3}
        />
      )}
      {game.tags && game.tags?.length > 0 && (
        <div className="gallery-card__tags tag-list">
          {game.tags?.slice(0, 3).map((tag) => (
            <TagLink key={tag.id} tag={tag} linkType="details" />
          ))}
        </div>
      )}
    </div>
  );
};

const GameCardStats: React.FC<{
  game: GQL.SlimGameDataFragment;
  intl: ReturnType<typeof useIntl>;
}> = ({ game, intl }) => {
  const hasStats =
    game.play_count > 0 || game.o_counter > 0 || game.omgCounter > 0;

  if (!hasStats && !game.organized) {
    return null;
  }

  return (
    <>
      <hr />
      <ButtonGroup className="card-popovers">
        {game.play_count > 0 && (
          <div className="image-count">
            <Button
              className="minimal"
              title={intl.formatMessage({ id: "play_count" })}
            >
              <Icon icon={faEye} />
              <span>{game.play_count}</span>
            </Button>
          </div>
        )}
        {game.o_counter > 0 && (
          <div className="o-count">
            <Button
              className="minimal"
              title={intl.formatMessage({ id: "o_counter" })}
            >
              <span className="fa-icon">
                <SweatDrops />
              </span>
              <span>{game.o_counter}</span>
            </Button>
          </div>
        )}
        {game.omgCounter > 0 && (
          <div className="omg-count">
            <Button
              className="minimal"
              title={intl.formatMessage({ id: "omg_counter" })}
            >
              <span className="fa-icon">
                <OMGIcon />
              </span>
              <span>{game.omgCounter}</span>
            </Button>
          </div>
        )}
        {game.organized && (
          <OverlayTrigger
            placement="bottom"
            overlay={
              <Tooltip id={`game-organized-${game.id}`}>
                {intl.formatMessage({ id: "organized" })}
              </Tooltip>
            }
          >
            <div className="organized">
              <Button className="minimal organized-indicator">
                <Icon icon={faBox} />
              </Button>
            </div>
          </OverlayTrigger>
        )}
      </ButtonGroup>
    </>
  );
};

export const GameCard: React.FC<IProps> = ({
  game,
  cardWidth,
  zoomIndex,
  selecting,
  selected,
  onSelectedChanged,
}) => {
  const intl = useIntl();
  const zoomClass = zoomIndex !== undefined ? `zoom-${zoomIndex}` : undefined;
  const className = cx("gallery-card", "game-card", zoomClass);

  return (
    <GridCard
      className={className}
      url={`/games/${game.id}`}
      width={cardWidth}
      title={game.title ?? ""}
      linkClassName="gallery-card-header"
      image={<GameCardImage game={game} />}
      details={<GameCardDetails game={game} intl={intl} />}
      popovers={<GameCardStats game={game} intl={intl} />}
      selecting={selecting}
      selected={selected}
      onSelectedChanged={onSelectedChanged}
    />
  );
};
