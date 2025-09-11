import React from "react";
import { Badge } from "./Badge";

interface ProbablyBrokenBadgeProps {
  className?: string;
}

export const ProbablyBrokenBadge: React.FC<ProbablyBrokenBadgeProps> = ({ className = "" }) => {
  return (
    <Badge
      className={`probably-broken-badge ${className}`}
    >
      BROKEN?
    </Badge>
  );
};

export default ProbablyBrokenBadge;
