import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

interface ITrimContextType {
  trimEnabled: boolean;
  setTrimEnabled: (enabled: boolean) => void;
}

const TrimContext = createContext<ITrimContextType | undefined>(undefined);

export const useTrimContext = () => {
  const context = useContext(TrimContext);
  if (context === undefined) {
    // Return default values instead of throwing error
    // This prevents issues with lazy-loaded components
    console.warn('useTrimContext used outside of TrimProvider, using default values');
    return {
      trimEnabled: true,
      setTrimEnabled: () => {}
    };
  }
  return context;
};

interface ITrimProviderProps {
  children: ReactNode;
}

const TRIM_ENABLED_KEY = 'stash_trim_enabled';

export const TrimProvider: React.FC<ITrimProviderProps> = ({ children }) => {
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
