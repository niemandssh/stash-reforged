import React from "react";
import cx from "classnames";
import { Button, Spinner } from "react-bootstrap";
import { Icon } from "src/components/Shared/Icon";
import { useIntl } from "react-intl";
import { faCircleCheck } from "@fortawesome/free-solid-svg-icons";

export interface IClassifiedButtonProps {
  loading: boolean;
  classified: boolean;
  onClick: () => void;
}

export const ClassifiedButton: React.FC<IClassifiedButtonProps> = (
  props: IClassifiedButtonProps
) => {
  const intl = useIntl();

  if (props.loading) return <Spinner animation="border" role="status" />;

  return (
    <Button
      variant="secondary"
      title={intl.formatMessage({ id: "classified" })}
      className={cx(
        "minimal",
        "classified-button",
        props.classified ? "classified" : "not-classified"
      )}
      onClick={props.onClick}
    >
      <Icon icon={faCircleCheck} />
    </Button>
  );
};
