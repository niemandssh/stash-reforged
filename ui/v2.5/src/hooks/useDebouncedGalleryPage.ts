import { useCallback, useRef, useEffect } from "react";
import { useGalleryLastOpenedPage } from "./useGalleryLastOpenedPage";

interface IUseDebouncedGalleryPageOptions {
  galleryId: string;
  delay?: number;
  initialPage?: number;
}

export const useDebouncedGalleryPage = ({
  galleryId,
  delay = 1000,
  initialPage = 1,
}: IUseDebouncedGalleryPageOptions) => {
  const { updatePage } = useGalleryLastOpenedPage(galleryId);
  const timeoutRef = useRef<number | null>(null);
  const lastPageRef = useRef<number>(initialPage);
  const isInitializedRef = useRef<boolean>(false);

  // Debounced update function
  const debouncedUpdatePage = useCallback(
    (page: number) => {
      // Don't update if no galleryId, same page, or not initialized yet
      if (
        !galleryId ||
        page === lastPageRef.current ||
        !isInitializedRef.current
      ) {
        return;
      }

      // Clear existing timeout
      if (timeoutRef.current) {
        window.clearTimeout(timeoutRef.current);
      }

      // Set new timeout
      timeoutRef.current = window.setTimeout(() => {
        updatePage(page);
        lastPageRef.current = page;
      }, delay);
    },
    [updatePage, delay, galleryId]
  );

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        window.clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  // Initialize with a delay to avoid updating immediately after restoration
  useEffect(() => {
    if (!isInitializedRef.current) {
      const initTimeout = window.setTimeout(() => {
        isInitializedRef.current = true;
      }, 2000); // 2 second delay before we start tracking changes

      return () => window.clearTimeout(initTimeout);
    }
  }, []);

  // Function to set the current page without triggering update (for restoration)
  const setCurrentPage = useCallback((page: number) => {
    lastPageRef.current = page;
  }, []);

  return {
    updatePage: debouncedUpdatePage,
    setCurrentPage,
    isInitialized: isInitializedRef.current,
  };
};
