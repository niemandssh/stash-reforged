import React from "react";
import { Badge } from "./Badge";

interface IBrokenBadgeProps {
  className?: string;
}

export const BrokenBadge: React.FC<IBrokenBadgeProps> = ({
  className = "",
}) => {
  return (
    <Badge
      variant="danger"
      className={`broken-badge ${className}`}
      style={{
        backgroundColor: "#dc3545",
        color: "white",
        fontSize: "0.7em",
        fontWeight: "bold",
        padding: "2px 6px",
        borderRadius: "3px",
        textTransform: "uppercase",
        letterSpacing: "0.5px",
      }}
    >
      BROKEN
    </Badge>
  );
};

export default BrokenBadge;
