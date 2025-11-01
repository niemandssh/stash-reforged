import React, { useState, useRef, useEffect } from "react";
import {
  Button,
  Col,
  Form,
  OverlayTrigger,
  Popover,
  Row,
} from "react-bootstrap";
import { createPortal } from "react-dom";
import { useIntl } from "react-intl";
import { ModalComponent } from "./Modal";
import { Icon } from "src/components/Shared/Icon";
import { faFile, faLink } from "@fortawesome/free-solid-svg-icons";
import { PatchComponent } from "src/patch";

interface IImageInput {
  isEditing: boolean;
  text?: string;
  onImageChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onImageURL?: (url: string) => void;
  acceptSVG?: boolean;
}

function acceptExtensions(acceptSVG: boolean = false) {
  return `.jpg,.jpeg,.png,.webp,.gif${acceptSVG ? ",.svg" : ""}`;
}

export const ImageInput: React.FC<IImageInput> = PatchComponent(
  "ImageInput",
  ({ isEditing, text, onImageChange, onImageURL, acceptSVG = false }) => {
    const [isShowDialog, setIsShowDialog] = useState(false);
    const [showPopover, setShowPopover] = useState(false);
    const [url, setURL] = useState("");
    const urlInputRef = useRef<HTMLInputElement>(null);
    const intl = useIntl();

    // Auto-focus URL input when dialog opens
    useEffect(() => {
      if (isShowDialog && urlInputRef.current) {
        setTimeout(() => {
          urlInputRef.current?.focus();
        }, 100);
      }
    }, [isShowDialog]);

    if (!isEditing) return <div />;

    if (!onImageURL) {
      return (
        <Form.Label className="image-input">
          <Button variant="secondary">
            {text ?? intl.formatMessage({ id: "actions.browse_for_image" })}
          </Button>
          <Form.Control
            type="file"
            onChange={onImageChange}
            accept={acceptExtensions(acceptSVG)}
          />
        </Form.Label>
      );
    }

    function showDialog() {
      setURL("");
      setIsShowDialog(true);
    }

    function handleFileSelect() {
      setShowPopover(false);
    }

    function onConfirmURL() {
      if (!onImageURL) {
        return;
      }

      setIsShowDialog(false);
      setShowPopover(false);
      onImageURL(url);
    }

    function handleKeyPress(event: React.KeyboardEvent<HTMLInputElement>) {
      if (event.key === "Enter") {
        onConfirmURL();
      }
    }

    function renderDialog() {
      const modalContent = (
        <ModalComponent
          show={!!isShowDialog}
          onHide={() => setIsShowDialog(false)}
          header={intl.formatMessage({ id: "dialogs.set_image_url_title" })}
          accept={{
            onClick: onConfirmURL,
            text: intl.formatMessage({ id: "actions.confirm" }),
          }}
          modalProps={{
            style: { zIndex: 9999 },
            backdrop: true,
            keyboard: true,
          }}
          dialogClassName="image-url-modal"
        >
          <div className="dialog-content">
            <Form.Group controlId="url" as={Row}>
              <Form.Label column xs={3}>
                {intl.formatMessage({ id: "url" })}
              </Form.Label>
              <Col xs={9}>
                <Form.Control
                  ref={urlInputRef}
                  className="text-input"
                  onChange={(event: React.ChangeEvent<HTMLInputElement>) =>
                    setURL(event.currentTarget.value)
                  }
                  onKeyPress={handleKeyPress}
                  value={url}
                  placeholder={intl.formatMessage({ id: "url" })}
                  autoFocus
                />
              </Col>
            </Form.Group>
          </div>
        </ModalComponent>
      );

      // Portal the modal to document.body to ensure it's on top
      return isShowDialog ? createPortal(modalContent, document.body) : null;
    }

    const popover = (
      <Popover id="set-image-popover">
        <Popover.Content>
          <>
            <div>
              <Form.Label className="image-input">
                <Button variant="secondary">
                  <Icon icon={faFile} className="fa-fw" />
                  <span>{intl.formatMessage({ id: "actions.from_file" })}</span>
                </Button>
                <Form.Control
                  type="file"
                  onChange={(e) => {
                    handleFileSelect();
                    onImageChange(e as React.ChangeEvent<HTMLInputElement>);
                  }}
                  accept={acceptExtensions(acceptSVG)}
                />
              </Form.Label>
            </div>
            <div>
              <Button
                className="minimal"
                onClick={() => {
                  setShowPopover(false);
                  showDialog();
                }}
              >
                <Icon icon={faLink} className="fa-fw" />
                <span>{intl.formatMessage({ id: "actions.from_url" })}</span>
              </Button>
            </div>
          </>
        </Popover.Content>
      </Popover>
    );

    return (
      <>
        {renderDialog()}
        <OverlayTrigger
          trigger="click"
          placement="top"
          overlay={popover}
          show={showPopover}
          rootClose
        >
          <Button
            variant="secondary"
            onClick={() => setShowPopover(!showPopover)}
          >
            {text ?? intl.formatMessage({ id: "actions.set_image" })}
          </Button>
        </OverlayTrigger>
      </>
    );
  }
);

// Add CSS styles for the modal
const modalStyles = `
  .image-url-modal .modal-dialog {
    z-index: 10000 !important;
  }
  .image-url-modal .modal-backdrop {
    z-index: 9999 !important;
  }
  .image-url-modal .modal {
    z-index: 10000 !important;
  }
`;

// Inject styles into the document head
if (typeof document !== "undefined") {
  const styleElement = document.createElement("style");
  styleElement.textContent = modalStyles;
  document.head.appendChild(styleElement);
}
