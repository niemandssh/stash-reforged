import { Form, Col, Row, FormControl } from "react-bootstrap";
import React, { useEffect, useMemo, useState } from "react";
import * as GQL from "src/core/generated-graphql";
import { ModalComponent } from "src/components/Shared/Modal";
import * as FormUtils from "src/utils/form";
import { queryFindStudiosByIDs, useStudiosMerge } from "src/core/StashService";
import { useIntl, FormattedMessage } from "react-intl";
import { useToast } from "src/hooks/Toast";
import { useHistory } from "react-router-dom";
import { faSignInAlt, faSignOutAlt } from "@fortawesome/free-solid-svg-icons";
import { Studio, StudioSelect } from "../StudioSelect";
import {
  aggregateStudiosForMerge,
  studioMergeHasConflicts,
} from "./studioMergeUtils";
import {
  ScrapeDialog,
  ScrapedInputGroupRow,
  ScrapedTextAreaRow,
  ScrapeDialogRow,
  ScrapedImageRow,
} from "src/components/Shared/ScrapeDialog/ScrapeDialog";
import {
  mergeFavoriteScrapeResult,
  mergeFieldScrapeResult,
  mergeImagePathScrapeResult,
  mergeRatingScrapeResult,
  hasRealImagePath,
} from "src/components/Shared/mergeConflictUtils";
import {
  ObjectScrapeResult,
  ScrapeResult,
  hasScrapedValues,
} from "src/components/Shared/ScrapeDialog/scrapeResult";
import { ScrapedStudioRow } from "src/components/Shared/ScrapeDialog/ScrapedObjectsRow";
import { RatingSystem } from "src/components/Shared/Rating/RatingSystem";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import ImageUtils from "src/utils/image";

interface IStudioMergeDetailsProps {
  sources: GQL.StudioDataFragment[];
  dest: GQL.StudioDataFragment;
  onClose: (values?: GQL.StudioUpdateInput) => void;
}

const StudioMergeDetails: React.FC<IStudioMergeDetailsProps> = ({
  sources,
  dest,
  onClose,
}) => {
  const intl = useIntl();
  const [loading, setLoading] = useState(true);
  const scraped = aggregateStudiosForMerge(sources);

  const [url, setURL] = useState(mergeFieldScrapeResult(dest.url, scraped.url));
  const [details, setDetails] = useState(
    mergeFieldScrapeResult(dest.details, scraped.details)
  );
  const [rating, setRating] = useState(
    mergeRatingScrapeResult(
      dest.rating100,
      sources.find((s) => s.rating100)?.rating100
    )
  );
  const [favorite, setFavorite] = useState(
    mergeFavoriteScrapeResult(dest.favorite, sources.every((s) => s.favorite))
  );
  const [parent, setParent] = useState<ObjectScrapeResult<GQL.ScrapedStudio>>(
    new ObjectScrapeResult<GQL.ScrapedStudio>(
      dest.parent_studio
        ? { stored_id: dest.parent_studio.id, name: dest.parent_studio.name }
        : undefined,
      scraped.parent,
      !dest.parent_studio && !!scraped.parent
    )
  );
  const [image, setImage] = useState(
    mergeImagePathScrapeResult(dest.image_path, scraped.image)
  );

  useEffect(() => {
    async function loadImages() {
      const srcPath = sources.find((s) => hasRealImagePath(s.image_path))
        ?.image_path;
      const destReal = hasRealImagePath(dest.image_path);

      if (!srcPath && !destReal) {
        setLoading(false);
        return;
      }

      setLoading(true);
      const destData =
        destReal && dest.image_path
          ? await ImageUtils.imageToDataURL(dest.image_path)
          : undefined;
      const srcData = srcPath
        ? await ImageUtils.imageToDataURL(srcPath)
        : undefined;

      const isConflict = destReal && !!srcPath && dest.image_path !== srcPath;
      const useNew = !destReal && !!srcData;

      setImage(new ScrapeResult(destData, srcData, isConflict ? false : useNew));
      setLoading(false);
    }

    loadImages();
  }, [sources, dest]);

  const hasValues = useMemo(
    () => hasScrapedValues([url, details, rating, favorite, parent, image]),
    [url, details, rating, favorite, parent, image]
  );

  function createValues(): GQL.StudioUpdateInput {
    const coverImage = image.useNewValue ? image.getNewValue() : undefined;
    return {
      id: dest.id,
      name: dest.name,
      url: url.getNewValue(),
      details: details.getNewValue(),
      rating100: rating.getNewValue(),
      favorite: favorite.getNewValue(),
      parent_id: parent.getNewValue()?.stored_id,
      image: coverImage,
    };
  }

  function renderScrapeRows() {
    if (loading) {
      return <LoadingIndicator />;
    }

    if (!hasValues) {
      return (
        <div>
          <FormattedMessage id="dialogs.merge.empty_results" />
        </div>
      );
    }

    const trueString = intl.formatMessage({ id: "true" });
    const falseString = intl.formatMessage({ id: "false" });

    return (
      <>
        <ScrapedInputGroupRow
          title={intl.formatMessage({ id: "url" })}
          result={url}
          onChange={(value) => setURL(value)}
        />
        <ScrapedTextAreaRow
          title={intl.formatMessage({ id: "details" })}
          result={details}
          onChange={(value) => setDetails(value)}
        />
        <ScrapeDialogRow
          title={intl.formatMessage({ id: "rating" })}
          result={rating}
          renderOriginalField={() => (
            <RatingSystem value={rating.originalValue} disabled />
          )}
          renderNewField={() => (
            <RatingSystem value={rating.newValue} disabled />
          )}
          onChange={(value) => setRating(value)}
        />
        <ScrapeDialogRow
          title={intl.formatMessage({ id: "favourite" })}
          result={favorite}
          renderOriginalField={() => (
            <FormControl
              value={favorite.originalValue ? trueString : falseString}
              readOnly
              onChange={() => {}}
              className="bg-secondary text-white border-secondary"
            />
          )}
          renderNewField={() => (
            <FormControl
              value={favorite.newValue ? trueString : falseString}
              readOnly
              onChange={() => {}}
              className="bg-secondary text-white border-secondary"
            />
          )}
          onChange={(value) => setFavorite(value)}
        />
        <ScrapedStudioRow
          title={intl.formatMessage({ id: "parent_studio" })}
          result={parent}
          onChange={(value) => setParent(value)}
        />
        <ScrapedImageRow
          title={intl.formatMessage({ id: "image" })}
          result={image}
          onChange={(value) => setImage(value)}
        />
      </>
    );
  }

  return (
    <ScrapeDialog
      title={intl.formatMessage({ id: "actions.merge" })}
      existingLabel={intl.formatMessage({ id: "dialogs.merge.destination" })}
      scrapedLabel={intl.formatMessage({ id: "dialogs.merge.source" })}
      renderScrapeRows={renderScrapeRows}
      onClose={(apply) => {
        if (!apply) {
          onClose();
        } else {
          onClose(createValues());
        }
      }}
    />
  );
};

