import React from "react";
import { Badge } from "./Badge";

interface IProbablyBrokenBadgeProps {
  className?: string;
}

export const ProbablyBrokenBadge: React.FC<IProbablyBrokenBadgeProps> = ({ className = "" }) => {
  return (
    <Badge
      className={`probably-broken-badge ${className}`}
    >
      BROKEN?
    </Badge>
  );
};

export default ProbablyBrokenBadge;
