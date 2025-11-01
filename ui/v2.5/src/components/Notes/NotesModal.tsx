import React, { useState, useEffect } from "react";
import { Modal, Button, Form } from "react-bootstrap";
import { defineMessages, FormattedMessage, useIntl } from "react-intl";
import { Icon } from "src/components/Shared/Icon";
import { faSave, faTimes } from "@fortawesome/free-solid-svg-icons";

const messages = defineMessages({
  title: {
    id: "notes.title",
    defaultMessage: "Notes",
  },
  save: {
    id: "notes.save",
    defaultMessage: "Save",
  },
  cancel: {
    id: "notes.cancel",
    defaultMessage: "Cancel",
  },
  placeholder: {
    id: "notes.placeholder",
    defaultMessage: "Enter your notes...",
  },
});

interface INotesModalProps {
  show: boolean;
  onHide: () => void;
  notes: string;
  onSave: (notes: string) => void;
}

export const NotesModal: React.FC<INotesModalProps> = ({
  show,
  onHide,
  notes,
  onSave,
}) => {
  const intl = useIntl();
  const [localNotes, setLocalNotes] = useState(notes);

  useEffect(() => {
    setLocalNotes(notes);
  }, [notes]);

  const handleSave = () => {
    onSave(localNotes);
    onHide();
  };

  const handleCancel = () => {
    setLocalNotes(notes);
    onHide();
  };

  return (
    <Modal show={show} onHide={handleCancel} size="lg">
      <Modal.Header closeButton>
        <Modal.Title>
          <Icon icon={faSave} className="mr-2" />
          <FormattedMessage {...messages.title} />
        </Modal.Title>
      </Modal.Header>
      <Modal.Body>
        <Form.Group>
          <Form.Control
            as="textarea"
            rows={15}
            value={localNotes}
            onChange={(e) => setLocalNotes(e.target.value)}
            placeholder={intl.formatMessage(messages.placeholder)}
            style={{ resize: "vertical" }}
            className="text-input"
          />
        </Form.Group>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" onClick={handleCancel}>
          <Icon icon={faTimes} className="mr-2" />
          <FormattedMessage {...messages.cancel} />
        </Button>
        <Button variant="primary" onClick={handleSave}>
          <Icon icon={faSave} className="mr-2" />
          <FormattedMessage {...messages.save} />
        </Button>
      </Modal.Footer>
    </Modal>
  );
};
