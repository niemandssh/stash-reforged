import React from "react";

export function useModal() {
  const [modal, setModal] = React.useState<React.ReactNode>();

  const closeModal = () => setModal(undefined);
  const showModal = (m: React.ReactNode) => setModal(m);

  return { modal, closeModal, showModal };
}

function shouldSkipConfirmOnEnter(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  if (target.isContentEditable) {
    return true;
  }

  const tag = target.tagName;
  if (tag === "TEXTAREA" || tag === "SELECT") {
    return true;
  }
  if (tag === "INPUT") {
    const type = (target as HTMLInputElement).type;
    return !["checkbox", "radio", "button", "submit", "reset"].includes(type);
  }

  return false;
}

/** Triggers onConfirm when Enter is pressed outside text fields. */
export function useConfirmOnEnter(
  enabled: boolean,
  onConfirm: (() => void) | undefined,
  disabled?: boolean
) {
  React.useEffect(() => {
    if (!enabled || !onConfirm || disabled) {
      return;
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key !== "Enter") {
        return;
      }
      if (shouldSkipConfirmOnEnter(event.target)) {
        return;
      }
      if (event.target instanceof HTMLButtonElement) {
        return;
      }

      event.preventDefault();
      onConfirm();
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [enabled, onConfirm, disabled]);
}
