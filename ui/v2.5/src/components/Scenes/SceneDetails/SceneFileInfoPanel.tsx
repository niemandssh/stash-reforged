import React, { useMemo, useState } from "react";
import { Accordion, Button, Card } from "react-bootstrap";
import {
  FormattedMessage,
  FormattedNumber,
  FormattedTime,
  useIntl,
} from "react-intl";
import { useHistory } from "react-router-dom";
import { TruncatedText } from "src/components/Shared/TruncatedText";
import { DeleteFilesDialog } from "src/components/Shared/DeleteFilesDialog";
import { ReassignFilesDialog } from "src/components/Shared/ReassignFilesDialog";
import * as GQL from "src/core/generated-graphql";
import {
  mutateSceneSetPrimaryFile,
  useOpenInExternalPlayer,
  useScanVideoFileThreats,
} from "src/core/StashService";
import { useToast } from "src/hooks/Toast";
import NavUtils from "src/utils/navigation";
import TextUtils from "src/utils/text";
import { TextField, URLField } from "src/utils/field";
import { StashIDPill } from "src/components/Shared/StashID";
import { PatchComponent } from "../../../patch";
import { FileSize } from "src/components/Shared/FileSize";

interface IFileInfoPanelProps {
  sceneID: string;
  file: GQL.VideoFileDataFragment;
  primary?: boolean;
  ofMany?: boolean;
  onSetPrimaryFile?: () => void;
  onDeleteFile?: () => void;
  onReassign?: () => void;
  onScanThreats?: (fileId: string) => void;
  loading?: boolean;
  scanningThreats?: boolean;
}

