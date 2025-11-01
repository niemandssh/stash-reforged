import { useState, useEffect } from "react";
import { useConfigureUISetting, useConfiguration } from "src/core/StashService";

export interface INotesData {
  content: string;
  lastModified: string;
}

export function useNotes() {
  const { data } = useConfiguration();
  const [updateUIConfig] = useConfigureUISetting();
  const [notes, setNotes] = useState<string>("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (data?.configuration?.ui?.notes) {
      setNotes(data.configuration.ui.notes);
    }
  }, [data?.configuration?.ui?.notes]);

  const saveNotes = async (content: string) => {
    setIsLoading(true);
    setError(null);

    try {
      await updateUIConfig({
        variables: {
          key: "notes",
          value: content,
        },
      });

      setNotes(content);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error saving notes");
    } finally {
      setIsLoading(false);
    }
  };

  const clearNotes = () => {
    saveNotes("");
  };

  return {
    notes,
    saveNotes,
    clearNotes,
    isLoading,
    error,
  };
}
