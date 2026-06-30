import React from "react";
import { Badge } from "./Badge";

interface ITrimmedBadgeProps {
  className?: string;
}

export const TrimmedBadge: React.FC<ITrimmedBadgeProps> = ({
  className = "",
}) => {
  return (
    <Badge
      variant="success"
      className={`trimmed-badge ${className}`}
      style={{
        backgroundColor: "#28a745",
        color: "white",
        fontSize: "0.7em",
        fontWeight: "bold",
        padding: "2px 6px",
        borderRadius: "3px",
        textTransform: "uppercase",
        letterSpacing: "0.5px",
      }}
    >
      TRIMMED
    </Badge>
  );
};

export default TrimmedBadge;