const FileInfoPanel: React.FC<IFileInfoPanelProps> = ({
  sceneID,
  file,
  primary,
  ofMany,
  onSetPrimaryFile,
  onDeleteFile,
  onReassign,
  onScanThreats,
  loading,
  scanningThreats,
}) => {
  const intl = useIntl();
  const history = useHistory();
  const Toast = useToast();
  const [openInExternalPlayer] = useOpenInExternalPlayer();

  // TODO - generalise fingerprints
  const oshash = file.fingerprints.find((f) => f.type === "oshash");
  const phash = file.fingerprints.find((f) => f.type === "phash");
  const checksum = file.fingerprints.find((f) => f.type === "md5");

  function onSplit() {
    history.push(`/scenes/new?from_scene_id=${sceneID}&file_id=${file.id}`);
  }

  async function onOpenExternalPlayer() {
    try {
      await openInExternalPlayer({ variables: { id: sceneID } });
      Toast.success("Opened in external player");
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <div>
      <dl className="container scene-file-info details-list">
        {primary && (
          <>
            <dt></dt>
            <dd className="primary-file">
              <FormattedMessage id="primary_file" />
            </dd>
          </>
        )}
        <TextField id="media_info.hash" value={oshash?.value} truncate />
        <TextField id="media_info.checksum" value={checksum?.value} truncate />
        <URLField
          id="media_info.phash"
          abbr="Perceptual hash"
          value={phash?.value}
          url={NavUtils.makeScenesPHashMatchUrl(phash?.value)}
          target="_self"
          truncate
          internal
        />
        <URLField
          id="path"
          url={`file://${file.path}`}
          value={`file://${file.path}`}
        />
        <>
          <dt>
            <FormattedMessage id="actions_name" defaultMessage="Actions" />:
          </dt>
          <dd>
            <Button
              size="sm"
              variant="secondary"
              onClick={onOpenExternalPlayer}
              title="Open in external player"
            >
              <FormattedMessage id="actions.open_in_external_player" />
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => onScanThreats?.(file.id)}
              disabled={scanningThreats}
              title="Scan for security threats"
            >
              <FormattedMessage id="actions.scan_for_threats" defaultMessage="Scan for threats" />
            </Button>
          </dd>
        </>
        <TextField id="filesize">
          <span className="text-truncate">
            <FileSize size={file.size} />
          </span>
        </TextField>
        <TextField id="file_mod_time">
          <FormattedTime
            dateStyle="medium"
            timeStyle="medium"
            value={file.mod_time ?? 0}
          />
        </TextField>
        <TextField
          id="duration"
          value={TextUtils.secondsToTimestamp(file.duration ?? 0)}
          truncate
        />
        <TextField
          id="dimensions"
          value={`${file.width} x ${file.height}`}
          truncate
        />
        <TextField id="framerate">
          <FormattedMessage
            id="frames_per_second"
            values={{ value: intl.formatNumber(file.frame_rate ?? 0) }}
          />
        </TextField>
        <TextField id="bitrate">
          <FormattedMessage
            id="megabits_per_second"
            values={{
              value: intl.formatNumber((file.bit_rate ?? 0) / 1000000, {
                maximumFractionDigits: 2,
              }),
            }}
          />
        </TextField>
        <TextField
          id="media_info.video_codec"
          value={file.video_codec ?? ""}
          truncate
        />
        <TextField
          id="media_info.audio_codec"
          value={file.audio_codec ?? ""}
          truncate
        />
        <TextField id="threats_checked" name="Threats checked">
          {file.threats_scanned_at ? (
            <FormattedTime
              dateStyle="medium"
              timeStyle="medium"
              value={new Date(file.threats_scanned_at)}
            />
          ) : (
            <FormattedMessage id="threats_not_checked" defaultMessage="Not scanned" />
          )}
        </TextField>
        {file.threats && (
          <TextField id="threats" name="Threats">
            <span className="text-danger">
              {file.threats.split("\n").map((t, i, arr) => (
                <span key={i}>
                  {t}
                  {i < arr.length - 1 && <br />}
                </span>
              ))}
            </span>
          </TextField>
        )}
      </dl>
      {ofMany && onSetPrimaryFile && !primary && (
        <div>
          <Button
            className="edit-button"
            disabled={loading}
            onClick={onSetPrimaryFile}
          >
            <FormattedMessage id="actions.make_primary" />
          </Button>
          <Button
            className="edit-button"
            disabled={loading}
            onClick={onReassign}
          >
            <FormattedMessage id="actions.reassign" />
          </Button>
          <Button className="edit-button" onClick={onSplit}>
            <FormattedMessage id="actions.split" />
          </Button>
          <Button variant="danger" disabled={loading} onClick={onDeleteFile}>
            <FormattedMessage id="actions.delete_file" />
          </Button>
        </div>
      )}
    </div>
  );
};

interface ISceneFileInfoPanelProps {
  scene: GQL.SceneDataFragment;
  onRefetch?: () => void;
}

