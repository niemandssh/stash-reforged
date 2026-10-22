import React, { useCallback, useEffect, useRef, useState } from "react";
import ReactDOM from "react-dom";
import { useIntl } from "react-intl";
import { faCloud } from "@fortawesome/free-solid-svg-icons";
import { Icon } from "src/components/Shared/Icon";
import { mutateCreateScene } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";

export type RemoteSceneSuggestionContextMenuProps = {
  sourceUrl: string;
  sourceTitle?: string;
  children: React.ReactNode;
};

type MenuPosition = { x: number; y: number };

const RemoteSceneSuggestionContextMenu: React.FC<
  RemoteSceneSuggestionContextMenuProps
> = ({ sourceUrl, sourceTitle, children }) => {
  const intl = useIntl();
  const Toast = useToast();

  const [position, setPosition] = useState<MenuPosition | null>(null);
  const [creating, setCreating] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const closeMenu = useCallback(() => setPosition(null), []);

  const createRemoteScene = useCallback(
    async (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();

      if (!sourceUrl || creating) {
        closeMenu();
        return;
      }

      setCreating(true);
      try {
        const title = sourceTitle?.trim() || "Untitled";
        const result = await mutateCreateScene({
          title,
          embed_url: sourceUrl,
          // Same as SceneEditPanel: copy embed into urls so scrape-after-create works.
          urls: [sourceUrl],
        });

        const sceneId = result.data?.sceneCreate?.id;
        if (!sceneId) {
          throw new Error("Scene was not created");
        }

        Toast.success(
          intl.formatMessage(
            { id: "toast.created_entity" },
            {
              entity: intl
                .formatMessage({ id: "scene" })
                .toLocaleLowerCase(),
            }
          )
        );

        closeMenu();
        const editUrl = `/scenes/${sceneId}?autoEdit=true&scrapeUrl=${encodeURIComponent(
          sourceUrl
        )}`;
        window.open(editUrl, "_blank", "noopener,noreferrer");
      } catch (err) {
        Toast.error(err);
        closeMenu();
      } finally {
        setCreating(false);
      }
    },
    [closeMenu, creating, intl, sourceTitle, sourceUrl, Toast]
  );

  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setPosition({ x: e.clientX, y: e.clientY });
  }, []);

  useEffect(() => {
    if (!position) return;

    const handlePointerDown = (e: MouseEvent) => {
      // Ignore the right-button release that follows contextmenu.
      if (e.button === 2) return;
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        closeMenu();
      }
    };

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") closeMenu();
    };

    const handleScroll = () => closeMenu();

    // Defer so the opening contextmenu/mousedown sequence doesn't close us.
    const timer = window.setTimeout(() => {
      document.addEventListener("mousedown", handlePointerDown, true);
      document.addEventListener("keydown", handleEscape);
      window.addEventListener("scroll", handleScroll, true);
    }, 0);

    return () => {
      window.clearTimeout(timer);
      document.removeEventListener("mousedown", handlePointerDown, true);
      document.removeEventListener("keydown", handleEscape);
      window.removeEventListener("scroll", handleScroll, true);
    };
  }, [closeMenu, position]);

  return (
    <>
      <div
        className="remote-scene-suggestion-context"
        onContextMenuCapture={handleContextMenu}
        onMouseDownCapture={(e) => {
          // Don't let slick start a drag on right-click.
          if (e.button === 2) {
            e.stopPropagation();
          }
        }}
      >
        {children}
      </div>
      {position &&
        ReactDOM.createPortal(
          <div
            ref={menuRef}
            className="scene-context-menu dropdown-menu show bg-secondary text-white"
            style={{
              position: "fixed",
              left: position.x,
              top: position.y,
              zIndex: 2000,
            }}
            onContextMenu={(e) => {
              e.preventDefault();
              e.stopPropagation();
            }}
          >
            <button
              type="button"
              className="dropdown-item bg-secondary text-white d-flex align-items-center"
              disabled={creating || !sourceUrl}
              onClick={createRemoteScene}
            >
              <Icon icon={faCloud} className="mr-2" />
              {creating ? "Creating…" : "Add to remote scenes"}
            </button>
          </div>,
          document.body
        )}
    </>
  );
};

export default RemoteSceneSuggestionContextMenu;
