import * as GQL from "src/core/generated-graphql";

// Matched studio from a compound scrape result (e.g. TeamSkeet when primary is a new LittleAsians proposal).
export function scrapedStudioForSelect(
  studio?: GQL.ScrapedStudio | null
): GQL.ScrapedStudio | undefined {
  if (!studio) {
    return undefined;
  }

  if (studio.stored_id) {
    return studio;
  }

  if (studio.parent?.stored_id) {
    return studio.parent;
  }

  return undefined;
}

// Primary studio proposal when the scraper suggests creating a new studio.
export function scrapedStudioForCreate(
  studio?: GQL.ScrapedStudio | null
): GQL.ScrapedStudio | undefined {
  if (!studio || studio.stored_id) {
    return undefined;
  }

  return studio;
}
