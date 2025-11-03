import { useRef } from "react";
import { useDirectory } from "src/core/StashService";

export const useDirectoryPaths = (path: string, hideError: boolean) => {
  const { data, loading, error } = useDirectory(path);
  const prevData = useRef<typeof data | undefined>(undefined);

  if (!loading) prevData.current = data;

  const currentData = loading ? prevData.current : data;
  const directories =
    error && hideError ? [] : currentData?.directory.directories;
  const files = error && hideError ? [] : currentData?.directory.files;
  const parent = currentData?.directory.parent;

  return { directories, files, parent, loading, error };
};
