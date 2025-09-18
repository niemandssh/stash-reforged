import { useCallback } from "react";
import { useGalleryUpdate } from "src/core/StashService";
import { DisplayMode } from "src/models/list-filter/types";
import { useToast } from "src/hooks/Toast";
import { useIntl } from "react-intl";

export function useGalleryDisplayMode(galleryId: string, currentDisplayMode: number | undefined) {
  const [updateGallery] = useGalleryUpdate();
  const Toast = useToast();
  const intl = useIntl();

  const displayMode = (currentDisplayMode ?? 0) as DisplayMode;

  const setDisplayMode = useCallback(async (newDisplayMode: DisplayMode) => {
    try {
      await updateGallery({
        variables: {
          input: {
            id: galleryId,
            display_mode: newDisplayMode,
          },
        },
      });
    } catch (e) {
      Toast.error(e);
    }
  }, [galleryId, updateGallery, Toast]);

  return {
    displayMode,
    setDisplayMode,
  };
}
