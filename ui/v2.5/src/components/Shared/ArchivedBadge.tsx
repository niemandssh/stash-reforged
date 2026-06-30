import React from "react";
import { Badge } from "./Badge";

interface IArchivedBadgeProps {
  className?: string;
}

export const ArchivedBadge: React.FC<IArchivedBadgeProps> = ({
  className = "",
}) => {
  return (
    <Badge
      variant="secondary"
      className={`archived-badge ${className}`}
      style={{
        backgroundColor: "#6c757d",
        color: "white",
        fontSize: "0.7em",
        fontWeight: "bold",
        padding: "2px 6px",
        borderRadius: "3px",
        textTransform: "uppercase",
        letterSpacing: "0.5px",
      }}
    >
      ARCHIVED
    </Badge>
  );
};

export default ArchivedBadge;
