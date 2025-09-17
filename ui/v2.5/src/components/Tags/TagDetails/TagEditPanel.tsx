import React, { useEffect, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import * as yup from "yup";
import { DetailsEditNavbar } from "src/components/Shared/DetailsEditNavbar";
import { Form } from "react-bootstrap";
import ImageUtils from "src/utils/image";
import { useFormik } from "formik";
import { Prompt } from "react-router-dom";
import Mousetrap from "mousetrap";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import isEqual from "lodash-es/isEqual";
import { useToast } from "src/hooks/Toast";
import { handleUnsavedChanges } from "src/utils/navigation";
import { formikUtils } from "src/utils/form";
import { yupFormikValidate, yupUniqueAliases } from "src/utils/yup";
import { Tag, TagSelect } from "../TagSelect";
import { ColorPresetSelector } from "../../Shared/ColorPresetSelector";
import { ColorPalette } from "../../Shared/ColorPalette";

interface ITagEditPanel {
  tag: Partial<GQL.TagDataFragment>;
  onSubmit: (tag: GQL.TagCreateInput | GQL.TagUpdateInput) => Promise<void>;
  onCancel: () => void;
  onDelete: () => void;
  setImage: (image?: string | null) => void;
  setEncodingImage: (loading: boolean) => void;
}

export const TagEditPanel: React.FC<ITagEditPanel> = ({
  tag,
  onSubmit,
  onCancel,
  onDelete,
  setImage,
  setEncodingImage,
}) => {
  const intl = useIntl();
  const Toast = useToast();

  const isNew = tag.id === undefined;

  // Network state
  const [isLoading, setIsLoading] = useState(false);

  const [childTags, setChildTags] = useState<Tag[]>([]);
  const [parentTags, setParentTags] = useState<Tag[]>([]);

  const schema = yup.object({
    name: yup.string().required(),
    sort_name: yup.string().ensure(),
    aliases: yupUniqueAliases(intl, "name"),
    description: yup.string().ensure(),
    parent_ids: yup.array(yup.string().required()).defined(),
    child_ids: yup.array(yup.string().required()).defined(),
    ignore_auto_tag: yup.boolean().defined(),
    is_pose_tag: yup.boolean().defined(),
    ignore_suggestions: yup.boolean().defined(),
    weight: yup.number().min(0).max(1).default(0.5),
    color: yup.string().nullable().optional(),
    image: yup.string().nullable().optional(),
  });

  const initialValues = {
    name: tag?.name ?? "",
    sort_name: tag?.sort_name ?? "",
    aliases: tag?.aliases ?? [],
    description: tag?.description ?? "",
    parent_ids: (tag?.parents ?? []).map((t) => t.id),
    child_ids: (tag?.children ?? []).map((t) => t.id),
    ignore_auto_tag: tag?.ignore_auto_tag ?? false,
    is_pose_tag: tag?.is_pose_tag ?? false,
    ignore_suggestions: tag?.ignore_suggestions ?? false,
    weight: tag?.weight ?? 0.5,
    color: tag?.color ?? "",
    image: tag?.image_path ?? null,
  };

  type InputValues = yup.InferType<typeof schema>;

  const formik = useFormik<InputValues>({
    initialValues,
    enableReinitialize: true,
    validate: yupFormikValidate(schema),
    onSubmit: (values) => onSave(schema.cast(values)),
  });

  function onSetParentTags(items: Tag[]) {
    setParentTags(items);
    formik.setFieldValue(
      "parent_ids",
      items.map((item) => item.id)
    );
  }

  function onSetChildTags(items: Tag[]) {
    setChildTags(items);
    formik.setFieldValue(
      "child_ids",
      items.map((item) => item.id)
    );
  }

  useEffect(() => {
    setParentTags(tag.parents ?? []);
  }, [tag.parents]);

  useEffect(() => {
    setChildTags(tag.children ?? []);
  }, [tag.children]);

  // set up hotkeys
  useEffect(() => {
    Mousetrap.bind("s s", () => {
      if (formik.dirty) {
        formik.submitForm();
      }
    });

    return () => {
      Mousetrap.unbind("s s");
    };
  });

  async function onSave(input: InputValues) {
    setIsLoading(true);
    try {
      if (isNew) {
        // Convert InputValues to GQL create input format
        const gqlInput: GQL.TagCreateInput = {
          name: input.name,
          sort_name: input.sort_name || undefined,
          aliases: input.aliases,
          description: input.description || undefined,
          parent_ids: input.parent_ids,
          child_ids: input.child_ids,
          ignore_auto_tag: input.ignore_auto_tag,
          is_pose_tag: input.is_pose_tag,
          ignore_suggestions: input.ignore_suggestions,
          weight: input.weight,
          color: input.color || undefined,
          image: input.image || undefined,
        };
        
        await onSubmit(gqlInput);
      } else {
        // Convert InputValues to GQL update input format
        const gqlInput: GQL.TagUpdateInput = {
          id: tag.id!,
          name: input.name,
          sort_name: input.sort_name || undefined,
          aliases: input.aliases,
          description: input.description || undefined,
          parent_ids: input.parent_ids,
          child_ids: input.child_ids,
          ignore_auto_tag: input.ignore_auto_tag,
          is_pose_tag: input.is_pose_tag,
          ignore_suggestions: input.ignore_suggestions,
          weight: input.weight,
          color: input.color || undefined,
          image: input.image || undefined,
        };
        
        await onSubmit(gqlInput);
      }
      formik.resetForm();
    } catch (e) {
      Toast.error(e);
    }
    setIsLoading(false);
  }

  const encodingImage = ImageUtils.usePasteImage(onImageLoad);

  useEffect(() => {
    setImage(formik.values.image);
  }, [formik.values.image, setImage]);

  useEffect(() => {
    setEncodingImage(encodingImage);
  }, [setEncodingImage, encodingImage]);

  function onImageLoad(imageData: string | null) {
    formik.setFieldValue("image", imageData);
  }

  function onImageChange(event: React.FormEvent<HTMLInputElement>) {
    ImageUtils.onImageChange(event, onImageLoad);
  }

  const { renderField, renderInputField, renderStringListField } = formikUtils(
    intl,
    formik
  );

  // Утилита для определения цвета текста на основе цвета фона
  const getContrastColor = (backgroundColor: string): string => {
    if (!backgroundColor) return "#000000";
    
    let r = 0, g = 0, b = 0;
    
    // Обработка hex цветов
    if (backgroundColor.startsWith("#")) {
      const hex = backgroundColor.replace("#", "");
      if (hex.length === 3) {
        r = parseInt(hex[0] + hex[0], 16);
        g = parseInt(hex[1] + hex[1], 16);
        b = parseInt(hex[2] + hex[2], 16);
      } else if (hex.length === 6) {
        r = parseInt(hex.substr(0, 2), 16);
        g = parseInt(hex.substr(2, 2), 16);
        b = parseInt(hex.substr(4, 2), 16);
      }
    }
    // Обработка rgb/rgba цветов
    else if (backgroundColor.startsWith("rgb")) {
      const matches = backgroundColor.match(/\d+/g);
      if (matches && matches.length >= 3) {
        r = parseInt(matches[0]);
        g = parseInt(matches[1]);
        b = parseInt(matches[2]);
      }
    }
    // Обработка hsl цветов
    else if (backgroundColor.startsWith("hsl")) {
      const matches = backgroundColor.match(/\d+/g);
      if (matches && matches.length >= 3) {
        const h = parseInt(matches[0]) / 360;
        const s = parseInt(matches[1]) / 100;
        const l = parseInt(matches[2]) / 100;
        
        const hue2rgb = (p: number, q: number, t: number) => {
          if (t < 0) t += 1;
          if (t > 1) t -= 1;
          if (t < 1/6) return p + (q - p) * 6 * t;
          if (t < 1/2) return q;
          if (t < 2/3) return p + (q - p) * (2/3 - t) * 6;
          return p;
        };
        
        if (s === 0) {
          r = g = b = l;
        } else {
          const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
          const p = 2 * l - q;
          r = hue2rgb(p, q, h + 1/3);
          g = hue2rgb(p, q, h);
          b = hue2rgb(p, q, h - 1/3);
        }
        
        r = Math.round(r * 255);
        g = Math.round(g * 255);
        b = Math.round(b * 255);
      }
    }
    
    const brightness = (r * 299 + g * 587 + b * 114) / 1000;
    return brightness > 128 ? "#000000" : "#ffffff";
  };

  function renderColorField() {
    const title = intl.formatMessage({ id: "Color" });
    const backgroundColor = formik.values.color || "#bfccd6";
    const textColor = getContrastColor(backgroundColor);
    
    const control = (
      <div>
        {/* Стандартное поле цвета */}
        <div className="d-flex align-items-center mb-3">
          <Form.Control
            type="color"
            name="color"
            className="text-input mr-2"
            value={formik.values.color || "#bfccd6"}
            onChange={(e) => formik.setFieldValue("color", e.target.value)}
            style={{ width: "60px", height: "38px" }}
          />
          <Form.Control
            type="text"
            name="color_text"
            className="text-input"
            value={formik.values.color || ""}
            onChange={(e) => formik.setFieldValue("color", e.target.value)}
            placeholder="#000000"
            style={{ flex: 1 }}
          />
        </div>

        {/* Превью тега */}
        <div className="tag-preview d-flex align-items-center mb-3">
          <small className="text-muted mr-3">
            <FormattedMessage id="preview" />:
          </small>
          <div 
            className="px-2 py-1 rounded"
            style={{ 
              backgroundColor: backgroundColor,
              color: textColor,
              border: `1px solid ${backgroundColor}`,
              fontSize: "0.875rem",
              fontWeight: "500"
            }}
          >
            {formik.values.name || intl.formatMessage({ id: "tag" })}
          </div>
        </div>

        {/* Селектор пресетов цветов */}
        <ColorPresetSelector
          selectedColor={formik.values.color || ""}
          onColorSelect={(color) => formik.setFieldValue("color", color)}
          onPresetSelect={(preset) => {
            formik.setFieldValue("color", preset.color);
          }}
        />

        {/* Палитра всех цветов тегов */}
        <div className="mt-2">
          <ColorPalette
            onColorSelect={(color) => formik.setFieldValue("color", color)}
          />
        </div>
      </div>
    );

    return renderField("color", title, control);
  }

  // Утилита для генерации случайного цвета если цвет не задан
  // const _generateRandomColor = (seed: string): string => {
  //   let hash = 0;
  //   for (let i = 0; i < seed.length; i++) {
  //     hash = seed.charCodeAt(i) + ((hash << 5) - hash);
  //   }
  //   
  //   const hue = Math.abs(hash) % 360;
  //   const saturation = 60 + (Math.abs(hash) % 30);
  //   const lightness = 45 + (Math.abs(hash >> 8) % 20);
  //   
  //   return `hsl(${hue}, ${saturation}%, ${lightness}%)`;
  // };

  function renderParentTagsField() {
    const title = intl.formatMessage({ id: "parent_tags" });
    const control = (
      <TagSelect
        isMulti
        onSelect={onSetParentTags}
        values={parentTags}
        excludeIds={[...(tag?.id ? [tag.id] : []), ...formik.values.child_ids]}
        creatable={false}
        hoverPlacement="right"
      />
    );

    return renderField("parent_ids", title, control);
  }

  function renderSubTagsField() {
    const title = intl.formatMessage({ id: "sub_tags" });
    const control = (
      <TagSelect
        isMulti
        onSelect={onSetChildTags}
        values={childTags}
        excludeIds={[...(tag?.id ? [tag.id] : []), ...formik.values.parent_ids]}
        creatable={false}
        hoverPlacement="right"
      />
    );

    return renderField("child_ids", title, control);
  }

  if (isLoading) return <LoadingIndicator />;

  // TODO: CSS class
  return (
    <div>
      {isNew && (
        <h2>
          <FormattedMessage
            id="actions.add_entity"
            values={{ entityType: intl.formatMessage({ id: "tag" }) }}
          />
        </h2>
      )}

      <Prompt
        when={formik.dirty}
        message={(location, action) => {
          // Check if it's a redirect after tag creation
          if (action === "PUSH" && location.pathname.startsWith("/tags/")) {
            return true;
          }

          return handleUnsavedChanges(intl, "tags", tag.id)(location);
        }}
      />

      <Form noValidate onSubmit={formik.handleSubmit} id="tag-edit">
        {renderInputField("name")}
        {renderInputField("sort_name", "text")}
        {renderStringListField("aliases")}
        {renderInputField("description", "textarea")}
        {renderInputField("weight", "number", "tag_weight")}
        {renderColorField()}
        {renderParentTagsField()}
        {renderSubTagsField()}
        <hr />
        {renderInputField("ignore_auto_tag", "checkbox")}
        {renderInputField("is_pose_tag", "checkbox")}
        {renderInputField("ignore_suggestions", "checkbox")}
      </Form>

      <DetailsEditNavbar
        objectName={tag?.name ?? intl.formatMessage({ id: "tag" })}
        classNames="col-xl-9 mt-3"
        isNew={isNew}
        isEditing
        onToggleEdit={onCancel}
        onSave={formik.handleSubmit}
        saveDisabled={(!isNew && !formik.dirty) || !isEqual(formik.errors, {})}
        onImageChange={onImageChange}
        onImageChangeURL={onImageLoad}
        onClearImage={() => onImageLoad(null)}
        onDelete={onDelete}
        acceptSVG
      />
    </div>
  );
};
