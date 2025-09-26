import React from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import TextUtils from "src/utils/text";
import { TagLink } from "src/components/Shared/TagLink";
import { PerformerCard } from "src/components/Performers/PerformerCard";
import { sortPerformers } from "src/core/performers";
import { DirectorLink } from "src/components/Shared/Link";
import { SimilarScenes } from "./SimilarScenes";
import { URLsField } from "src/utils/field";
import { PoseTagsDisplay } from "./PoseTagsDisplay";
import GenderIcon from "src/components/Performers/GenderIcon";

interface ISceneDetailProps {
  scene: GQL.SceneDataFragment;
}

export const SceneDetailPanel: React.FC<ISceneDetailProps> = (props) => {
  const intl = useIntl();

  function renderDetails() {
    if (!props.scene.details || props.scene.details === "") return;
    return (
      <>
        <h6>
          <FormattedMessage id="details" />:{" "}
        </h6>
        <p className="pre">{props.scene.details}</p>
      </>
    );
  }

  function renderTags() {
    const regularTags = props.scene.tags.filter(tag => !tag.is_pose_tag);
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
    if (props.scene.performers.length === 0) return;

    // Separate performers into two groups
    const mainPerformers = props.scene.performers.filter(p => !p.small_role);
    const smallRolePerformers = props.scene.performers.filter(p => p.small_role);

    // Sort main performers
    const sortedMainPerformers = sortPerformers(mainPerformers);

    // Create cards for main performers
    const mainCards = sortedMainPerformers.map((performer) => (
      <PerformerCard
        key={performer.id}
        performer={performer}
        ageFromDate={props.scene.date ?? undefined}
      />
    ));

    // Create list for performers with small role
    const smallRoleList = smallRolePerformers.length > 0 ? (
      <div className="mt-3">
        <h6 className="scene-performers-small-role-header">
          <FormattedMessage id="scene_performers.small_role" defaultMessage="Also starring:" />
        </h6>
        <div className="scene-performers-small-role">
          {smallRolePerformers.map((performer) => {
            const currentAge = TextUtils.age(
              performer.birthdate,
              performer.death_date
            );
            const productionAge = TextUtils.age(
              performer.birthdate,
              props.scene.date ?? undefined
            );
            const intl = useIntl();
            const ageShortString = intl.formatMessage({
              id: "years_old_short",
              defaultMessage: "yo",
            });
            const atProductionString = intl.formatMessage({
              id: "at_production",
              defaultMessage: "at production",
            });

            const currentAgeString = currentAge > 0 ? `${currentAge} ${ageShortString}` : "";
            const productionAgeString = productionAge > 0 ? `${productionAge} ${ageShortString} ${atProductionString}` : "";

            return (
              <Link
                key={performer.id}
                to={`/performers/${performer.id}`}
                className="scene-performer-small-role-tag"
              >
                <div className="performer-info">
                  <div className="performer-header">
                    <GenderIcon gender={performer.gender} className="gender-icon-small" />
                    <span className="performer-name">{performer.name}</span>
                    {currentAgeString && (
                      <span className="performer-age-small">({currentAgeString})</span>
                    )}
                  </div>
                  {productionAgeString && (
                    <div className="performer-age-at-production">{productionAgeString}</div>
                  )}
                </div>
              </Link>
            );
          })}
        </div>
      </div>
    ) : null;

    return (
      <>
        <div className="mt-3 mb-3">
          <h4>
            <FormattedMessage
              id="countables.performers"
              values={{ count: props.scene.performers.length }}
            />
          </h4>
        </div>
        <div className="row justify-content-center scene-performers">
          {mainCards}
        </div>
        {smallRoleList}
      </>
    );
  }

  // filename should use entire row if there is no studio
  const sceneDetailsWidth = props.scene.studio ? "col-9" : "col-12";

  return (
    <>
      <div className="row">
        <div className={`${sceneDetailsWidth} col-12 scene-details`}>
          <URLsField id="urls" urls={props.scene.urls} truncate />
          <h6>
            <FormattedMessage id="created_at" />:{" "}
            {TextUtils.formatDateTime(intl, props.scene.created_at)}{" "}
          </h6>
          <h6>
            <FormattedMessage id="updated_at" />:{" "}
            {TextUtils.formatDateTime(intl, props.scene.updated_at)}{" "}
          </h6>
          {props.scene.code && (
            <h6>
              <FormattedMessage id="scene_code" />: {props.scene.code}{" "}
            </h6>
          )}
          {props.scene.director && (
            <h6>
              <FormattedMessage id="director" />:{" "}
              <DirectorLink director={props.scene.director} linkType="scene" />
            </h6>
          )}
        </div>
      </div>
      <div className="row">
        <div className="col-12">
          {renderDetails()}
          <PoseTagsDisplay scene={props.scene} />
          {renderTags()}
          {renderPerformers()}
        </div>
      </div>
      <div className="row">
        <div className="col-12">
          <SimilarScenes scene={props.scene} />
        </div>
      </div>
    </>
  );
};

export default SceneDetailPanel;
