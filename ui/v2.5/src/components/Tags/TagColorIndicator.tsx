import React from "react";
import * as GQL from "src/core/generated-graphql";

interface ITagColorIndicatorProps {
  tag: Pick<GQL.TagDataFragment, "color">;
  size?: "sm" | "md" | "lg";
  className?: string;
}

export const TagColorIndicator: React.FC<ITagColorIndicatorProps> = ({
  tag,
  size = "md",
  className = "",
}) => {
  // Если у тега нет цвета, не отображаем квадратик
  if (!tag.color) {
    return null;
  }

  const sizeClasses = {
    sm: "tag-color-indicator-sm",
    md: "tag-color-indicator-md", 
    lg: "tag-color-indicator-lg"
  };

  return (
    <span
      className={`tag-color-indicator ${sizeClasses[size]} ${className}`}
      style={{ backgroundColor: tag.color }}
      title={`Цвет тега: ${tag.color}`}
    />
  );
};
