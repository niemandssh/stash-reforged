import React, { useState, useEffect, useRef, useCallback } from "react";
import { useIntl } from "react-intl";
import { Helmet } from "react-helmet";
import { LoadingIndicator } from "../Shared/LoadingIndicator";
import { useTitleProps } from "src/hooks/title";
import { useFindViewHistoryQuery } from "src/core/rest-hooks";
import { ViewHistoryCard } from "./ViewHistoryCard";
import { IViewHistoryEntry, IViewHistoryResult } from "./types";
import "./ViewHistory.scss";

const ITEMS_PER_PAGE = 50;

export const ViewHistory: React.FC = () => {
  const intl = useIntl();
  const titleProps = useTitleProps({ id: "view_history" });

  const [page, setPage] = useState(1);
  const [items, setItems] = useState<IViewHistoryEntry[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const loadingRef = useRef<HTMLDivElement>(null);

  const { data, loading } = useFindViewHistoryQuery({
    variables: {
      filter: {
        page,
        per_page: ITEMS_PER_PAGE,
      },
    },
  });

  // Update items when data changes
  React.useEffect(() => {
    if (data?.findViewHistory?.items) {
      setItems((prev) => {
        if (page === 1) {
          return (data.findViewHistory as any).items as any;
        }
        return [...prev, ...(data.findViewHistory as any).items] as any;
      });
      setHasMore(((data.findViewHistory as any).items as any).length === ITEMS_PER_PAGE);
    }
  }, [data, page]);

  const loadMore = useCallback(() => {
    if (!loading && hasMore) {
      setPage((prev) => prev + 1);
    }
  }, [loading, hasMore]);

  // Infinite scroll
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loading) {
          loadMore();
        }
      },
      {
        rootMargin: "200px",
      }
    );

    const currentRef = loadingRef.current;
    if (currentRef) {
      observer.observe(currentRef);
    }

    return () => {
      if (currentRef) {
        observer.unobserve(currentRef);
      }
    };
  }, [loadMore, hasMore, loading]);

  const totalViews = data?.findViewHistory?.count || 0;
  const totalOCount = (data?.findViewHistory as any)?.totalOCount || 0;
  const totalOMGCount = (data?.findViewHistory as any)?.totalOMGCount || 0;

  if (loading && page === 1) {
    return <LoadingIndicator />;
  }

  // Group items by date
  const groupedItems = items.reduce((acc, item) => {
    const date = new Date(item.viewDate).toLocaleDateString();
    if (!acc[date]) {
      acc[date] = [];
    }
    acc[date].push(item);
    return acc;
  }, {} as Record<string, IViewHistoryEntry[]>);

  return (
    <>
      <Helmet {...titleProps} />
      <div className="view-history-container">
        <div className="view-history-header">
          <h1>{intl.formatMessage({ id: "view_history" })}</h1>
          <div className="view-history-stats">
            <span className="view-history-total-views">
              {intl.formatMessage({ id: "total_views" }, { count: totalViews })}
            </span>
            {totalOCount > 0 && (
              <span className="view-history-total-o-count">
                {intl.formatMessage({ id: "o_count" })}: {totalOCount}
              </span>
            )}
            {totalOMGCount > 0 && (
              <span className="view-history-total-omg-count">
                {intl.formatMessage({ id: "omg_counter" })}: {totalOMGCount}
              </span>
            )}
          </div>
        </div>

        <div className="view-history-content">
          {Object.entries(groupedItems).map(([date, dateItems]) => (
            <div key={date} className="view-history-date-group">
              <h3 className="view-history-date-header">{date}</h3>
              <div className="view-history-items-list">
                {dateItems.map((item) => (
                  <ViewHistoryCard
                    key={`${item.scene?.id || item.gallery?.id}-${
                      item.viewDate
                    }`}
                    scene={item.scene}
                    gallery={item.gallery}
                    viewDate={item.viewDate}
                    oDate={item.oDate}
                    omgDate={item.omgDate}
                    viewCount={item.viewCount}
                  />
                ))}
              </div>
            </div>
          ))}

          {hasMore && (
            <div ref={loadingRef} className="view-history-loading-more">
              {loading && <LoadingIndicator inline />}
            </div>
          )}
        </div>
      </div>
    </>
  );
};

export default ViewHistory;