const _SceneFileInfoPanel: React.FC<ISceneFileInfoPanelProps> = (
  props: ISceneFileInfoPanelProps
) => {
  const Toast = useToast();
  const intl = useIntl();
  const [scanVideoFileThreats] = useScanVideoFileThreats();

  const [loading, setLoading] = useState(false);
  const [scanningThreats, setScanningThreats] = useState<string | null>(null);
  const [deletingFile, setDeletingFile] = useState<GQL.VideoFileDataFragment>();
  const [reassigningFile, setReassigningFile] =
    useState<GQL.VideoFileDataFragment>();

  async function onScanThreats(fileId: string) {
    try {
      setScanningThreats(fileId);
      await scanVideoFileThreats({ variables: { fileId } });
      Toast.success(intl.formatMessage({ id: "toast.scan_started" }));
      if (props.onRefetch) {
        setTimeout(() => props.onRefetch?.(), 15000);
      }
    } catch (e) {
      Toast.error(e);
    } finally {
      setScanningThreats(null);
    }
  }

  function renderStashIDs() {
    if (!props.scene.stash_ids.length) {
      return;
    }

    return (
      <>
        <dt>
          <FormattedMessage id="stash_ids" />
        </dt>
        <dd>
          <dl>
            {props.scene.stash_ids.map((stashID) => {
              return (
                <dd key={stashID.stash_id} className="row no-gutters">
                  <StashIDPill stashID={stashID} linkType="scenes" />
                </dd>
              );
            })}
          </dl>
        </dd>
      </>
    );
  }

  function renderFunscript() {
    if (props.scene.interactive) {
      return (
        <URLField
          name="Funscript"
          url={props.scene.paths.funscript}
          value={props.scene.paths.funscript}
          truncate
        />
      );
    }
  }

  function renderInteractiveSpeed() {
    if (props.scene.interactive_speed) {
      return (
        <TextField id="media_info.interactive_speed">
          <FormattedNumber value={props.scene.interactive_speed} />
        </TextField>
      );
    }
  }

  const { scene, onRefetch } = props;
  const filesPanel = useMemo(() => {
    if (scene.files.length === 0) {
      return;
    }

    if (scene.files.length === 1) {
      return (
        <FileInfoPanel
          sceneID={scene.id}
          file={scene.files[0]}
          onScanThreats={onScanThreats}
          scanningThreats={scanningThreats === scene.files[0].id}
        />
      );
    }

    async function onSetPrimaryFile(fileID: string) {
      try {
        setLoading(true);
        await mutateSceneSetPrimaryFile(scene.id, fileID);

        // Get the file name for the success message
        const targetFile = scene.files.find((f) => f.id === fileID);
        const fileName = targetFile
          ? TextUtils.fileNameFromPath(targetFile.path)
          : "Unknown file";

        // Refetch the scene data to update the UI
        if (onRefetch) {
          await onRefetch();
        }

        Toast.success(
          intl.formatMessage({ id: "toast.primary_file_set" }, { fileName })
        );
      } catch (e) {
        console.error("Error setting primary file:", e);
        Toast.error(
          intl.formatMessage(
            { id: "toast.error_setting_primary_file" },
            { error: e instanceof Error ? e.message : String(e) }
          )
        );
      } finally {
        setLoading(false);
      }
    }

    return (
      <Accordion defaultActiveKey={scene.files[0].id}>
        {deletingFile && (
          <DeleteFilesDialog
            onClose={() => setDeletingFile(undefined)}
            selected={[deletingFile]}
            onRefetch={onRefetch}
          />
        )}
        {reassigningFile && (
          <ReassignFilesDialog
            onClose={() => setReassigningFile(undefined)}
            selected={reassigningFile}
          />
        )}
        {scene.files.map((file, index) => (
          <Card key={file.id} className="scene-file-card">
            <Accordion.Toggle as={Card.Header} eventKey={file.id}>
              <TruncatedText text={TextUtils.fileNameFromPath(file.path)} />
            </Accordion.Toggle>
            <Accordion.Collapse eventKey={file.id}>
              <Card.Body>
                <FileInfoPanel
                  sceneID={scene.id}
                  file={file}
                  primary={index === 0}
                  ofMany
                  onSetPrimaryFile={() => onSetPrimaryFile(file.id)}
                  onDeleteFile={() => setDeletingFile(file)}
                  onReassign={() => setReassigningFile(file)}
                  onScanThreats={onScanThreats}
                  loading={loading}
                  scanningThreats={scanningThreats === file.id}
                />
              </Card.Body>
            </Accordion.Collapse>
          </Card>
        ))}
      </Accordion>
    );
  }, [scene, loading, Toast, deletingFile, reassigningFile, intl, onRefetch, onScanThreats, scanningThreats]);

  return (
    <>
      <dl className="container scene-file-info details-list">
        {props.scene.files.length > 0 && (
          <URLField
            id="media_info.stream"
            url={props.scene.paths.stream}
            value={props.scene.paths.stream}
            truncate
          />
        )}
        {renderFunscript()}
        {renderInteractiveSpeed()}
        {renderStashIDs()}
      </dl>

      {filesPanel}
    </>
  );
};

export const SceneFileInfoPanel = PatchComponent(
  "SceneFileInfoPanel",
  _SceneFileInfoPanel
);
export default SceneFileInfoPanel;
