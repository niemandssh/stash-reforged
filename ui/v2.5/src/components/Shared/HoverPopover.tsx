import React, { useState, useCallback, useEffect, useRef } from "react";
import { Overlay, Popover, OverlayProps } from "react-bootstrap";
import { PatchComponent } from "src/patch";

interface IHoverPopover {
  enterDelay?: number;
  leaveDelay?: number;
  content: JSX.Element[] | JSX.Element | string;
  className?: string;
  popoverClassName?: string;
  placement?: OverlayProps["placement"];
  offset?: OverlayProps["offset"];
  onOpen?: () => void;
  onClose?: () => void;
  target?: React.RefObject<HTMLElement> | (() => HTMLElement | null);
}

export const HoverPopover: React.FC<IHoverPopover> = PatchComponent(
  "HoverPopover",
  ({
    enterDelay = 200,
    leaveDelay = 200,
    content,
    children,
    className,
    popoverClassName,
    placement = "top",
    offset,
    onOpen,
    onClose,
    target,
  }) => {
    const [show, setShow] = useState(false);
    const triggerRef = useRef<HTMLDivElement>(null);
    const enterTimer = useRef<number>();
    const leaveTimer = useRef<number>();

    const handleMouseEnter = useCallback(() => {
      window.clearTimeout(leaveTimer.current);
      enterTimer.current = window.setTimeout(() => {
        setShow(true);
        onOpen?.();
      }, enterDelay);
    }, [enterDelay, onOpen]);

    const handleMouseLeave = useCallback(() => {
      window.clearTimeout(enterTimer.current);
      leaveTimer.current = window.setTimeout(() => {
        setShow(false);
        onClose?.();
      }, leaveDelay);
    }, [leaveDelay, onClose]);

    useEffect(
      () => () => {
        window.clearTimeout(enterTimer.current);
        window.clearTimeout(leaveTimer.current);
      },
      []
    );

    return (
      <>
        <div
          className={className}
          onMouseEnter={handleMouseEnter}
          onMouseLeave={handleMouseLeave}
          ref={triggerRef}
        >
          {children}
        </div>
        {triggerRef.current && (typeof target === 'function' ? target() : (target?.current ?? triggerRef.current)) && (
          <Overlay
            show={show}
            placement={placement}
            offset={offset}
            container={document.body}
            target={typeof target === 'function' ? target() : (target?.current ?? triggerRef.current)}
          >
            <Popover
              onMouseEnter={handleMouseEnter}
              onMouseLeave={handleMouseLeave}
              id="popover"
              className={`hover-popover-content ${popoverClassName || ''}`}
            >
              {content}
            </Popover>
          </Overlay>
        )}
      </>
    );
  }
);
