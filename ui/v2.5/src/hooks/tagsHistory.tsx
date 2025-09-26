import { useEffect, useState, useCallback } from 'react';
import { Tag } from 'src/components/Tags/TagSelect';

interface TagsHistoryState {
  tags: Tag[];
  timestamp: number;
}

const MAX_HISTORY_SIZE = 100;
const STORAGE_KEY_PREFIX = 'scene-tags-history-';

export function useTagsHistory(sceneId: string | undefined) {
  const [history, setHistory] = useState<TagsHistoryState[]>([]);
  const [currentIndex, setCurrentIndex] = useState(-1);

  const storageKey = sceneId ? `${STORAGE_KEY_PREFIX}${sceneId}` : null;

  useEffect(() => {
    if (!storageKey) return;

    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) {
        const parsedHistory: TagsHistoryState[] = JSON.parse(stored);
        setHistory(parsedHistory);
        setCurrentIndex(parsedHistory.length - 1);
      }
    } catch (error) {
      console.error('Error loading tags history from localStorage:', error);
    }
  }, [storageKey]);

  const saveToStorage = useCallback((newHistory: TagsHistoryState[]) => {
    if (!storageKey) return;

    try {
      localStorage.setItem(storageKey, JSON.stringify(newHistory));
    } catch (error) {
      console.error('Error saving tags history to localStorage:', error);
    }
  }, [storageKey]);

  const addToHistory = useCallback((tags: Tag[]) => {
    const newState: TagsHistoryState = {
      tags: [...tags],
      timestamp: Date.now()
    };

    setHistory(prevHistory => {
      const newHistory = prevHistory.slice(0, currentIndex + 1);

      newHistory.push(newState);

      if (newHistory.length > MAX_HISTORY_SIZE) {
        newHistory.shift();
      }

      const newIndex = newHistory.length - 1;
      setCurrentIndex(newIndex);

      saveToStorage(newHistory);

      return newHistory;
    });
  }, [currentIndex, saveToStorage]);

  const undo = useCallback(() => {
    if (currentIndex > 0) {
      const newIndex = currentIndex - 1;
      setCurrentIndex(newIndex);
      return history[newIndex].tags;
    }
    return null;
  }, [currentIndex, history]);

  const redo = useCallback(() => {
    if (currentIndex < history.length - 1) {
      const newIndex = currentIndex + 1;
      setCurrentIndex(newIndex);
      return history[newIndex].tags;
    }
    return null;
  }, [currentIndex, history]);

  const clearHistory = useCallback(() => {
    setHistory([]);
    setCurrentIndex(-1);
    if (storageKey) {
      localStorage.removeItem(storageKey);
    }
  }, [storageKey]);

  const canUndo = currentIndex > 0;
  const canRedo = currentIndex < history.length - 1;

  return {
    addToHistory,
    undo,
    redo,
    clearHistory,
    canUndo,
    canRedo,
    currentTags: currentIndex >= 0 ? history[currentIndex]?.tags : null
  };
}
