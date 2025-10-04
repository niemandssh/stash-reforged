import React, { useState, useEffect, useRef, useCallback } from "react";
import { useQuery } from "@apollo/client";
import { useIntl } from "react-intl";
import { Helmet } from "react-helmet";
import { LoadingIndicator } from "../Shared/LoadingIndicator";
import { useTitleProps } from "src/hooks/title";
import { FIND_VIEW_HISTORY } from "src/core/StashService/types/viewHistory";
import { ViewHistoryCard } from "./ViewHistoryCard";
import { ViewHistoryEntry, ViewHistoryResult } from "./types";
import "./ViewHistory.scss";

const ITEMS_PER_PAGE = 50;

export const ViewHistory: React.FC = () => {
  const intl = useIntl();
  const titleProps = useTitleProps({ id: "view_history" });

  const [page, setPage] = useState(1);
  const [items, setItems] = useState<ViewHistoryEntry[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const loadingRef = useRef<HTMLDivElement>(null);

  const { data, loading, fetchMore } = useQuery<{
    findViewHistory: ViewHistoryResult;
  }>(FIND_VIEW_HISTORY, {
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
          return data.findViewHistory.items;
        }
        return [...prev, ...data.findViewHistory.items];
      });
      setHasMore(data.findViewHistory.items.length === ITEMS_PER_PAGE);
    }
  }, [data, page]);

  const loadMore = useCallback(() => {
    if (!loading && hasMore) {
      setPage((prev) => prev + 1);
      fetchMore({
        variables: {
          filter: {
            page: page + 1,
            per_page: ITEMS_PER_PAGE,
          },
        },
      });
    }
  }, [loading, hasMore, page, fetchMore]);

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

    if (loadingRef.current) {
      observer.observe(loadingRef.current);
    }

    return () => {
      if (loadingRef.current) {
        observer.unobserve(loadingRef.current);
      }
    };
  }, [loadMore, hasMore, loading]);

  const totalViews = data?.findViewHistory?.count || 0;

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
  }, {} as Record<string, ViewHistoryEntry[]>);

    return (
      <>
        <Helmet {...titleProps} />
        <div className="view-history-container">
          <div className="view-history-header">
            <h1>{intl.formatMessage({ id: "view_history" })}</h1>
            <span className="view-history-total-views">
              {intl.formatMessage(
                { id: "total_views" },
                { count: totalViews }
              )}
            </span>
          </div>

          <div className="view-history-content">
            {Object.entries(groupedItems).map(([date, dateItems]) => (
              <div key={date} className="view-history-date-group">
                <h3 className="view-history-date-header">{date}</h3>
                <div className="view-history-items-list">
                  {dateItems.map((item) => (
                    <ViewHistoryCard
                      key={`${item.scene?.id || item.gallery?.id}-${item.viewDate}`}
                      scene={item.scene}
                      gallery={item.gallery}
                      viewDate={item.viewDate}
                      oDate={item.oDate}
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