import React from "react";
import { PatchComponent } from "src/patch";

interface IDeathRibbonProps {
  className?: string;
  size?: "small" | "large";
}

export const DeathRibbon: React.FC<IDeathRibbonProps> = PatchComponent(
  "DeathRibbon",
  ({ className = "", size = "small" }) => {
    return (
      <div className={`death-ribbon death-ribbon--${size} ${className}`}>
        <div className="death-ribbon__inner">
          <span className="death-ribbon__text">RIP</span>
        </div>
      </div>
    );
  }
);
