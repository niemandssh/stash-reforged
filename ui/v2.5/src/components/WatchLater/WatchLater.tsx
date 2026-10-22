import React, { useState, useEffect, useRef, useCallback } from "react";
import { useQuery } from "@apollo/client";
import { useIntl } from "react-intl";
import { Helmet } from "react-helmet";
import { useLocation } from "react-router-dom";
import { LoadingIndicator } from "../Shared/LoadingIndicator";
import { useTitleProps } from "src/hooks/title";
import {
  FindWatchLaterDocument,
  FindWatchLaterQuery,
} from "src/core/generated-graphql";
import { ViewHistoryCard } from "../ViewHistory/ViewHistoryCard";
import { SceneContextMenu } from "../Shared/SceneContextMenu";
import { IWatchLaterEntry } from "./types";
import { IScene } from "../ViewHistory/types";
import "../ViewHistory/ViewHistory.scss";

const ITEMS_PER_PAGE = 50;

export const WatchLater: React.FC = () => {
  const intl = useIntl();
  const location = useLocation();
  const titleProps = useTitleProps({ id: "watch_later" });

  const [page, setPage] = useState(1);
  const [items, setItems] = useState<IWatchLaterEntry[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const [displayCount, setDisplayCount] = useState(0);
  const loadingRef = useRef<HTMLDivElement>(null);

  const { data, loading, fetchMore, refetch } = useQuery<FindWatchLaterQuery>(
    FindWatchLaterDocument,
    {
      variables: {
        filter: {
          page,
          per_page: ITEMS_PER_PAGE,
        },
      },
      fetchPolicy: "cache-and-network",
      notifyOnNetworkStatusChange: true,
    }
  );

  useEffect(() => {
    setPage(1);
    setItems([]);
    setHasMore(true);
    refetch({
      filter: {
        page: 1,
        per_page: ITEMS_PER_PAGE,
      },
    });
  }, [location.pathname, refetch]);

  React.useEffect(() => {
    if (data?.findWatchLater?.items) {
      setItems((prev) => {
        if (page === 1) {
          return data.findWatchLater!.items as IWatchLaterEntry[];
        }
        return [
          ...prev,
          ...(data.findWatchLater!.items as IWatchLaterEntry[]),
        ];
      });
      setHasMore(data.findWatchLater.items.length === ITEMS_PER_PAGE);
    }
  }, [data, page]);

  React.useEffect(() => {
    if (data?.findWatchLater?.count !== undefined) {
      setDisplayCount(data.findWatchLater.count);
    }
  }, [data?.findWatchLater?.count]);

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

  const handleRemoved = useCallback((sceneId: string) => {
    setItems((prev) => prev.filter((item) => item.scene.id !== sceneId));
    setDisplayCount((count) => Math.max(0, count - 1));
  }, []);

  if (loading && page === 1 && items.length === 0) {
    return <LoadingIndicator />;
  }

  const groupedItems = items.reduce((acc, item) => {
    const date = new Date(item.addedAt).toLocaleDateString();
    if (!acc[date]) {
      acc[date] = [];
    }
    acc[date].push(item);
    return acc;
  }, {} as Record<string, IWatchLaterEntry[]>);

  return (
    <>
      <Helmet {...titleProps} />
      <div className="view-history-container">
        <div className="view-history-header">
          <h1>{intl.formatMessage({ id: "watch_later" })}</h1>
          <div className="view-history-stats">
            <span className="view-history-total-views">
              {intl.formatMessage(
                { id: "watch_later.total" },
                { count: displayCount }
              )}
            </span>
          </div>
        </div>

        <div className="view-history-content">
          {Object.entries(groupedItems).map(([date, dateItems]) => (
            <div key={date} className="view-history-date-group">
              <h3 className="view-history-date-header">{date}</h3>
              <div className="view-history-items-list">
                {dateItems.map((item) => (
                  <SceneContextMenu
                    key={`${item.scene.id}-${item.addedAt}`}
                    variant="watch-later"
                    sceneId={item.scene.id}
                    sceneUrl={`/scenes/${item.scene.id}`}
                    watchLater
                    onRemoved={() => handleRemoved(item.scene.id)}
                  >
                    <ViewHistoryCard
                      scene={item.scene as IScene}
                      viewDate={item.addedAt}
                    />
                  </SceneContextMenu>
                ))}
              </div>
            </div>
          ))}

          {items.length === 0 && !loading && (
            <p className="text-muted text-center">
              {intl.formatMessage({ id: "watch_later.empty" })}
            </p>
          )}

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

export default WatchLater;
