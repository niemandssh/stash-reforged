import { Form, Col, Row } from "react-bootstrap";
import React, { useState } from "react";
import * as GQL from "src/core/generated-graphql";
import { ModalComponent } from "src/components/Shared/Modal";
import * as FormUtils from "src/utils/form";
import {
  queryFindPerformersByIDs,
  usePerformersMerge,
} from "src/core/StashService";
import { useIntl } from "react-intl";
import { useToast } from "src/hooks/Toast";
import { useHistory } from "react-router-dom";
import { faSignInAlt, faSignOutAlt } from "@fortawesome/free-solid-svg-icons";
import { Performer, PerformerSelect } from "../PerformerSelect";
import { PerformerScrapeDialog } from "./PerformerScrapeDialog";
import {
  aggregatePerformersForMerge,
  performerMergeHasConflicts,
  performerToUpdateInput,
} from "./performerMergeUtils";
import { stringToGender } from "src/utils/gender";
import { stringToCircumcised } from "src/utils/circumcised";

function scrapedToUpdateInput(
  dest: GQL.PerformerDataFragment,
  scraped: GQL.ScrapedPerformer
): GQL.PerformerUpdateInput {
  return {
    id: dest.id,
    name: dest.name,
    disambiguation: scraped.disambiguation,
    urls: scraped.urls,
    gender: scraped.gender ? stringToGender(scraped.gender) : undefined,
    birthdate: scraped.birthdate,
    death_date: scraped.death_date,
    ethnicity: scraped.ethnicity,
    country: scraped.country,
    eye_color: scraped.eye_color,
    hair_color: scraped.hair_color,
    height_cm: scraped.height ? parseInt(scraped.height, 10) : undefined,
    weight: scraped.weight ? parseInt(scraped.weight, 10) : undefined,
    measurements: scraped.measurements,
    fake_tits: scraped.fake_tits,
    penis_length: scraped.penis_length
      ? parseFloat(scraped.penis_length)
      : undefined,
    circumcised: scraped.circumcised
      ? stringToCircumcised(scraped.circumcised)
      : undefined,
    career_length: scraped.career_length,
    tattoos: scraped.tattoos,
    piercings: scraped.piercings,
    details: scraped.details,
    image: scraped.images?.[0],
  };
}

interface IPerformerMergeModalProps {
  show: boolean;
  onClose: () => void;
  performer: Pick<GQL.Performer, "id">;
  mergeType: "from" | "into";
}

