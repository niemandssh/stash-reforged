import React, { useEffect, useMemo, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Prompt } from "react-router-dom";
import { Button, Form, Col, Row, InputGroup, Modal } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import * as yup from "yup";
import { useFormik } from "formik";
import { yupFormikValidate } from "src/utils/yup";
import { formikUtils } from "src/utils/form";
import { handleUnsavedChanges } from "src/utils/navigation";
import isEqual from "lodash-es/isEqual";
import { ImageInput } from "src/components/Shared/ImageInput";
import ImageUtils from "src/utils/image";
import { Tag, TagSelect } from "src/components/Tags/TagSelect";
import { FolderSelectDialog } from "src/components/Shared/FolderSelect/FolderSelectDialog";
import { Icon } from "src/components/Shared/Icon";
import {
  faEllipsisH,
  faFile,
  faFolder,
  faArrowUp,
} from "@fortawesome/free-solid-svg-icons";
import { useDirectory } from "src/core/StashService";
import TextUtils from "src/utils/text";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";

interface IProps {
  game: Partial<GQL.GameDataFragment>;
  isVisible: boolean;
  onSubmit: (input: GQL.GameCreateInput) => Promise<void>;
  onDelete: () => void;
  setImage?: (image?: string | null) => void;
  setEncodingImage?: (loading: boolean) => void;
}

