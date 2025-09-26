import React, { PropsWithChildren } from "react";

export const DetailTitle: React.FC<
  PropsWithChildren<{
    name: string;
    disambiguation?: string;
    classNamePrefix: string;
    className?: string;
  }>
> = ({ name, disambiguation, classNamePrefix, className, children }) => {
  return (
    <h2 className={className}>
      <span className={`${classNamePrefix}-name`}>{name}</span>
      {disambiguation && (
        <span className={`${classNamePrefix}-disambiguation`}>
          {` (${disambiguation})`}
        </span>
      )}
      {children}
    </h2>
  );
};
