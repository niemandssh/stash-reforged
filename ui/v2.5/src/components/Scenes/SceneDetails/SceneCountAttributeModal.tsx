import React, { useState, useCallback, useEffect } from "react";
import { Button, Form } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { ModalComponent } from "src/components/Shared/Modal";
import { CountryFlag } from "src/components/Shared/CountryFlag";
import { SweatDrops } from "src/components/Shared/SweatDrops";
import { OMGIcon } from "src/components/Shared/OMGIcon";
import { PerformerPopover } from "src/components/Performers/PerformerPopover";
import { DateInput } from "src/components/Shared/DateInput";
import TextUtils from "src/utils/text";
import { Icon } from "src/components/Shared/Icon";
import { faCheck } from "@fortawesome/free-solid-svg-icons";

type CountType = "o" | "omg";

interface IProps {
  show: boolean;
  onHide: () => void;
  type: CountType;
  performers: GQL.PerformerDataFragment[];
  /** When true, show date/time picker at top and pass date to onConfirm */
  includeDatePicker?: boolean;
  onConfirm: (performerIds: string[] | null, date?: string) => void;
}

export const SceneCountAttributeModal: React.FC<IProps> = ({
  show,
  onHide,
  type,
  performers,
  includeDatePicker = false,
  onConfirm,
}) => {
  const intl = useIntl();
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [selectedDate, setSelectedDate] = useState<string>(() =>
    TextUtils.dateTimeToString(new Date())
  );

  useEffect(() => {
    if (show) {
      setSelectedDate(TextUtils.dateTimeToString(new Date()));
    }
  }, [show]);

  const countLabel =
    type === "o"
      ? intl.formatMessage({ id: "o_count" })
      : intl.formatMessage({ id: "omg_count", defaultMessage: "OMG count" });

  const countIcon =
    type === "o" ? (
      <span className="mr-2 align-middle d-inline-flex">
        <SweatDrops />
      </span>
    ) : (
      <span className="mr-2 align-middle d-inline-flex">
        <OMGIcon />
      </span>
    );

  const handlePerformerClick = useCallback(
    (performerId: string) => {
      onConfirm([performerId], includeDatePicker ? selectedDate : undefined);
      onHide();
    },
    [onConfirm, onHide, includeDatePicker, selectedDate]
  );

  const handleCheckboxChange = useCallback((performerId: string, checked: boolean) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (checked) next.add(performerId);
      else next.delete(performerId);
      return next;
    });
  }, []);

  const handleConfirmSelected = useCallback(() => {
    if (selectedIds.size === 0) return;
    onConfirm(
      Array.from(selectedIds),
      includeDatePicker ? selectedDate : undefined
    );
    setSelectedIds(new Set());
    onHide();
  }, [selectedIds, onConfirm, onHide, includeDatePicker, selectedDate]);

  const handleWholeScene = useCallback(() => {
    onConfirm(null, includeDatePicker ? selectedDate : undefined);
    onHide();
  }, [onConfirm, onHide, includeDatePicker, selectedDate]);

  const ageStr = (p: GQL.PerformerDataFragment) => {
    const age = TextUtils.age(p.birthdate, p.death_date ?? undefined);
    if (!p.birthdate && age === 0) return null;
    return intl.formatMessage(
      {
        id: "media_info.performer_card.age",
        defaultMessage: "{age} years old",
      },
      { age }
    );
  };

  return (
    <ModalComponent
      show={show}
      onHide={onHide}
      header={
        <>
          {countIcon}
          {intl.formatMessage(
            {
              id: "dialogs.attribute_count_to_performer",
              defaultMessage: "Attribute {count} to",
            },
            { count: countLabel }
          )}
        </>
      }
      hideFooter
      dialogClassName="scene-count-attribute-modal-dialog"
    >
      <div className="scene-count-attribute-modal">
        {includeDatePicker && (
          <div className="mb-3">
            <Form.Group>
              <Form.Label>
                <FormattedMessage id="actions.choose_date" />
              </Form.Label>
              <DateInput
                value={selectedDate}
                onValueChange={setSelectedDate}
                isTime
              />
            </Form.Group>
          </div>
        )}
        <p className="text-muted mb-3">
          <FormattedMessage
            id="dialogs.attribute_count_choose"
            defaultMessage="Choose who to attribute {count} to, or confirm selection below."
            values={{ count: countLabel }}
          />
        </p>
        <div className="performer-list mb-3">
          {performers.map((p) => (
            <div
              key={p.id}
              className="performer-row d-flex align-items-center mb-2 px-0"
            >
              <Form.Check
                type="checkbox"
                checked={selectedIds.has(p.id)}
                onChange={(e) =>
                  handleCheckboxChange(p.id, e.currentTarget.checked)
                }
                onClick={(e) => e.stopPropagation()}
                className="mr-3 flex-shrink-0"
              />
              <Button
                variant="secondary"
                className="d-flex align-items-center flex-grow-1 text-left performer-attribution-btn"
                onClick={() => handlePerformerClick(p.id)}
              >
                <PerformerPopover id={p.id} placement="right">
                  <img
                    src={p.primary_image_path ?? p.image_path ?? ""}
                    alt=""
                    className="performer-thumb"
                  />
                </PerformerPopover>
                <div className="performer-info ml-2">
                  <div className="performer-name">{p.name}</div>
                  <div className="performer-meta text-muted small">
                    {p.country && (
                      <span className="mr-3">
                        <CountryFlag
                          country={p.country}
                          className="ml-2"
                          includeName
                        />
                      </span>
                    )}
                    {ageStr(p) && <span className="mr-3">{ageStr(p)}</span>}
                    {(p.o_counter ?? 0) > 0 && (
                      <span className="count-badge mr-3">
                        <SweatDrops />
                        {p.o_counter}
                      </span>
                    )}
                    {(p.omg_counter ?? 0) > 0 && (
                      <span className="count-badge">
                        <OMGIcon />
                        {p.omg_counter}
                      </span>
                    )}
                  </div>
                </div>
              </Button>
            </div>
          ))}
        </div>
        {selectedIds.size > 0 && (
          <Button
            variant="success"
            className="mb-2 w-100"
            onClick={handleConfirmSelected}
          >
            <Icon icon={faCheck} className="mr-3" />
            {intl.formatMessage(
              {
                id: "dialogs.confirm_to_performers",
                defaultMessage: "Confirm to {count} performers",
              },
              { count: selectedIds.size }
            )}
          </Button>
        )}
        <Button
          variant="secondary"
          block
          onClick={handleWholeScene}
        >
          {countIcon}
          <FormattedMessage
            id="dialogs.attribute_count_whole_scene"
            defaultMessage="Attribute {count} to whole scene"
            values={{ count: countLabel }}
          />
        </Button>
      </div>
    </ModalComponent>
  );
};