export const PerformerMergeModal: React.FC<IPerformerMergeModalProps> = ({
  show,
  onClose,
  performer,
  mergeType,
}) => {
  const [src, setSrc] = useState<Performer[]>([]);
  const [dest, setDest] = useState<Performer | null>(null);

  const [loadedSources, setLoadedSources] = useState<
    GQL.PerformerDataFragment[]
  >([]);
  const [loadedDest, setLoadedDest] = useState<GQL.PerformerDataFragment>();

  const [running, setRunning] = useState(false);
  const [secondStep, setSecondStep] = useState(false);

  const [mergePerformers] = usePerformersMerge();

  const intl = useIntl();
  const Toast = useToast();
  const history = useHistory();

  const title = intl.formatMessage({
    id: mergeType === "from" ? "actions.merge_from" : "actions.merge_into",
  });

  function getSourceAndDestinationIds() {
    const source =
      mergeType === "from" ? src.map((s) => s.id) : [performer.id];
    const destination = mergeType === "from" ? performer.id : dest?.id ?? null;
    return { source, destination };
  }

  async function executeMerge(
    source: string[],
    destination: string,
    values?: GQL.PerformerUpdateInput
  ) {
    const result = await mergePerformers({
      variables: {
        input: {
          source,
          destination,
          values,
        },
      },
    });
    if (result.data?.performersMerge) {
      Toast.success(intl.formatMessage({ id: "toast.merged_performers" }));
      onClose();
      history.replace(`/performers/${destination}`);
    }
  }

  async function onMerge() {
    const { source, destination } = getSourceAndDestinationIds();
    if (!destination) return;

    try {
      setRunning(true);
      const allIds = [...source, destination];
      const query = await queryFindPerformersByIDs(allIds);

      const performers = query.data.findPerformers.performers;
      const loadedDestination = performers.find(
        (p: GQL.PerformerDataFragment) => p.id === destination
      );
      const loadedSourcePerformers = performers.filter(
        (p: GQL.PerformerDataFragment) => source.includes(p.id)
      );

      if (!loadedDestination) {
        throw new Error("Destination performer not found");
      }

      if (performerMergeHasConflicts(loadedDestination, loadedSourcePerformers)) {
        setLoadedDest(loadedDestination);
        setLoadedSources(loadedSourcePerformers);
        setSecondStep(true);
      } else {
        await executeMerge(source, destination);
      }
    } catch (e) {
      Toast.error(e);
    } finally {
      setRunning(false);
    }
  }

  function canMerge() {
    return (
      (mergeType === "from" && src.length > 0) ||
      (mergeType === "into" && dest !== null)
    );
  }

  if (secondStep && loadedDest) {
    const destInput = performerToUpdateInput(loadedDest);
    return (
      <PerformerScrapeDialog
        mergeMode
        performer={{
          ...destInput,
          image: loadedDest.image_path ?? undefined,
        }}
        performerTags={loadedDest.tags}
        scraped={aggregatePerformersForMerge(loadedSources)}
        onClose={async (scraped) => {
          if (scraped) {
            const { source, destination } = getSourceAndDestinationIds();
            if (!destination) return;
            try {
              setRunning(true);
              await executeMerge(
                source,
                destination,
                scrapedToUpdateInput(loadedDest, scraped)
              );
            } catch (e) {
              Toast.error(e);
            } finally {
              setRunning(false);
            }
          } else {
            onClose();
          }
        }}
      />
    );
  }

  return (
    <ModalComponent
      show={show}
      header={title}
      icon={mergeType === "from" ? faSignInAlt : faSignOutAlt}
      accept={{
        text: intl.formatMessage({ id: "actions.merge" }),
        onClick: () => onMerge(),
      }}
      disabled={!canMerge()}
      cancel={{
        variant: "secondary",
        onClick: () => onClose(),
      }}
      isRunning={running}
    >
      <div className="form-container row px-3">
        <div className="col-12 col-lg-6 col-xl-12">
          {mergeType === "from" && (
            <Form.Group controlId="source" as={Row}>
              {FormUtils.renderLabel({
                title: intl.formatMessage({
                  id: "dialogs.merge_performers.source",
                }),
                labelProps: {
                  column: true,
                  sm: 3,
                  xl: 12,
                },
              })}
              <Col sm={9} xl={12}>
                <PerformerSelect
                  isMulti
                  creatable={false}
                  onSelect={(items) => setSrc(items)}
                  values={src}
                  excludeIds={performer?.id ? [performer.id] : []}
                  menuPortalTarget={document.body}
                />
              </Col>
            </Form.Group>
          )}
          {mergeType === "into" && (
            <Form.Group controlId="destination" as={Row}>
              {FormUtils.renderLabel({
                title: intl.formatMessage({
                  id: "dialogs.merge_performers.destination",
                }),
                labelProps: {
                  column: true,
                  sm: 3,
                  xl: 12,
                },
              })}
              <Col sm={9} xl={12}>
                <PerformerSelect
                  isMulti={false}
                  creatable={false}
                  onSelect={(items) => setDest(items[0])}
                  values={dest ? [dest] : undefined}
                  excludeIds={performer?.id ? [performer.id] : []}
                  menuPortalTarget={document.body}
                />
              </Col>
            </Form.Group>
          )}
        </div>
      </div>
    </ModalComponent>
  );
};
