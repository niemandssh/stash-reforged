import React from "react";
import { Badge as BootstrapBadge } from "react-bootstrap";

interface BadgeProps {
  variant?: "primary" | "secondary" | "success" | "danger" | "warning" | "info" | "light" | "dark";
  children: React.ReactNode;
  className?: string;
  style?: React.CSSProperties;
}

export const Badge: React.FC<BadgeProps> = ({ 
  variant = "primary", 
  children, 
  className = "",
  style
}) => {
  return (
    <BootstrapBadge 
      variant={variant} 
      className={`badge-custom ${className}`}
      style={{
        fontSize: "0.75em",
        fontWeight: "bold",
        padding: "4px 8px",
        borderRadius: "4px",
        ...style,
      }}
    >
      {children}
    </BootstrapBadge>
  );
};

export default Badge;
