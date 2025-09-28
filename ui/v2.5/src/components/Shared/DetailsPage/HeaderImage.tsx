import { PropsWithChildren } from "react";
import { PatchComponent } from "src/patch";

export const HeaderImage: React.FC<
  PropsWithChildren<{
    hasImages?: boolean;
  }>
> = PatchComponent("HeaderImage", ({ children, hasImages = true }) => {
  return (
    <div className={`detail-header-image ${!hasImages ? 'no-images' : ''}`}>
      {children}
    </div>
  );
});
