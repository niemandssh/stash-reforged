import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

interface TrimContextType {
  trimEnabled: boolean;
  setTrimEnabled: (enabled: boolean) => void;
}

const TrimContext = createContext<TrimContextType | undefined>(undefined);

export const useTrimContext = () => {
  const context = useContext(TrimContext);
  if (context === undefined) {
    throw new Error('useTrimContext must be used within a TrimProvider');
  }
  return context;
};

interface TrimProviderProps {
  children: ReactNode;
}

const TRIM_ENABLED_KEY = 'stash_trim_enabled';

export const TrimProvider: React.FC<TrimProviderProps> = ({ children }) => {
  // Initialize from localStorage or default to true
  const [trimEnabled, setTrimEnabledState] = useState(() => {
    try {
      const saved = localStorage.getItem(TRIM_ENABLED_KEY);
      return saved !== null ? JSON.parse(saved) : true;
    } catch (error) {
      console.warn('Failed to load trim state from localStorage:', error);
      return true;
    }
  });

  // Save to localStorage whenever trimEnabled changes
  useEffect(() => {
    try {
      localStorage.setItem(TRIM_ENABLED_KEY, JSON.stringify(trimEnabled));
    } catch (error) {
      console.warn('Failed to save trim state to localStorage:', error);
    }
  }, [trimEnabled]);

  const setTrimEnabled = (enabled: boolean) => {
    setTrimEnabledState(enabled);
  };

  return (
    <TrimContext.Provider value={{ trimEnabled, setTrimEnabled }}>
      {children}
    </TrimContext.Provider>
  );
};
