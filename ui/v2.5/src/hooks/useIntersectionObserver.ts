import { useEffect, useRef, useState } from 'react';

interface UseIntersectionObserverOptions {
  threshold?: number | number[];
  rootMargin?: string;
  root?: Element | null;
}

export function useIntersectionObserver(
  options: UseIntersectionObserverOptions = {}
) {
  const [entries, setEntries] = useState<IntersectionObserverEntry[]>([]);
  const [activeIndex, setActiveIndex] = useState<number>(0);
  const observerRef = useRef<IntersectionObserver | null>(null);
  const elementsRef = useRef<Element[]>([]);

  const { threshold = 0.5, rootMargin = '0px', root = null } = options;

  useEffect(() => {
    observerRef.current = new IntersectionObserver(
      (observedEntries) => {
        setEntries(observedEntries);
        
        let maxIntersectionRatio = 0;
        let maxIndex = 0;
        
        observedEntries.forEach((entry, index) => {
          if (entry.isIntersecting && entry.intersectionRatio > maxIntersectionRatio) {
            maxIntersectionRatio = entry.intersectionRatio;
            maxIndex = elementsRef.current.indexOf(entry.target);
          }
        });
        
        if (maxIntersectionRatio > 0) {
          setActiveIndex(maxIndex);
        }
      },
      {
        threshold,
        rootMargin,
        root,
      }
    );

    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect();
      }
    };
  }, [threshold, rootMargin, root]);

  const observe = (element: Element | null) => {
    if (element && observerRef.current) {
      observerRef.current.observe(element);
      elementsRef.current.push(element);
    }
  };

  const unobserve = (element: Element | null) => {
    if (element && observerRef.current) {
      observerRef.current.unobserve(element);
      const index = elementsRef.current.indexOf(element);
      if (index > -1) {
        elementsRef.current.splice(index, 1);
      }
    }
  };

  const disconnect = () => {
    if (observerRef.current) {
      observerRef.current.disconnect();
      elementsRef.current = [];
    }
  };

  return {
    entries,
    activeIndex,
    observe,
    unobserve,
    disconnect,
  };
}
