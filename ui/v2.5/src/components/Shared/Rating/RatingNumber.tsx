import React, { useState } from "react";
import { Button } from "react-bootstrap";
import { Icon } from "../Icon";
import { faStar as fasStar } from "@fortawesome/free-solid-svg-icons";
import { faStar as farStar } from "@fortawesome/free-regular-svg-icons";
import { PatchComponent } from "src/patch";

export interface IRatingNumberProps {
  value: number | null;
  onSetRating?: (value: number | null) => void;
  disabled?: boolean;
  clickToRate?: boolean;
  // true if we should indicate that this is a rating
  withoutContext?: boolean;
}

export const RatingNumber = PatchComponent(
  "RatingNumber",
  (props: IRatingNumberProps) => {
    const [hoverRating, setHoverRating] = useState<number | undefined>();
    const [, setHoverIsHalf] = useState<boolean>(false);
    const disabled = props.disabled || !props.onSetRating;

    // Конвертируем десятибальный рейтинг (0-100) в звездочный (0-10)
    const rating = props.value ? props.value / 10 : 0;

    const max = 20; // 20 звезд, но визуально отображаются как 10 пар

    function setRating(starIndex: number) {
      if (!props.onSetRating) {
        return;
      }

      let newRating: number;
      
      if (starIndex % 2 === 0) {
        // Четные звезды (2, 4, 6...) - полный рейтинг
        newRating = starIndex * 0.5;
      } else {
        // Нечетные звезды (1, 3, 5...) - половинка рейтинга
        newRating = starIndex * 0.5;
      }

      // Конвертируем обратно в десятибальную систему (0-100)
      const decimalRating = Math.round(newRating * 10);
      props.onSetRating(decimalRating);
    }

    function onMouseOver(starIndex: number, event: React.MouseEvent) {
      if (!disabled) {
        const rect = event.currentTarget.getBoundingClientRect();
        const clickX = event.clientX - rect.left;
        const isHalf = clickX < rect.width / 2;
        
        setHoverRating(starIndex);
        setHoverIsHalf(isHalf);
      }
    }

    function onFocus(starIndex: number) {
      if (!disabled) {
        setHoverRating(starIndex);
        setHoverIsHalf(false); // По умолчанию полная звезда при фокусе
      }
    }

    function onMouseOut() {
      if (!disabled) {
        setHoverRating(undefined);
        setHoverIsHalf(false);
      }
    }

    function getStarFill(starIndex: number) {
      if (hoverRating !== undefined) {
        // При наведении показываем текущий рейтинг + потенциальный
        const hoverValue = hoverRating * 0.5; // Конвертируем в 0-10 шкалу
        
        if (starIndex % 2 === 0) {
          // Четные звезды (2, 4, 6...) - правая половина звезды
          const pairIndex = starIndex / 2; // Номер пары (1, 2, 3...)
          if (hoverValue >= pairIndex) {
            return 100; // Полная звезда
          } else {
            return 0; // Пустая звезда
          }
        } else {
          // Нечетные звезды (1, 3, 5...) - левая половина звезды
          const pairIndex = (starIndex + 1) / 2; // Номер пары (1, 2, 3...)
          if (hoverValue >= pairIndex - 0.5) {
            return 100; // Полная звезда
          } else {
            return 0; // Пустая звезда
          }
        }
      } else {
        // Обычное состояние - текущий рейтинг
        if (starIndex % 2 === 0) {
          // Четные звезды (2, 4, 6...) - правая половина звезды
          const pairIndex = starIndex / 2; // Номер пары (1, 2, 3...)
          if (rating >= pairIndex) {
            return 100; // Полная звезда
          } else {
            return 0; // Пустая звезда
          }
        } else {
          // Нечетные звезды (1, 3, 5...) - левая половина звезды
          const pairIndex = (starIndex + 1) / 2; // Номер пары (1, 2, 3...)
          if (rating >= pairIndex - 0.5) {
            return 100; // Полная звезда
          } else {
            return 0; // Пустая звезда
          }
        }
      }
    }

    function getStarColorClass(starIndex: number) {
      // При наведении - золотой для всех звезд до hoverRating, иначе - белый для примененного рейтинга
      if (hoverRating !== undefined) {
        if (starIndex <= hoverRating) {
          return 'star-color-gold';
        } else {
          return 'star-color-white';
        }
      } else {
        // Примененный рейтинг всегда белый
        return 'star-color-white';
      }
    }

    function getStarClassName(starIndex: number) {
      const fill = getStarFill(starIndex);
      const isEven = starIndex % 2 === 0;
      return `star-fill-${fill} ${isEven ? 'star-even' : 'star-odd'}`;
    }

    function handleStarClick(starIndex: number) {
      if (disabled) return;

      setRating(starIndex);
    }

    if (props.clickToRate && !props.disabled) {
      return (
        <div className="rating-stars rating-number-stars">
          {Array.from(Array(max)).map((_, index) => {
            const starIndex = index + 1;
            return (
              <Button
                key={`star-${starIndex}`}
                disabled={disabled}
                className={`minimal ${getStarClassName(starIndex)}`}
                onClick={() => handleStarClick(starIndex)}
                variant="secondary"
                onMouseEnter={(e) => onMouseOver(starIndex, e)}
                onMouseLeave={onMouseOut}
                onFocus={() => onFocus(starIndex)}
                onBlur={onMouseOut}
                title={`${starIndex} звезд${starIndex > 1 ? 'ы' : 'а'}`}
              >
                <div 
                  className={`filled-star ${getStarColorClass(starIndex)}`}
                >
                  <Icon icon={fasStar} className="set" />
                </div>
                <div className="unfilled-star">
                  <Icon icon={farStar} className="unset" />
                </div>
              </Button>
            );
          })}
          <span className={`star-rating-number ${getStarColorClass(1)}`}>
            {hoverRating !== undefined 
              ? (hoverRating * 0.5).toFixed(1)
              : (rating > 0 ? rating.toFixed(1) : '')
            }
          </span>
        </div>
      );
    } else {
      return (
        <div className="rating-number disabled">
          {props.withoutContext && <Icon icon={fasStar} />}
          <span>{rating > 0 ? rating.toFixed(1) : '0.0'}</span>
        </div>
      );
    }
  }
);
