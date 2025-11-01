import React, { useEffect, useState } from "react";
import { Form, Col, Row, Button, ButtonGroup } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import isEqual from "lodash-es/isEqual";
import { useBulkSceneUpdate } from "src/core/StashService";
import * as GQL from "src/core/generated-graphql";
import { StudioSelect } from "../Shared/Select";
import { ModalComponent } from "../Shared/Modal";
import { MultiSet } from "../Shared/MultiSet";
import { useToast } from "src/hooks/Toast";
import * as FormUtils from "src/utils/form";
import { RatingSystem } from "../Shared/Rating/RatingSystem";
import { URLListInput } from "../Shared/URLField";
import {
  getAggregateInputIDs,
  getAggregateInputStrings,
  getAggregateInputValue,
  getAggregateGroupIds,
  getAggregatePerformerIds,
  getAggregateRating,
  getAggregateStudioId,
  getAggregateTagIds,
  getAggregateURLs,
} from "src/utils/bulkUpdate";
import { faPencilAlt } from "@fortawesome/free-solid-svg-icons";

interface IListOperationProps {
  selected: GQL.SlimSceneDataFragment[];
  onClose: (applied: boolean) => void;
}

export const EditScenesDialog: React.FC<IListOperationProps> = (
  props: IListOperationProps
) => {
  const intl = useIntl();
  const Toast = useToast();
  const [rating100, setRating] = useState<number | undefined>(undefined);
  const [studioId, setStudioId] = useState<string | undefined>(undefined);

  // Memoize arrays to prevent unnecessary re-renders
  const studioIds = React.useMemo(
    () => (studioId ? [studioId] : []),
    [studioId]
  );
  const [performerMode, setPerformerMode] =
    React.useState<GQL.BulkUpdateIdMode>(GQL.BulkUpdateIdMode.Add);
  const [performerIds, setPerformerIds] = useState<string[]>([]);
  const [existingPerformerIds, setExistingPerformerIds] = useState<string[]>(
    []
  );
  const [tagMode, setTagMode] = React.useState<GQL.BulkUpdateIdMode>(
    GQL.BulkUpdateIdMode.Add
  );
  const [tagIds, setTagIds] = useState<string[]>([]);
  const [existingTagIds, setExistingTagIds] = useState<string[]>([]);
  const [groupMode, setGroupMode] = React.useState<GQL.BulkUpdateIdMode>(
    GQL.BulkUpdateIdMode.Add
  );
  const [groupIds, setGroupIds] = useState<string[]>([]);
  const [existingGroupIds, setExistingGroupIds] = useState<string[]>([]);
  const [urlMode, setUrlMode] = React.useState<GQL.BulkUpdateIdMode>(
    GQL.BulkUpdateIdMode.Add
  );
  const [urls, setUrls] = useState<string[]>([]);
  const [existingUrls, setExistingUrls] = useState<string[]>([]);
  const [organized, setOrganized] = useState<boolean | undefined>();

  const [updateScenes] = useBulkSceneUpdate(getSceneInput());

  // Network state
  const [isUpdating, setIsUpdating] = useState(false);

  const checkboxRef = React.createRef<HTMLInputElement>();

  function getSceneInput(): GQL.BulkSceneUpdateInput {
    // need to determine what we are actually setting on each scene
    const aggregateRating = getAggregateRating(props.selected);
    const aggregateStudioId = getAggregateStudioId(props.selected);
    const aggregatePerformerIds = getAggregatePerformerIds(props.selected);
    const aggregateTagIds = getAggregateTagIds(props.selected);
    const aggregateGroupIds = getAggregateGroupIds(props.selected);
    const aggregateURLs = getAggregateURLs(props.selected);

    const sceneInput: GQL.BulkSceneUpdateInput = {
      ids: props.selected.map((scene) => {
        return scene.id;
      }),
    };

    sceneInput.rating100 = getAggregateInputValue(rating100, aggregateRating);
    sceneInput.studio_id = getAggregateInputValue(studioId, aggregateStudioId);

    sceneInput.performer_ids = getAggregateInputIDs(
      performerMode,
      performerIds,
      aggregatePerformerIds
    );
    sceneInput.tag_ids = getAggregateInputIDs(tagMode, tagIds, aggregateTagIds);
    sceneInput.group_ids = getAggregateInputIDs(
      groupMode,
      groupIds,
      aggregateGroupIds
    );
    sceneInput.urls = getAggregateInputStrings(urlMode, urls, aggregateURLs);

    if (organized !== undefined) {
      sceneInput.organized = organized;
    }

    return sceneInput;
  }

  async function onSave() {
    setIsUpdating(true);
    try {
      await updateScenes();
      Toast.success(
        intl.formatMessage({ id: "toast.scene_with_similars_updated" })
      );
      props.onClose(true);
    } catch (e) {
      Toast.error(e);
    }
    setIsUpdating(false);
  }

  useEffect(() => {
    const state = props.selected;
    let updateRating: number | undefined;
    let updateStudioID: string | undefined;
    let updatePerformerIds: string[] = [];
    let updateTagIds: string[] = [];
    let updateGroupIds: string[] = [];
    let updateUrls: string[] = [];
    let updateOrganized: boolean | undefined;
    let first = true;

    state.forEach((scene: GQL.SlimSceneDataFragment) => {
      const sceneRating = scene.rating100;
      const sceneStudioID = scene?.studio?.id;
      const scenePerformerIDs = (scene.performers ?? [])
        .map((p) => p.id)
        .filter((id) => id != null)
        .sort();
      const sceneTagIDs = (scene.tags ?? [])
        .map((p) => p.id)
        .filter((id) => id != null)
        .sort();
      const sceneGroupIDs = (scene.groups ?? [])
        .map((m) => m.group.id)
        .filter((id) => id != null)
        .sort();
      const sceneURLs = (scene.urls ?? [])
        .filter((url) => url != null && url !== "")
        .sort();

      if (first) {
        updateRating = sceneRating ?? undefined;
        updateStudioID = sceneStudioID;
        updatePerformerIds = scenePerformerIDs;
        updateTagIds = sceneTagIDs;
        updateGroupIds = sceneGroupIDs;
        updateUrls = sceneURLs;
        first = false;
        updateOrganized = scene.organized;
      } else {
        if (sceneRating !== updateRating) {
          updateRating = undefined;
        }
        if (sceneStudioID !== updateStudioID) {
          updateStudioID = undefined;
        }
        if (!isEqual(scenePerformerIDs, updatePerformerIds)) {
          updatePerformerIds = [];
        }
        if (!isEqual(sceneTagIDs, updateTagIds)) {
          updateTagIds = [];
        }
        if (!isEqual(sceneGroupIDs, updateGroupIds)) {
          updateGroupIds = [];
        }
        if (!isEqual(sceneURLs, updateUrls)) {
          updateUrls = [];
        }
        if (scene.organized !== updateOrganized) {
          updateOrganized = undefined;
        }
      }
    });

    setRating(updateRating);
    setStudioId(updateStudioID);
    setExistingPerformerIds(updatePerformerIds);
    setExistingTagIds(updateTagIds);
    setExistingGroupIds(updateGroupIds);
    setExistingUrls(updateUrls);
    setOrganized(updateOrganized);
  }, [props.selected]);

  useEffect(() => {
    if (checkboxRef.current) {
      checkboxRef.current.indeterminate = organized === undefined;
    }
  }, [organized, checkboxRef]);

  function renderMultiSelect(
    type: "performers" | "tags" | "groups",
    ids: string[] | undefined
  ) {
    let mode = GQL.BulkUpdateIdMode.Add;
    let existingIds: string[] | undefined = [];
    switch (type) {
      case "performers":
        mode = performerMode;
        existingIds = existingPerformerIds;
        break;
      case "tags":
        mode = tagMode;
        existingIds = existingTagIds;
        break;
      case "groups":
        mode = groupMode;
        existingIds = existingGroupIds;
        break;
    }

    return (
      <MultiSet
        key={`${type}-select`}
        type={type}
        disabled={isUpdating}
        onUpdate={(itemIDs) => {
          switch (type) {
            case "performers":
              setPerformerIds(itemIDs);
              break;
            case "tags":
              setTagIds(itemIDs);
              break;
            case "groups":
              setGroupIds(itemIDs);
              break;
          }
        }}
        onSetMode={(newMode) => {
          switch (type) {
            case "performers":
              setPerformerMode(newMode);
              break;
            case "tags":
              setTagMode(newMode);
              break;
            case "groups":
              setGroupMode(newMode);
              break;
          }
        }}
        ids={ids}
        existingIds={existingIds}
        mode={mode}
        menuPortalTarget={document.body}
      />
    );
  }

  function renderMultiString(type: "urls", values: string[]) {
    const mode = urlMode;
    const existingValues = existingUrls;

    // Ensure values is always a clean array of strings
    const cleanValues = (values || []).filter(
      (v) => typeof v === "string" && v.trim() !== ""
    );

    function onSetMode(m: GQL.BulkUpdateIdMode) {
      if (m === mode) {
        return;
      }

      // if going to Set, set the existing values
      if (m === GQL.BulkUpdateIdMode.Set && existingValues) {
        setUrls(existingValues.filter((v) => v != null && v !== ""));
        // if going from Set, wipe the values
      } else if (
        m !== GQL.BulkUpdateIdMode.Set &&
        mode === GQL.BulkUpdateIdMode.Set
      ) {
        setUrls([]);
      }

      setUrlMode(m);
    }

    return (
      <div className="multi-set">
        <ButtonGroup className="button-group-above">
          {[
            GQL.BulkUpdateIdMode.Set,
            GQL.BulkUpdateIdMode.Add,
            GQL.BulkUpdateIdMode.Remove,
          ].map((m) => (
            <Button
              key={m}
              variant="primary"
              active={mode === m}
              size="sm"
              onClick={() => onSetMode(m)}
              disabled={isUpdating}
            >
              {m === GQL.BulkUpdateIdMode.Set
                ? intl.formatMessage({ id: "actions.overwrite" })
                : m === GQL.BulkUpdateIdMode.Add
                ? intl.formatMessage({ id: "actions.add" })
                : intl.formatMessage({ id: "actions.remove" })}
            </Button>
          ))}
        </ButtonGroup>
        <URLListInput
          key="urls-select"
          value={cleanValues}
          setValue={setUrls}
          readOnly={isUpdating}
        />
        {existingValues &&
          existingValues.length > 0 &&
          existingValues.every((v) => v != null) && (
            <div className="existing-values">
              <small className="text-muted">
                {intl.formatMessage(
                  { id: "countables.urls" },
                  { count: existingValues.length }
                )}
                : {existingValues.join(", ")}
              </small>
            </div>
          )}
      </div>
    );
  }

  function cycleOrganized() {
    if (organized) {
      setOrganized(undefined);
    } else if (organized === undefined) {
      setOrganized(false);
    } else {
      setOrganized(true);
    }
  }

  function render() {
    return (
      <ModalComponent
        show
        icon={faPencilAlt}
        header={intl.formatMessage(
          { id: "dialogs.edit_entity_title" },
          {
            count: props?.selected?.length ?? 1,
            singularEntity: intl.formatMessage({ id: "scene" }),
            pluralEntity: intl.formatMessage({ id: "scenes" }),
          }
        )}
        accept={{
          onClick: onSave,
          text: intl.formatMessage({ id: "actions.apply" }),
        }}
        cancel={{
          onClick: () => props.onClose(false),
          text: intl.formatMessage({ id: "actions.cancel" }),
          variant: "secondary",
        }}
        isRunning={isUpdating}
      >
        <Form>
          <Form.Group controlId="rating" as={Row}>
            {FormUtils.renderLabel({
              title: intl.formatMessage({ id: "rating" }),
            })}
            <Col xs={9}>
              <RatingSystem
                value={rating100}
                onSetRating={(value) => setRating(value ?? undefined)}
                disabled={isUpdating}
              />
            </Col>
          </Form.Group>
          <Form.Group controlId="studio" as={Row}>
            {FormUtils.renderLabel({
              title: intl.formatMessage({ id: "studio" }),
            })}
            <Col xs={9}>
              <StudioSelect
                key="studio-select"
                onSelect={(items) =>
                  setStudioId(items.length > 0 ? items[0]?.id : undefined)
                }
                ids={studioIds}
                isDisabled={isUpdating}
                menuPortalTarget={document.body}
              />
            </Col>
          </Form.Group>

          <Form.Group controlId="performers">
            <Form.Label>
              <FormattedMessage id="performers" />
            </Form.Label>
            {renderMultiSelect("performers", performerIds)}
          </Form.Group>

          <Form.Group controlId="tags">
            <Form.Label>
              <FormattedMessage id="tags" />
            </Form.Label>
            {renderMultiSelect("tags", tagIds)}
          </Form.Group>

          <Form.Group controlId="groups">
            <Form.Label>
              <FormattedMessage id="groups" />
            </Form.Label>
            {renderMultiSelect("groups", groupIds)}
          </Form.Group>

          <Form.Group controlId="urls">
            <Form.Label>
              <FormattedMessage id="urls" />
            </Form.Label>
            {renderMultiString("urls", urls || [])}
          </Form.Group>

          <Form.Group controlId="organized">
            <Form.Check
              type="checkbox"
              label={intl.formatMessage({ id: "organized" })}
              checked={organized}
              ref={checkboxRef}
              onChange={() => cycleOrganized()}
            />
          </Form.Group>
        </Form>
      </ModalComponent>
    );
  }

  return render();
};
