import React from "react";
import { FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import TextUtils from "src/utils/text";
import { URLsField } from "src/utils/field";
import { TagLink } from "src/components/Shared/TagLink";

interface IProps {
  game: GQL.GameDataFragment;
}

export const GameDetailPanel: React.FC<IProps> = ({ game }) => {
  const intl = useIntl();

  const createdAt = game.created_at
    ? TextUtils.formatDateTime(intl, game.created_at)
    : undefined;
  const updatedAt = game.updated_at
    ? TextUtils.formatDateTime(intl, game.updated_at)
    : undefined;

  return (
    <div className="gallery-details">
      {game.details && (
        <p className="text-muted pre" style={{ whiteSpace: "pre-wrap" }}>
          {game.details}
        </p>
      )}
      {game.urls && game.urls.length > 0 && (
        <div className="mb-3">
          <URLsField id="game-urls" urls={game.urls} />
        </div>
      )}
      {game.date && (
        <div className="mb-2">
          <strong className="mr-1">
            <FormattedMessage id="date" />:
          </strong>
          {TextUtils.formatDate(intl, game.date)}
        </div>
      )}
      {createdAt && (
        <div className="mb-2">
          <strong className="mr-1">
            <FormattedMessage id="created_at" />:
          </strong>
          {createdAt}
        </div>
      )}
      {updatedAt && (
        <div className="mb-2">
          <strong className="mr-1">
            <FormattedMessage id="updated_at" />:
          </strong>
          {updatedAt}
        </div>
      )}
      {game.tags && game.tags.length > 0 && (
        <div className="mt-3">
          <h6>
            <FormattedMessage
              id="countables.tags"
              values={{ count: game.tags.length }}
            />
          </h6>
          <div className="tag-list">
            {game.tags.map((tag) => (
              <TagLink key={tag.id} tag={tag} linkType="details" />
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
