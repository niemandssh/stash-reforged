import { Badge, OverlayTrigger, Tooltip } from "react-bootstrap";
import React, { useMemo } from "react";
import { Link } from "react-router-dom";
import cx from "classnames";
import NavUtils, { INamedObject } from "src/utils/navigation";
import TextUtils from "src/utils/text";
import { IFile, IObjectWithTitleFiles, objectTitle } from "src/core/files";
import { galleryTitle } from "src/core/galleries";
import * as GQL from "src/core/generated-graphql";
import { TagPopover } from "../Tags/TagPopover";
import { markerTitle } from "src/core/markers";
import { Placement } from "react-bootstrap/esm/Overlay";
import { faFolderTree } from "@fortawesome/free-solid-svg-icons";
import { Icon } from "../Shared/Icon";
import { FormattedMessage } from "react-intl";
import { PatchComponent } from "src/patch";

const getContrastColor = (backgroundColor: string): string => {
  if (!backgroundColor) return "#000000";
  
  let r = 0, g = 0, b = 0;
  
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
  else if (backgroundColor.startsWith("rgb")) {
    const matches = backgroundColor.match(/\d+/g);
    if (matches && matches.length >= 3) {
      r = parseInt(matches[0]);
      g = parseInt(matches[1]);
      b = parseInt(matches[2]);
    }
  }
  else if (backgroundColor.startsWith("hsl")) {
    const matches = backgroundColor.match(/\d+/g);
    if (matches && matches.length >= 3) {
      const h = parseInt(matches[0]) / 360;
      const s = parseInt(matches[1]) / 100;
      const l = parseInt(matches[2]) / 100;
      
      const hue2rgb = (p: number, q: number, t: number) => {
        if (t < 0) t += 1;
        if (t > 1) t += 1;
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
  const textColor = brightness > 128 ? "#000000" : "#ffffff";
  
  return textColor;
};

type SceneMarkerFragment = Pick<GQL.SceneMarker, "id" | "title" | "seconds"> & {
  scene: Pick<GQL.Scene, "id">;
  primary_tag: Pick<GQL.Tag, "id" | "name">;
};

interface ISortNameLinkProps {
  link: string;
  className?: string;
  sortName?: string;
  style?: React.CSSProperties;
}

const SortNameLinkComponent: React.FC<ISortNameLinkProps> = ({
  link,
  sortName,
  className,
  style,
  children,
}) => {
  return (
    <Badge
      data-name={className}
      data-sort-name={sortName}
      className={cx("tag-item", className)}
      variant="secondary"
      style={style}
    >
      <Link to={link} target="_blank" rel="noopener noreferrer">{children}</Link>
    </Badge>
  );
};

interface ICommonLinkProps {
  link: string;
  className?: string;
}

const CommonLinkComponent: React.FC<ICommonLinkProps> = ({
  link,
  className,
  children,
}) => {
  return (
    <Badge className={cx("tag-item", className)} variant="secondary">
      <Link to={link}>{children}</Link>
    </Badge>
  );
};

interface IPerformerLinkProps {
  performer: INamedObject & { disambiguation?: string | null };
  linkType?: "scene" | "gallery" | "image" | "scene_marker";
  className?: string;
}

export type PerformerLinkType = IPerformerLinkProps["linkType"];

export const PerformerLink: React.FC<IPerformerLinkProps> = ({
  performer,
  linkType = "scene",
  className,
}) => {
  const link = useMemo(() => {
    switch (linkType) {
      case "gallery":
        return NavUtils.makePerformerGalleriesUrl(performer);
      case "image":
        return NavUtils.makePerformerImagesUrl(performer);
      case "scene_marker":
        return NavUtils.makePerformerSceneMarkersUrl(performer);
      case "scene":
      default:
        return NavUtils.makePerformerScenesUrl(performer);
    }
  }, [performer, linkType]);

  const title = performer.name || "";

  return (
    <CommonLinkComponent link={link} className={className}>
      <span>{title}</span>
      {performer.disambiguation && (
        <span className="performer-disambiguation">{` (${performer.disambiguation})`}</span>
      )}
    </CommonLinkComponent>
  );
};

interface IGroupLinkProps {
  group: INamedObject;
  description?: string;
  linkType?: "scene" | "sub_group" | "details";
  className?: string;
}

export const GroupLink: React.FC<IGroupLinkProps> = ({
  group,
  description,
  linkType = "scene",
  className,
}) => {
  const link = useMemo(() => {
    switch (linkType) {
      case "scene":
        return NavUtils.makeGroupScenesUrl(group);
      case "sub_group":
        return NavUtils.makeSubGroupsUrl(group);
      case "details":
        return NavUtils.makeGroupUrl(group.id ?? "");
    }
  }, [group, linkType]);

  const title = group.name || "";

  return (
    <CommonLinkComponent link={link} className={className}>
      {title}{" "}
      {description && (
        <span className="group-description">({description})</span>
      )}
    </CommonLinkComponent>
  );
};

interface ISceneMarkerLinkProps {
  marker: SceneMarkerFragment;
  linkType?: "scene";
  className?: string;
}

export const SceneMarkerLink: React.FC<ISceneMarkerLinkProps> = ({
  marker,
  linkType = "scene",
  className,
}) => {
  const link = useMemo(() => {
    switch (linkType) {
      case "scene":
        return NavUtils.makeSceneMarkerUrl(marker);
    }
  }, [marker, linkType]);

  const title = `${markerTitle(marker)} - ${TextUtils.secondsToTimestamp(
    marker.seconds || 0
  )}`;

  return (
    <CommonLinkComponent link={link} className={className}>
      {title}
    </CommonLinkComponent>
  );
};

interface IObjectWithIDTitleFiles extends IObjectWithTitleFiles {
  id: string;
}

interface ISceneLinkProps {
  scene: IObjectWithIDTitleFiles;
  linkType?: "details";
  className?: string;
}

export const SceneLink: React.FC<ISceneLinkProps> = ({
  scene,
  linkType = "details",
  className,
}) => {
  const link = useMemo(() => {
    switch (linkType) {
      case "details":
        return `/scenes/${scene.id}`;
    }
  }, [scene, linkType]);

  const title = objectTitle(scene);

  return (
    <CommonLinkComponent link={link} className={className}>
      {title}
    </CommonLinkComponent>
  );
};

interface IGallery extends IObjectWithIDTitleFiles {
  folder?: GQL.Maybe<IFile>;
}

interface IGalleryLinkProps {
  gallery: IGallery;
  linkType?: "details";
  className?: string;
}

export const GalleryLink: React.FC<IGalleryLinkProps> = ({
  gallery,
  linkType = "details",
  className,
}) => {
  const link = useMemo(() => {
    switch (linkType) {
      case "details":
        return `/galleries/${gallery.id}`;
    }
  }, [gallery, linkType]);

  const title = galleryTitle(gallery);

  return (
    <CommonLinkComponent link={link} className={className}>
      {title}
    </CommonLinkComponent>
  );
};

interface ITagLinkProps {
  tag: INamedObject & { color?: string | null };
  linkType?:
    | "scene"
    | "gallery"
    | "image"
    | "details"
    | "performer"
    | "group"
    | "studio"
    | "scene_marker";
  className?: string;
  hoverPlacement?: Placement;
  showHierarchyIcon?: boolean;
  hierarchyTooltipID?: string;
}

export const TagLink: React.FC<ITagLinkProps> = PatchComponent(
  "TagLink",
  ({
    tag,
    linkType = "scene",
    className,
    hoverPlacement,
    showHierarchyIcon = false,
    hierarchyTooltipID,
  }) => {
    const link = useMemo(() => {
      switch (linkType) {
        case "scene":
          return NavUtils.makeTagScenesUrl(tag);
        case "performer":
          return NavUtils.makeTagPerformersUrl(tag);
        case "studio":
          return NavUtils.makeTagStudiosUrl(tag);
        case "gallery":
          return NavUtils.makeTagGalleriesUrl(tag);
        case "image":
          return NavUtils.makeTagImagesUrl(tag);
        case "group":
          return NavUtils.makeTagGroupsUrl(tag);
        case "scene_marker":
          return NavUtils.makeTagSceneMarkersUrl(tag);
        case "details":
          return NavUtils.makeTagUrl(tag.id ?? "");
      }
    }, [tag, linkType]);

    const title = tag.name || "";

    const tooltip = useMemo(() => {
      if (!hierarchyTooltipID) {
        return <></>;
      }

      return (
        <Tooltip id="tag-hierarchy-tooltip">
          <FormattedMessage id={hierarchyTooltipID} />
        </Tooltip>
      );
    }, [hierarchyTooltipID]);

    const tagStyle = tag.color ? {
      backgroundColor: tag.color
    } : undefined;

    return (
      <SortNameLinkComponent
        sortName={tag.sort_name || title}
        link={link}
        className={className}
        style={tagStyle}
      >
        <TagPopover id={tag.id ?? ""} placement={hoverPlacement}>
          <span style={tag.color ? { color: getContrastColor(tag.color) } : undefined}>
            {title}
          </span>
          {showHierarchyIcon && (
            <OverlayTrigger placement="top" overlay={tooltip}>
              <span className="icon-wrapper" style={tag.color ? { color: getContrastColor(tag.color) } : undefined}>
                <span className="vertical-line">|</span>
                <Icon icon={faFolderTree} className="tag-icon" />
              </span>
            </OverlayTrigger>
          )}
        </TagPopover>
      </SortNameLinkComponent>
    );
  }
);
