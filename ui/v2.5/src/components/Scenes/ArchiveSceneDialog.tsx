import React, { useState } from "react";
import { Form } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { faBoxArchive } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { useSceneArchive } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";
import { ModalComponent } from "../Shared/Modal";
import { objectPath } from "src/core/files";

interface IArchiveSceneDialogProps {
  scene: GQL.SceneDataFragment;
  onClose: (archived: boolean) => void;
}

function funscriptPath(sp: string) {
  const extIndex = sp.lastIndexOf(".");
  if (extIndex !== -1) {
    return sp.substring(0, extIndex + 1) + "funscript";
  }

  return sp;
}

export const ArchiveSceneDialog: React.FC<IArchiveSceneDialogProps> = (
  props: IArchiveSceneDialogProps
) => {
  const intl = useIntl();
  const Toast = useToast();
  const [archiveScene] = useSceneArchive();
  const [deleteFile, setDeleteFile] = useState(true);
  const [archiveReason, setArchiveReason] = useState("");
  const [isArchiving, setIsArchiving] = useState(false);

  async function onArchive() {
    setIsArchiving(true);
    try {
      const trimmedReason = archiveReason.trim();
      await archiveScene({
        variables: {
          id: props.scene.id,
          delete_file: deleteFile,
          archive_reason: trimmedReason || null,
        },
      });
      Toast.success(
        intl.formatMessage({ id: "toast.scene_archived" })
      );
      props.onClose(true);
    } catch (e) {
      Toast.error(e);
      props.onClose(false);
    }
    setIsArchiving(false);
  }

  function maybeRenderDeleteFileAlert() {
    if (!deleteFile) {
      return;
    }

    const deletedFiles: string[] = props.scene.files.map((f) => f.path);
    if (props.scene.interactive && props.scene.files.length) {
      deletedFiles.push(funscriptPath(objectPath(props.scene)));
    }

    if (deletedFiles.length === 0) {
      return;
    }

    return (
      <div className="delete-dialog alert alert-danger text-break">
        <p className="font-weight-bold">
          <FormattedMessage
            values={{
              count: deletedFiles.length,
              singularEntity: intl.formatMessage({ id: "file" }),
              pluralEntity: intl.formatMessage({ id: "files" }),
            }}
            id="dialogs.delete_alert"
          />
        </p>
        <ul>
          {deletedFiles.slice(0, 5).map((s) => (
            <li key={s}>{s}</li>
          ))}
          {deletedFiles.length > 5 && (
            <FormattedMessage
              values={{
                count: deletedFiles.length - 5,
                singularEntity: intl.formatMessage({ id: "file" }),
                pluralEntity: intl.formatMessage({ id: "files" }),
              }}
              id="dialogs.delete_object_overflow"
            />
          )}
        </ul>
      </div>
    );
  }

  return (
    <ModalComponent
      show
      icon={faBoxArchive}
      header={intl.formatMessage({ id: "dialogs.archive_scene.title" })}
      accept={{
        variant: "primary",
        onClick: onArchive,
        text: intl.formatMessage({ id: "actions.archive_scene" }),
      }}
      cancel={{
        onClick: () => props.onClose(false),
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
      }}
      isRunning={isArchiving}
    >
      <p>{intl.formatMessage({ id: "dialogs.archive_scene.description" })}</p>
      {maybeRenderDeleteFileAlert()}
      <Form>
        <Form.Group className="mb-3">
          <Form.Label>
            {intl.formatMessage({ id: "dialogs.archive_scene.reason" })}
          </Form.Label>
          <Form.Control
            as="textarea"
            rows={2}
            className="text-input"
            value={archiveReason}
            onChange={(e) => setArchiveReason(e.target.value)}
            placeholder={intl.formatMessage({
              id: "dialogs.archive_scene.reason_placeholder",
            })}
          />
        </Form.Group>
        <Form.Check
          id="archive-delete-file"
          checked={deleteFile}
          label={intl.formatMessage({
            id: "actions.delete_file_and_funscript",
          })}
          onChange={() => setDeleteFile(!deleteFile)}
        />
      </Form>
    </ModalComponent>
  );
};

export default ArchiveSceneDialog;
