import React from "react";
import { Badge } from "./Badge";

interface IRemoteBadgeProps {
  className?: string;
}

export const RemoteBadge: React.FC<IRemoteBadgeProps> = ({
  className = "",
}) => {
  return (
    <Badge
      variant="info"
      className={`remote-badge ${className}`}
      style={{
        backgroundColor: "#17a2b8",
        color: "white",
        fontSize: "0.7em",
        fontWeight: "bold",
        padding: "2px 6px",
        borderRadius: "3px",
        textTransform: "uppercase",
        letterSpacing: "0.5px",
      }}
    >
      REMOTE
    </Badge>
  );
};

export default RemoteBadge;
