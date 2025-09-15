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
  withoutContext?: boolean;
}

export const RatingNumber = PatchComponent(
  "RatingNumber",
  (props: IRatingNumberProps) => {
    const [hoverRating, setHoverRating] = useState<number | undefined>();
    const [, setHoverIsHalf] = useState<boolean>(false);
    const disabled = props.disabled || !props.onSetRating;

    const rating = props.value ? props.value / 10 : 0;

    const max = 20;

    function setRating(starIndex: number) {
      if (!props.onSetRating) {
        return;
      }

      let newRating: number;
      
      if (starIndex % 2 === 0) {
        newRating = starIndex * 0.5;
      } else {
        newRating = starIndex * 0.5;
      }

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
        setHoverIsHalf(false);
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
        const hoverValue = hoverRating * 0.5;
        
        if (starIndex % 2 === 0) {
          const pairIndex = starIndex / 2;
          if (hoverValue >= pairIndex) {
            return 100;
          } else {
            return 0;
          }
        } else {
          const pairIndex = (starIndex + 1) / 2;
          if (hoverValue >= pairIndex - 0.5) {
            return 100;
          } else {
            return 0;
          }
        }
      } else {
        if (starIndex % 2 === 0) {
          const pairIndex = starIndex / 2;
          if (rating >= pairIndex) {
            return 100;
          } else {
            return 0;
          }
        } else {
          const pairIndex = (starIndex + 1) / 2;
          if (rating >= pairIndex - 0.5) {
            return 100;
          } else {
            return 0;
          }
        }
      }
    }

    function getStarColorClass(starIndex: number) {
      if (hoverRating !== undefined) {
        if (starIndex <= hoverRating) {
          return 'star-color-gold';
        } else {
          return 'star-color-white';
        }
      } else {
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
