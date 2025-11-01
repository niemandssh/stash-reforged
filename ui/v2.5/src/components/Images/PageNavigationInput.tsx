import React, { useState, useCallback, KeyboardEvent, useRef, useEffect } from "react";
import { Form, Button } from "react-bootstrap";
import { Icon } from "src/components/Shared/Icon";
import { faArrowRight, faChevronDown } from "@fortawesome/free-solid-svg-icons";

interface IPageNavigationInputProps {
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  className?: string;
}

export const PageNavigationInput: React.FC<IPageNavigationInputProps> = ({
  currentPage,
  totalPages,
  onPageChange,
  className = "",
}) => {
  const [inputValue, setInputValue] = useState<string>("");
  const [isVisible, setIsVisible] = useState<boolean>(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const { value } = e.target;
    // Разрешаем только цифры
    if (value === "" || /^\d+$/.test(value)) {
      setInputValue(value);
    }
  }, []);

  const handleSubmit = useCallback(() => {
    const pageNumber = parseInt(inputValue, 10);
    if (pageNumber >= 1 && pageNumber <= totalPages) {
      onPageChange(pageNumber - 1); // Конвертируем в 0-based индекс
      setInputValue("");
      setIsVisible(false);
    }
  }, [inputValue, totalPages, onPageChange]);

  const handleKeyPress = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      handleSubmit();
    } else if (e.key === "Escape") {
      setInputValue("");
      setIsVisible(false);
    }
  }, [handleSubmit]);

  const toggleVisibility = useCallback(() => {
    setIsVisible(!isVisible);
    if (!isVisible) {
      setInputValue("");
    }
  }, [isVisible]);

  // Закрытие дропдауна при клике вне его
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsVisible(false);
        setInputValue("");
      }
    };

    if (isVisible) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isVisible]);

  if (totalPages <= 1) {
    return (
      <div className={`page-counter-static ${className}`}>
        {currentPage} / {totalPages}
      </div>
    );
  }

  return (
    <div className={`page-navigation-dropdown ${className}`} ref={containerRef}>
      <div
        className="page-counter-clickable"
        onClick={toggleVisibility}
        title="Кликните для перехода к странице"
      >
        {currentPage} / {totalPages}
        <Icon icon={faChevronDown} className="dropdown-arrow" />
      </div>
      
      {isVisible && (
        <div className="page-navigation-dropdown-content">
          <div className="page-navigation-form">
            <Form.Control
              type="number"
              value={inputValue}
              onChange={handleInputChange}
              onKeyDown={handleKeyPress}
              placeholder={`1-${totalPages}`}
              className="page-navigation-input-field"
              autoFocus
              size="sm"
              min="1"
              max={totalPages}
            />
            <Button
              variant="primary"
              size="sm"
              onClick={handleSubmit}
              disabled={!inputValue || parseInt(inputValue, 10) < 1 || parseInt(inputValue, 10) > totalPages}
              className="page-navigation-submit"
              title="Перейти"
            >
              <Icon icon={faArrowRight} />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};
