import React, { useMemo } from "react";
import { useIntl } from "react-intl";
import { Form, Row, Col } from "react-bootstrap";
import { Performer, PerformerSelect } from "src/components/Performers/PerformerSelect";
import cx from "classnames";

export interface IPerformerEntry {
  performer: Performer;
  small_role: boolean;
  role_description?: string | null;
}

export interface IProps {
  value: IPerformerEntry[];
  onUpdate: (input: IPerformerEntry[]) => void;
  onFieldUpdate?: (input: IPerformerEntry[]) => void;
  ageFromDate?: string;
}

export const ScenePerformerTable: React.FC<IProps> = (props) => {
  const { value, onUpdate, onFieldUpdate, ageFromDate } = props;

  const intl = useIntl();

  const performerIDs = useMemo(() => value.map((p) => p.performer.id), [value]);

  const updateFieldChanged = (
    index: number,
    field: "small_role" | "role_description",
    fieldValue: boolean | string | null
  ) => {
    const newValues = value.map((existing, i) => {
      if (i === index) {
        return {
          ...existing,
          [field]: fieldValue,
        };
      }
      return existing;
    });

    // Use onFieldUpdate for field changes to avoid updating tags
    if (onFieldUpdate) {
      onFieldUpdate(newValues);
    } else {
      onUpdate(newValues);
    }
  };

  function onPerformerSet(index: number, performers: Performer[]) {
    if (!performers.length) {
      // remove this entry
      const newValues = value.filter((_, i) => i !== index);
      onUpdate(newValues);
      return;
    }

    const performer = performers[0];

    const newValues = value.map((existing, i) => {
      if (i === index) {
        return {
          ...existing,
          performer: performer,
        };
      }
      return existing;
    });

    onUpdate(newValues);
  }

  function onNewPerformerSet(performers: Performer[]) {
    if (!performers.length) {
      return;
    }

    const performer = performers[0];

    const newValues = [
      ...value,
      {
        performer: performer,
        small_role: performer.small_role || false,
        role_description: null,
      },
    ];

    onUpdate(newValues);
  }

  function renderTableData() {
    return (
      <>
        {value.map((p, i) => (
          <Row key={p.performer.id} className="performer-row mb-2">
            <Col xs={12} md={6} className="px-2">
              <PerformerSelect
                onSelect={(items) => onPerformerSet(i, items)}
                values={[p.performer]}
                ageFromDate={ageFromDate}
                excludeIds={performerIDs}
              />
            </Col>
            <Col xs={6} md={1} className="px-1">
              <Form.Check
                type="checkbox"
                id={`small-role-${p.performer.id}`}
                checked={p.performer.small_role || p.small_role}
                disabled={p.performer.small_role === true}
                onChange={(e) =>
                  updateFieldChanged(i, "small_role", e.target.checked)
                }
              />
            </Col>
            <Col xs={12} md={5} className="px-2">
              <Form.Control
                type="text"
                className="text-input"
                placeholder={intl.formatMessage({
                  id: "role_description_placeholder",
                  defaultMessage: "Role description",
                })}
                value={p.role_description || ""}
                onChange={(e) =>
                  updateFieldChanged(
                    i,
                    "role_description",
                    e.target.value || null
                  )
                }
              />
            </Col>
          </Row>
        ))}
        <Row className="performer-row">
          <Col xs={12} className="px-2">
            <PerformerSelect
              onSelect={(items) => onNewPerformerSet(items)}
              values={[]}
              ageFromDate={ageFromDate}
              excludeIds={performerIDs}
            />
          </Col>
        </Row>
      </>
    );
  }

  return (
    <div className={cx("performer-table", { "no-performers": !value.length })}>
      {value.length > 0 && (
        <Row className="performer-table-header mb-2">
          <Col xs={12} md={6} className="px-2">
            <Form.Label className="mb-0 small">
              {intl.formatMessage({ id: "performer" })}
            </Form.Label>
          </Col>
          <Col xs={6} md={1} className="px-1">
            <Form.Label className="mb-0 small">
              {intl.formatMessage({ id: "small_role" })}
            </Form.Label>
          </Col>
          <Col xs={12} md={5} className="px-2">
            <Form.Label className="mb-0 small">
              {intl.formatMessage({
                id: "role_description",
                defaultMessage: "Role Description",
              })}
            </Form.Label>
          </Col>
        </Row>
      )}
      {renderTableData()}
    </div>
  );
};

