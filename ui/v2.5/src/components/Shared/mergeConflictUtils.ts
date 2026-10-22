import {
  ScrapeResult,
  ZeroableScrapeResult,
} from "src/components/Shared/ScrapeDialog/scrapeResult";

export function isMergeValueEmpty(v: unknown): boolean {
  if (v === null || v === undefined) return true;
  if (typeof v === "string") return v.trim() === "";
  if (Array.isArray(v)) return v.length === 0;
  return false;
}

export function isDefaultImagePath(path?: string | null): boolean {
  return !!path && path.includes("default=true");
}

export function hasRealImagePath(path?: string | null): boolean {
  return !isMergeValueEmpty(path) && !isDefaultImagePath(path);
}

export function scalarMergeConflict(
  dest: string | null | undefined,
  merged: string | null | undefined
): boolean {
  if (isMergeValueEmpty(dest) || isMergeValueEmpty(merged)) return false;
  return dest!.trim() !== merged!.trim();
}

export function imageMergeConflict(
  dest: string | null | undefined,
  merged: string | null | undefined
): boolean {
  const destReal = hasRealImagePath(dest);
  const mergedReal = hasRealImagePath(merged);
  if (!destReal || !mergedReal) return false;
  return dest !== merged;
}

export function mergeImagePathScrapeResult(
  destPath?: string | null,
  mergedPath?: string | null
): ScrapeResult<string> {
  const destReal = hasRealImagePath(destPath);
  const mergedReal = hasRealImagePath(mergedPath);

  if (destReal && mergedReal && destPath !== mergedPath) {
    return new ScrapeResult(
      destPath ?? undefined,
      mergedPath ?? undefined,
      false
    );
  }

  const useNew = !destReal && mergedReal;
  return new ScrapeResult(
    destReal ? destPath ?? undefined : undefined,
    mergedReal ? mergedPath ?? undefined : undefined,
    useNew
  );
}

export function ratingMergeConflict(
  dest: number | null | undefined,
  merged: number | null | undefined
): boolean {
  if (dest == null || merged == null) return false;
  return dest !== merged;
}

export function booleanMergeConflict(
  dest: boolean,
  merged: boolean
): boolean {
  if (!dest && !merged) return false;
  if (dest === merged) return false;
  return dest && merged;
}

export function mergeFieldScrapeResult(
  dest: string | null | undefined,
  merged: string | null | undefined
): ScrapeResult<string> {
  if (scalarMergeConflict(dest, merged)) {
    return new ScrapeResult(dest ?? undefined, merged ?? undefined, false);
  }

  const useNew = isMergeValueEmpty(dest) && !isMergeValueEmpty(merged);
  return new ScrapeResult(dest ?? undefined, merged ?? undefined, useNew);
}

export function mergeRatingScrapeResult(
  dest: number | null | undefined,
  merged: number | null | undefined
): ZeroableScrapeResult<number> {
  if (ratingMergeConflict(dest, merged)) {
    return new ZeroableScrapeResult(dest, merged, false);
  }

  const useNew = dest == null && merged != null;
  return new ZeroableScrapeResult(dest, merged, useNew);
}

export function mergeFavoriteScrapeResult(
  dest: boolean,
  merged: boolean
): ZeroableScrapeResult<boolean> {
  if (booleanMergeConflict(dest, merged)) {
    return new ZeroableScrapeResult(dest, merged, false);
  }

  const useNew = !dest && merged;
  return new ZeroableScrapeResult(dest, merged, useNew);
}
