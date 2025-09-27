import React from "react";
import { Badge } from "./Badge";

interface IHLSBadgeProps {
  className?: string;
}

export const HLSBadge: React.FC<IHLSBadgeProps> = ({ className = "" }) => {
  return (
    <Badge
      className={`hls-badge ${className}`}
      style={{
        backgroundColor: "#ffc107",
        color: "black",
        fontSize: "0.7em",
        fontWeight: "bold",
        padding: "2px 6px",
        borderRadius: "3px",
        textTransform: "uppercase",
        letterSpacing: "0.5px",
      }}
    >
      HLS
    </Badge>
  );
};

export default HLSBadge;
