import {
  faMinus,
  faPencilAlt,
  faPlus,
  faFolder,
  faFile,
  faFileAlt,
  faList,
  faThLarge,
  faArrowUp,
} from "@fortawesome/free-solid-svg-icons";
import React, { useState } from "react";
import { Button, Col, Form, Row, OverlayTrigger, Tooltip } from "react-bootstrap";
import { useIntl, FormattedMessage } from "react-intl";
import { Icon } from "src/components/Shared/Icon";
import { ModalComponent } from "src/components/Shared/Modal";
import { ConfigurationContext } from "src/hooks/Config";
import { useDirectory } from "src/core/StashService";
import TextUtils from "src/utils/text";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";

type ViewMode = "tree" | "tiles";

interface IDirectorySelectionDialogProps {
  animation?: boolean;
  initialPaths?: string[];
  allowEmpty?: boolean;
  onClose: (paths?: string[]) => void;
}

export const DirectorySelectionDialog: React.FC<
  IDirectorySelectionDialogProps
> = ({ animation, allowEmpty = false, initialPaths = [], onClose }) => {
  const intl = useIntl();
  const { configuration } = React.useContext(ConfigurationContext);

  const libraryPaths = configuration?.general.stashes.map((s) => s.path);

  const [paths, setPaths] = useState<string[]>(initialPaths);
  const [currentDirectory, setCurrentDirectory] = useState<string>("");
  const [viewMode, setViewMode] = useState<ViewMode>("tiles");
  const [navigatedPaths, setNavigatedPaths] = useState<string[]>([]);

  const { data, loading, error } = useDirectory(currentDirectory);
  const directories = data?.directory.directories ?? [];
  const files = data?.directory.files ?? [];
  const parent = data?.directory.parent;

  function removePath(p: string) {
    setPaths(paths.filter((path) => path !== p));
  }

  function addPath(p: string) {
    if (p && !paths.includes(p)) {
      setPaths(paths.concat(p));
    }
  }

  function navigateToPath(path: string) {
    setCurrentDirectory(path);
    if (!navigatedPaths.includes(path)) {
      setNavigatedPaths([...navigatedPaths, path]);
    }
  }

  function navigateUp() {
    if (parent) {
      navigateToPath(parent);
    } else if (libraryPaths && libraryPaths.length > 0) {
      navigateToPath("");
    }
  }

  const renderTreeView = () => {
    return (
      <ul className="folder-list">
        {currentDirectory && parent && (
          <li className="folder-list-parent folder-list-item">
            <Button
              variant="link"
              onClick={() => navigateUp()}
              disabled={loading}
            >
              <Icon icon={faArrowUp} className="mr-2" />
              <span>
                <FormattedMessage id="setup.folder.up_dir" />
              </span>
            </Button>
          </li>
        )}
        {directories.map((dir) => (
          <li key={dir} className="folder-list-item">
            <OverlayTrigger
              placement="top"
              overlay={<Tooltip id={`dir-tooltip-${dir}`}>{dir}</Tooltip>}
            >
              <Button
                variant="link"
                onClick={() => navigateToPath(dir)}
                disabled={loading}
                className="folder-list-name"
              >
                <Icon icon={faFolder} className="mr-2" />
                <span>{dir}</span>
              </Button>
            </OverlayTrigger>
            <OverlayTrigger
              placement="top"
              overlay={
                <Tooltip id={`dir-add-tooltip-${dir}`}>
                  {intl.formatMessage({ id: "actions.add" })}
                </Tooltip>
              }
            >
              <Button
                size="sm"
                variant="secondary"
                className="folder-list-add"
                onClick={(e) => {
                  e.stopPropagation();
                  addPath(dir);
                }}
                disabled={paths.includes(dir)}
              >
                <Icon icon={faPlus} />
              </Button>
            </OverlayTrigger>
          </li>
        ))}
        {files.map((file) => (
          <li key={file} className="folder-list-item">
            <OverlayTrigger
              placement="top"
              overlay={<Tooltip id={`file-tooltip-${file}`}>{file}</Tooltip>}
            >
              <Button
                variant="link"
                onClick={() => addPath(file)}
                disabled={paths.includes(file)}
                className="folder-list-name"
              >
                <Icon icon={faFile} className="mr-2" />
                <span>{TextUtils.fileNameFromPath(file)}</span>
              </Button>
            </OverlayTrigger>
            <OverlayTrigger
              placement="top"
              overlay={
                <Tooltip id={`file-add-tooltip-${file}`}>
                  {intl.formatMessage({ id: "actions.add" })}
                </Tooltip>
              }
            >
              <Button
                size="sm"
                variant="secondary"
                className="folder-list-add"
                onClick={(e) => {
                  e.stopPropagation();
                  addPath(file);
                }}
                disabled={paths.includes(file)}
              >
                <Icon icon={faPlus} />
              </Button>
            </OverlayTrigger>
          </li>
        ))}
      </ul>
    );
  };

  const renderTilesView = () => {
    return (
      <div className="directory-tiles">
        {currentDirectory && parent && (
          <div className="tile-item tile-up">
            <Button
              variant="secondary"
              className="tile-button"
              onClick={() => navigateUp()}
              disabled={loading}
            >
              <div className="tile-label">
                <Icon icon={faArrowUp} className="mr-2" />
                {intl.formatMessage({ id: "setup.folder.up_dir" })}
              </div>
            </Button>
          </div>
        )}
        {directories.map((dir) => (
          <div key={dir} className="tile-item">
            <div className="tile-wrapper">
              <Button
                variant="secondary"
                className="tile-button"
                onClick={() => navigateToPath(dir)}
                disabled={loading}
              >
                <Icon icon={faFolder} size="2x" />
                <OverlayTrigger
                  placement="top"
                  overlay={<Tooltip id={`tile-dir-tooltip-${dir}`}>{dir}</Tooltip>}
                >
                  <div className="tile-label">{TextUtils.fileNameFromPath(dir)}</div>
                </OverlayTrigger>
              </Button>
              <OverlayTrigger
                placement="top"
                overlay={
                  <Tooltip id={`tile-dir-add-tooltip-${dir}`}>
                    {intl.formatMessage({ id: "actions.add" })}
                  </Tooltip>
                }
              >
                <Button
                  size="sm"
                  variant="secondary"
                  className="tile-add-button"
                  onClick={(e) => {
                    e.stopPropagation();
                    addPath(dir);
                  }}
                  disabled={paths.includes(dir)}
                >
                  <Icon icon={faPlus} />
                </Button>
              </OverlayTrigger>
            </div>
          </div>
        ))}
        {files.map((file) => (
          <div key={file} className="tile-item">
            <div className="tile-wrapper">
              <Button
                variant="secondary"
                className="tile-button"
                onClick={() => addPath(file)}
                disabled={paths.includes(file)}
              >
                <Icon icon={faFile} size="2x" />
                <OverlayTrigger
                  placement="top"
                  overlay={<Tooltip id={`tile-file-tooltip-${file}`}>{file}</Tooltip>}
                >
                  <div className="tile-label">{TextUtils.fileNameFromPath(file)}</div>
                </OverlayTrigger>
              </Button>
              <OverlayTrigger
                placement="top"
                overlay={
                  <Tooltip id={`tile-file-add-tooltip-${file}`}>
                    {intl.formatMessage({ id: "actions.add" })}
                  </Tooltip>
                }
              >
                <Button
                  size="sm"
                  variant="secondary"
                  className="tile-add-button"
                  onClick={(e) => {
                    e.stopPropagation();
                    addPath(file);
                  }}
                  disabled={paths.includes(file)}
                >
                  <Icon icon={faPlus} />
                </Button>
              </OverlayTrigger>
            </div>
          </div>
        ))}
      </div>
    );
  };

  React.useEffect(() => {
    if (!currentDirectory && libraryPaths && libraryPaths.length > 0) {
      setCurrentDirectory(libraryPaths[0]);
    }
  }, [libraryPaths]);

  return (
    <ModalComponent
      show
      modalProps={{ animation, size: "xl", dialogClassName: "directory-selection-dialog" }}
      disabled={!allowEmpty && paths.length === 0}
      icon={faPencilAlt}
      header={intl.formatMessage({ id: "actions.select_folders" })}
      accept={{
        onClick: () => {
          onClose(paths);
        },
        text: intl.formatMessage({ id: "actions.confirm" }),
      }}
      cancel={{
        onClick: () => onClose(),
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
      }}
    >
      <div className="directory-selection-container">
        <Row className="directory-selection-row">
          <Col xs={12} md={6}>
            <div className="selection-panel">
              <div className="selection-header">
              <Form.Group className="flex-grow-1">
                <Form.Control
                  className="btn-secondary"
                  placeholder={intl.formatMessage({ id: "setup.folder.file_path" })}
                  onChange={(e) => {
                    setCurrentDirectory(e.currentTarget.value);
                  }}
                  value={currentDirectory}
                  spellCheck={false}
                />
              </Form.Group>
              <div className="view-mode-toggle">
                <Button
                  variant={viewMode === "tiles" ? "primary" : "secondary"}
                  size="sm"
                  onClick={() => setViewMode("tiles")}
                  title={intl.formatMessage({ id: "display_mode.grid" })}
                >
                  <Icon icon={faThLarge} />
                </Button>
                <Button
                  variant={viewMode === "tree" ? "primary" : "secondary"}
                  size="sm"
                  onClick={() => setViewMode("tree")}
                  title={intl.formatMessage({ id: "display_mode.list" })}
                >
                  <Icon icon={faList} />
                </Button>
              </div>
            </div>
            <div className="selection-content">
              {loading ? (
                <LoadingIndicator message="" />
              ) : error ? (
                <div className="text-danger">
                  {intl.formatMessage({ id: "errors.error" })}: {error.message}
                </div>
              ) : (
                <>
                  {viewMode === "tree" ? renderTreeView() : renderTilesView()}
                </>
              )}
              </div>
            </div>
          </Col>
          <Col xs={12} md={6}>
            <div className="selected-panel">
              <div className="selected-header">
              <h5>
                <FormattedMessage
                  id="config.tasks.selected_folders_and_files"
                  defaultMessage="Selected folders and files"
                />{" "}
                ({paths.length})
              </h5>
            </div>
            <div className="selected-content">
              {paths.length === 0 ? (
                <div className="text-muted text-center p-3">
                  {intl.formatMessage({ id: "config.tasks.scan_for_content_desc" })}
                </div>
              ) : (
                <div className="selected-list">
                  {paths.map((p) => {
                    const pathParts = p.split(/[\\/]/);
                    const lastPart = pathParts[pathParts.length - 1];
                    const isFile =
                      lastPart &&
                      lastPart.includes(".") &&
                      !p.endsWith("/") &&
                      !p.endsWith("\\");
                    return (
                      <div key={p} className="selected-item">
                      <div className="selected-item-icon">
                        <Icon icon={isFile ? faFile : faFolder} />
                      </div>
                      <OverlayTrigger
                        placement="top"
                        overlay={<Tooltip id={`selected-path-tooltip-${p}`}>{p}</Tooltip>}
                      >
                        <div className="selected-item-path">{p}</div>
                      </OverlayTrigger>
                      <Button
                          size="sm"
                          variant="danger"
                          title={intl.formatMessage({ id: "actions.delete" })}
                          onClick={() => removePath(p)}
                        >
                          <Icon icon={faMinus} />
                        </Button>
                      </div>
                    );
                  })}
                </div>
              )}
              </div>
            </div>
          </Col>
        </Row>
      </div>
    </ModalComponent>
  );
};
