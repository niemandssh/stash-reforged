import React from "react";
import { Badge as BootstrapBadge } from "react-bootstrap";

interface BadgeProps {
  variant?: "primary" | "secondary" | "success" | "danger" | "warning" | "info" | "light" | "dark";
  children: React.ReactNode;
  className?: string;
}

export const Badge: React.FC<BadgeProps> = ({ 
  variant = "primary", 
  children, 
  className = "" 
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
      }}
    >
      {children}
    </BootstrapBadge>
  );
};

export default Badge;
