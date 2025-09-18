import React from "react";
import { FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import TextUtils from "src/utils/text";
import { TagLink } from "src/components/Shared/TagLink";
import { PerformerCard } from "src/components/Performers/PerformerCard";
import { sortPerformers } from "src/core/performers";
import { PhotographerLink } from "src/components/Shared/Link";
import { GalleryPoseTagsDisplay } from "./GalleryPoseTagsDisplay";
import { URLsField } from "src/utils/field";

interface IGalleryDetailProps {
  gallery: GQL.GalleryDataFragment;
}

export const GalleryDetailPanel: React.FC<IGalleryDetailProps> = ({
  gallery,
}) => {
  const intl = useIntl();

  function renderDetails() {
    if (!gallery.details) return;
    return (
      <>
        <h6>
          <FormattedMessage id="details" />:{" "}
        </h6>
        <p className="pre">{gallery.details}</p>
      </>
    );
  }

  function renderTags() {
    const regularTags = gallery.tags.filter(tag => !tag.is_pose_tag);
    if (regularTags.length === 0) return;
    const tags = regularTags.map((tag) => (
      <TagLink key={tag.id} tag={tag} linkType="details" />
    ));
    return (
      <>
        <div className="mt-3 mb-3">
          <h4>
            <FormattedMessage
              id="countables.tags"
              values={{ count: regularTags.length }}
            />
          </h4>
        </div>
        {tags}
      </>
    );
  }

  function renderPerformers() {
    if (gallery.performers.length === 0) return;
    const performers = sortPerformers(gallery.performers);
    const cards = performers.map((performer) => (
      <PerformerCard
        key={performer.id}
        performer={performer}
        ageFromDate={gallery.date ?? undefined}
      />
    ));

    return (
      <>
        <h6>
          <FormattedMessage
            id="countables.performers"
            values={{ count: gallery.performers.length }}
          />
        </h6>
        <div className="row justify-content-center gallery-performers">
          {cards}
        </div>
      </>
    );
  }


  // filename should use entire row if there is no studio
  const galleryDetailsWidth = gallery.studio ? "col-9" : "col-12";

  return (
    <>
      <div className="row">
        <div className={`${galleryDetailsWidth} col-12 gallery-details`}>
          <URLsField id="urls" urls={gallery.urls} />
          <h6>
            <FormattedMessage id="created_at" />:{" "}
            {TextUtils.formatDateTime(intl, gallery.created_at)}{" "}
          </h6>
          <h6>
            <FormattedMessage id="updated_at" />:{" "}
            {TextUtils.formatDateTime(intl, gallery.updated_at)}{" "}
          </h6>
          {gallery.code && (
            <h6>
              <FormattedMessage id="scene_code" />: {gallery.code}{" "}
            </h6>
          )}
          {gallery.photographer && (
            <h6>
              <FormattedMessage id="photographer" />:{" "}
              <PhotographerLink
                photographer={gallery.photographer}
                linkType="gallery"
              />
            </h6>
          )}
        </div>
      </div>
      <div className="row">
        <div className="col-12">
          {renderDetails()}
          <GalleryPoseTagsDisplay gallery={gallery} />
          {renderTags()}
          {renderPerformers()}
        </div>
      </div>
    </>
  );
};
