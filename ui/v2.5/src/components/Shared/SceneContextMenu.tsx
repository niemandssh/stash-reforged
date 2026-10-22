import React, { useCallback, useEffect, useRef, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import {
  faClock,
  faExternalLinkAlt,
  faTrash,
} from "@fortawesome/free-solid-svg-icons";
import { Icon } from "./Icon";
import { ModalComponent } from "./Modal";
import {
  useSceneAddWatchLater,
  useSceneRemoveWatchLater,
} from "src/core/StashService";
import "./SceneContextMenu.scss";

type SceneContextMenuVariant = "browse" | "watch-later";

interface ISceneContextMenuProps {
  sceneId: string;
  sceneUrl: string;
  watchLater?: boolean;
  variant?: SceneContextMenuVariant;
  onRemoved?: () => void;
  children: React.ReactNode;
}

interface IMenuPosition {
  x: number;
  y: number;
}

export const SceneContextMenu: React.FC<ISceneContextMenuProps> = ({
  sceneId,
  sceneUrl,
  watchLater = false,
  variant = "browse",
  onRemoved,
  children,
}) => {
  const intl = useIntl();
  const [position, setPosition] = useState<IMenuPosition | null>(null);
  const [showRemoveConfirm, setShowRemoveConfirm] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const [addWatchLater] = useSceneAddWatchLater();
  const [removeWatchLater] = useSceneRemoveWatchLater();

  const closeMenu = useCallback(() => {
    setPosition(null);
  }, []);

  useEffect(() => {
    if (!position) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        closeMenu();
      }
    };

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        closeMenu();
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleEscape);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleEscape);
    };
  }, [position, closeMenu]);

  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setPosition({ x: e.clientX, y: e.clientY });
  };

  const toggleWatchLater = async () => {
    if (watchLater) {
      await removeWatchLater({ variables: { id: sceneId } });
    } else {
      await addWatchLater({ variables: { id: sceneId } });
    }
    closeMenu();
  };

  const openInNewTab = () => {
    window.open(sceneUrl, "_blank", "noopener,noreferrer");
    closeMenu();
  };

  const requestRemoveFromList = () => {
    closeMenu();
    setShowRemoveConfirm(true);
  };

  const confirmRemoveFromList = async () => {
    await removeWatchLater({ variables: { id: sceneId } });
    setShowRemoveConfirm(false);
    onRemoved?.();
  };

  return (
    <>
      <div
        className="scene-context-menu-wrapper"
        onContextMenu={handleContextMenu}
      >
        {children}
        {position && (
          <div
            ref={menuRef}
            className="scene-context-menu dropdown-menu show bg-secondary text-white"
            style={{
              position: "fixed",
              left: position.x,
              top: position.y,
              zIndex: 1050,
            }}
          >
            {variant === "browse" && (
              <button
                type="button"
                className="dropdown-item bg-secondary text-white d-flex align-items-center"
                onClick={toggleWatchLater}
              >
                <Icon icon={faClock} className="mr-2" />
                <FormattedMessage
                  id={
                    watchLater
                      ? "actions.remove_from_watch_later"
                      : "actions.add_to_watch_later"
                  }
                />
              </button>
            )}
            {variant === "watch-later" && (
              <button
                type="button"
                className="dropdown-item bg-secondary text-white d-flex align-items-center"
                onClick={requestRemoveFromList}
              >
                <Icon icon={faTrash} className="mr-2" />
                <FormattedMessage id="actions.remove_from_list" />
              </button>
            )}
            <button
              type="button"
              className="dropdown-item bg-secondary text-white d-flex align-items-center"
              onClick={openInNewTab}
            >
              <Icon icon={faExternalLinkAlt} className="mr-2" />
              <FormattedMessage id="actions.open_in_new_tab" />
            </button>
          </div>
        )}
      </div>

      {showRemoveConfirm && (
        <ModalComponent
          show
          icon={faTrash}
          header={intl.formatMessage({ id: "actions.remove_from_list" })}
          accept={{
            variant: "danger",
            onClick: confirmRemoveFromList,
            text: intl.formatMessage({ id: "actions.confirm" }),
          }}
          cancel={{
            onClick: () => setShowRemoveConfirm(false),
            text: intl.formatMessage({ id: "actions.cancel" }),
            variant: "secondary",
          }}
        >
          <FormattedMessage id="dialogs.watch_later.remove_confirm" />
        </ModalComponent>
      )}
    </>
  );
};
