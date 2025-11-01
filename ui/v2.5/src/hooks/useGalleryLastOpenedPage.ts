import { useCallback } from "react";
import LocalForage from "localforage";

const galleryPageStore = LocalForage.createInstance({
  name: "galleryLastOpenedPage",
  storeName: "galleryPages",
});

interface IUseGalleryLastOpenedPageReturn {
  updatePage: (page: number) => Promise<void>;
  getPage: (galleryId: string) => Promise<number | null>;
}

export const useGalleryLastOpenedPage = (
  galleryId: string
): IUseGalleryLastOpenedPageReturn => {
  const updatePage = useCallback(
    async (page: number) => {
      if (!galleryId) return;
      await galleryPageStore.setItem(galleryId, page);
    },
    [galleryId]
  );

  const getPage = useCallback(async (): Promise<number | null> => {
    if (!galleryId) return null;
    const page = await galleryPageStore.getItem<number>(galleryId);
    return page;
  }, [galleryId]);

  return {
    updatePage,
    getPage,
  };
};