export const GameEditPanel: React.FC<IProps> = ({
  game,
  onSubmit,
  onDelete,
  setImage,
  setEncodingImage,
}) => {
  const intl = useIntl();
  const [showFolderDialog, setShowFolderDialog] = useState(false);
  const [showFileDialog, setShowFileDialog] = useState(false);

  const isNew = game.id === undefined;

  const schema = yup.object({
    title: yup.string().required(),
    details: yup.string().ensure(),
    date: yup.string().nullable().optional(),
    folder_path: yup.string().nullable().optional(),
    executable_path: yup.string().nullable().optional(),
    urls: yup.array().of(yup.string().nullable()).ensure(),
    tag_ids: yup.array().of(yup.string()).ensure(),
    image: yup.string().nullable().optional(),
  });

  const initialValues = {
    title: game?.title ?? "",
    details: game?.details ?? "",
    date: game?.date ?? "",
    folder_path: game?.folder_path ?? "",
    executable_path: game?.executable_path ?? "",
    urls: game?.urls ?? [],
    tag_ids: (game?.tags ?? []).map((t) => t.id),
    image: undefined as string | null | undefined,
  };

  type InputValues = yup.InferType<typeof schema>;

  const formik = useFormik<InputValues>({
    initialValues,
    enableReinitialize: true,
    validate: yupFormikValidate(schema),
    onSubmit: async (values) => {
      const sanitizedUrls =
        values.urls?.filter(
          (url): url is string => !!url && url.trim().length > 0
        ) ?? [];
      const sanitizedTagIds =
        values.tag_ids?.filter(
          (id): id is string => !!id && id.trim().length > 0
        ) ?? [];

      const input: GQL.GameCreateInput = {
        title: values.title,
        details: values.details || undefined,
        date: values.date || undefined,
        folder_path: values.folder_path || undefined,
        executable_path: values.executable_path || undefined,
        urls: sanitizedUrls.length > 0 ? sanitizedUrls : undefined,
        tag_ids: sanitizedTagIds.length > 0 ? sanitizedTagIds : undefined,
        image: values.image || undefined,
      };

      await onSubmit(input);
      formik.resetForm({ values });
    },
  });

  const encodingImage = ImageUtils.usePasteImage((imageData) =>
    formik.setFieldValue("image", imageData)
  );

  useEffect(() => {
    if (setImage) {
      setImage(formik.values.image);
    }
  }, [formik.values.image, setImage]);

  useEffect(() => {
    if (setEncodingImage) {
      setEncodingImage(encodingImage);
    }
  }, [setEncodingImage, encodingImage]);

  function onImageLoad(imageData: string | null) {
    formik.setFieldValue("image", imageData);
  }

  function onCoverImageChange(event: React.FormEvent<HTMLInputElement>) {
    ImageUtils.onImageChange(event, onImageLoad);
  }

  const coverImagePreview = useMemo(() => {
    if (!formik.values.image) return null;
    return formik.values.image;
  }, [formik.values.image]);

  const mapTagsToSelect = (
    tags?: ReadonlyArray<GQL.SlimTagDataFragment>
  ): Tag[] => {
    if (!tags) {
      return [];
    }
    return tags.map((tag) => ({
      id: tag.id,
      name: tag.name,
      sort_name: tag.sort_name ?? undefined,
      aliases: tag.aliases ?? [],
      image_path: tag.image_path ?? undefined,
      is_pose_tag: tag.is_pose_tag ?? false,
      color: tag.color ?? undefined,
    }));
  };

  const [selectedTags, setSelectedTags] = useState<Tag[]>(
    mapTagsToSelect(game?.tags)
  );

  useEffect(() => {
    setSelectedTags(mapTagsToSelect(game?.tags));
  }, [game?.tags]);

  function onSetTags(items: Tag[]) {
    setSelectedTags(items);
    formik.setFieldValue(
      "tag_ids",
      items.map((item) => item.id)
    );
  }

  const splitProps = {
    labelProps: {
      column: true,
      sm: 3,
      xl: 12,
    },
    fieldProps: {
      sm: 9,
      xl: 12,
    },
  };
  const { renderField, renderInputField, renderDateField, renderURLListField } =
    formikUtils(intl, formik, splitProps);

  function renderDetailsField() {
    const props = {
      labelProps: {
        column: true,
        sm: 3,
        lg: 12,
      },
      fieldProps: {
        sm: 9,
        lg: 12,
      },
    };

    return renderInputField("details", "textarea", "details", props);
  }

  function renderCoverImageField() {
    const title = intl.formatMessage({ id: "cover_image" });
    const control = (
      <>
        {coverImagePreview && (
          <div className="mb-2">
            <img
              src={coverImagePreview}
              alt="Cover preview"
              style={{ maxWidth: "100%", maxHeight: "200px" }}
            />
          </div>
        )}
        <ImageInput
          isEditing
          onImageChange={onCoverImageChange}
          onImageURL={onImageLoad}
        />
      </>
    );

    return renderField("image", title, control);
  }

  function renderFolderPathField() {
    const title = intl.formatMessage({ id: "folder_path" });

    function onFolderSelectClosed(dir?: string) {
      if (dir) {
        formik.setFieldValue("folder_path", dir);
      }
      setShowFolderDialog(false);
    }

    const control = (
      <>
        {showFolderDialog ? (
          <FolderSelectDialog
            defaultValue={formik.values.folder_path ?? ""}
            onClose={onFolderSelectClosed}
          />
        ) : null}
        <InputGroup>
          <Form.Control
            className="btn-secondary"
            value={formik.values.folder_path ?? ""}
            placeholder={intl.formatMessage({ id: "setup.folder.file_path" })}
            onChange={(e) =>
              formik.setFieldValue("folder_path", e.currentTarget.value)
            }
            spellCheck={false}
          />
          <InputGroup.Append>
            <Button
              variant="secondary"
              onClick={() => setShowFolderDialog(true)}
            >
              <Icon icon={faEllipsisH} />
            </Button>
          </InputGroup.Append>
        </InputGroup>
      </>
    );

    return renderField("folder_path", title, control);
  }

  const FileSelectDialog: React.FC<{
    basePath: string;
    onClose: (filePath?: string) => void;
  }> = ({ basePath, onClose }) => {
    const [currentDirectory, setCurrentDirectory] = useState<string>(basePath);
    const { data, loading, error } = useDirectory(currentDirectory);
    const directories = data?.directory.directories ?? [];
    const files = data?.directory.files ?? [];
    const parent = data?.directory.parent;

    function navigateUp() {
      if (parent) {
        setCurrentDirectory(parent);
      } else if (basePath) {
        setCurrentDirectory(basePath);
      }
    }

    function navigateToPath(path: string) {
      setCurrentDirectory(path);
    }

    function selectFile(filePath: string) {
      onClose(filePath);
    }

    return (
      <Modal show onHide={() => onClose()}>
        <Modal.Header>
          <FormattedMessage
            id="actions.select_file"
            defaultMessage="Select File"
          />
        </Modal.Header>
        <Modal.Body>
          <div className="dialog-content">
            {loading && <LoadingIndicator />}
            {error && <div className="text-danger">{error.message}</div>}
            {!loading && !error && (
              <>
                {currentDirectory && parent && (
                  <div className="mb-2">
                    <Button
                      variant="secondary"
                      onClick={navigateUp}
                      disabled={loading}
                    >
                      <Icon icon={faArrowUp} className="mr-2" />
                      <FormattedMessage id="setup.folder.up_dir" />
                    </Button>
                  </div>
                )}
                <div className="directory-list">
                  <h6>
                    <FormattedMessage
                      id="directories"
                      defaultMessage="Directories"
                    />
                  </h6>
                  <ul className="list-unstyled">
                    {directories.map((dir) => (
                      <li key={dir}>
                        <Button
                          variant="link"
                          onClick={() => navigateToPath(dir)}
                          disabled={loading}
                        >
                          <Icon icon={faFolder} className="mr-2" />
                          {TextUtils.fileNameFromPath(dir)}
                        </Button>
                      </li>
                    ))}
                  </ul>
                  <h6 className="mt-3">
                    <FormattedMessage id="files" defaultMessage="Files" />
                  </h6>
                  <ul className="list-unstyled">
                    {files.map((file) => (
                      <li key={file}>
                        <Button
                          variant="link"
                          onClick={() => selectFile(file)}
                          disabled={loading}
                        >
                          <Icon icon={faFile} className="mr-2" />
                          {TextUtils.fileNameFromPath(file)}
                        </Button>
                      </li>
                    ))}
                  </ul>
                </div>
              </>
            )}
          </div>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => onClose()}>
            <FormattedMessage id="actions.cancel" />
          </Button>
        </Modal.Footer>
      </Modal>
    );
  };

  function renderExecutablePathField() {
    const title = intl.formatMessage({ id: "executable_path" });
    const folderPath = formik.values.folder_path ?? "";

    function getRelativePath(fullPath: string): string {
      if (!folderPath || !fullPath) {
        return fullPath;
      }
      try {
        // Normalize paths
        const base = folderPath.replace(/\\/g, "/").replace(/\/$/, "");
        const full = fullPath.replace(/\\/g, "/");

        if (full.startsWith(base + "/")) {
          return full.substring(base.length + 1);
        }
        return fullPath;
      } catch {
        return fullPath;
      }
    }

    function onFileSelectClosed(filePath?: string) {
      if (filePath) {
        const relativePath = getRelativePath(filePath);
        formik.setFieldValue("executable_path", relativePath);
      }
      setShowFileDialog(false);
    }

    const control = (
      <>
        {showFileDialog ? (
          <FileSelectDialog
            basePath={folderPath}
            onClose={onFileSelectClosed}
          />
        ) : null}
        <InputGroup>
          <Form.Control
            className="btn-secondary"
            value={formik.values.executable_path ?? ""}
            placeholder={intl.formatMessage({ id: "executable_path" })}
            onChange={(e) =>
              formik.setFieldValue("executable_path", e.currentTarget.value)
            }
            spellCheck={false}
          />
          <InputGroup.Append>
            <Button
              variant="secondary"
              onClick={() => setShowFileDialog(true)}
              disabled={!folderPath}
            >
              <Icon icon={faEllipsisH} />
            </Button>
          </InputGroup.Append>
        </InputGroup>
      </>
    );

    return renderField("executable_path", title, control);
  }

  return (
    <div id="game-edit-details">
      <Prompt
        when={formik.dirty}
        message={handleUnsavedChanges(intl, "games", game?.id)}
      />

      <Form noValidate onSubmit={formik.handleSubmit}>
        <Row className="form-container edit-buttons-container px-3 pt-3">
          <div className="edit-buttons mb-3 pl-0">
            <Button
              className="edit-button"
              variant="primary"
              disabled={
                (!isNew && !formik.dirty) || !isEqual(formik.errors, {})
              }
              onClick={() => formik.submitForm()}
            >
              <FormattedMessage id="actions.save" />
            </Button>
            {!isNew && (
              <Button
                className="edit-button"
                variant="danger"
                onClick={() => onDelete()}
              >
                <FormattedMessage id="actions.delete" />
              </Button>
            )}
          </div>
        </Row>
        <Row className="form-container px-3">
          <Col lg={7} xl={12}>
            {renderInputField("title")}
            {renderDateField("date")}
            {renderFolderPathField()}
            {renderExecutablePathField()}
            <Form.Group controlId="game-tags">
              <Form.Label>
                <FormattedMessage id="tags" />
              </Form.Label>
              <TagSelect
                isMulti
                onSelect={onSetTags}
                values={selectedTags}
                instanceId={`game-tags-${game.id ?? "new"}`}
              />
            </Form.Group>
            {renderURLListField("urls")}
          </Col>
          <Col lg={5} xl={12}>
            {renderDetailsField()}

            {renderCoverImageField()}
          </Col>
        </Row>
      </Form>
    </div>
  );
};