interface IStudioMergeModalProps {
  show: boolean;
  onClose: () => void;
  studio: Pick<GQL.Studio, "id">;
  mergeType: "from" | "into";
}

export const StudioMergeModal: React.FC<IStudioMergeModalProps> = ({
  show,
  onClose,
  studio,
  mergeType,
}) => {
  const [src, setSrc] = useState<Studio[]>([]);
  const [dest, setDest] = useState<Studio | null>(null);

  const [loadedSources, setLoadedSources] = useState<GQL.StudioDataFragment[]>(
    []
  );
  const [loadedDest, setLoadedDest] = useState<GQL.StudioDataFragment>();

  const [running, setRunning] = useState(false);
  const [secondStep, setSecondStep] = useState(false);

  const [mergeStudios] = useStudiosMerge();

  const intl = useIntl();
  const Toast = useToast();
  const history = useHistory();

  const title = intl.formatMessage({
    id: mergeType === "from" ? "actions.merge_from" : "actions.merge_into",
  });

  function getSourceAndDestinationIds() {
    const source =
      mergeType === "from" ? src.map((s) => s.id) : [studio.id];
    const destination = mergeType === "from" ? studio.id : dest?.id ?? null;
    return { source, destination };
  }

  async function executeMerge(
    source: string[],
    destination: string,
    values?: GQL.StudioUpdateInput
  ) {
    const result = await mergeStudios({
      variables: {
        input: {
          source,
          destination,
          values,
        },
      },
    });
    if (result.data?.studiosMerge) {
      Toast.success(intl.formatMessage({ id: "toast.merged_studios" }));
      onClose();
      history.replace(`/studios/${destination}`);
    }
  }

  async function onMerge() {
    const { source, destination } = getSourceAndDestinationIds();
    if (!destination) return;

    try {
      setRunning(true);
      const allIds = [...source, destination];
      const query = await queryFindStudiosByIDs(allIds);

      const studios = query.data.findStudios.studios;
      const loadedDestination = studios.find(
        (s: GQL.StudioDataFragment) => s.id === destination
      );
      const loadedSourceStudios = studios.filter((s: GQL.StudioDataFragment) =>
        source.includes(s.id)
      );

      if (!loadedDestination) {
        throw new Error("Destination studio not found");
      }

      if (studioMergeHasConflicts(loadedDestination, loadedSourceStudios)) {
        setLoadedDest(loadedDestination);
        setLoadedSources(loadedSourceStudios);
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
    return (
      <StudioMergeDetails
        sources={loadedSources}
        dest={loadedDest}
        onClose={async (values) => {
          const { source, destination } = getSourceAndDestinationIds();
          if (!destination) return;
          if (values) {
            try {
              setRunning(true);
              await executeMerge(source, destination, values);
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
                  id: "dialogs.merge_studios.source",
                }),
                labelProps: {
                  column: true,
                  sm: 3,
                  xl: 12,
                },
              })}
              <Col sm={9} xl={12}>
                <StudioSelect
                  isMulti
                  creatable={false}
                  onSelect={(items) => setSrc(items)}
                  values={src}
                  excludeIds={studio?.id ? [studio.id] : []}
                  menuPortalTarget={document.body}
                />
              </Col>
            </Form.Group>
          )}
          {mergeType === "into" && (
            <Form.Group controlId="destination" as={Row}>
              {FormUtils.renderLabel({
                title: intl.formatMessage({
                  id: "dialogs.merge_studios.destination",
                }),
                labelProps: {
                  column: true,
                  sm: 3,
                  xl: 12,
                },
              })}
              <Col sm={9} xl={12}>
                <StudioSelect
                  isMulti={false}
                  creatable={false}
                  onSelect={(items) => setDest(items[0])}
                  values={dest ? [dest] : undefined}
                  excludeIds={studio?.id ? [studio.id] : []}
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
