import { useEffect, useRef } from "react";
import { useApolloClient } from "@apollo/client";
import { useJobsSubscribeSubscription, JobStatus, JobStatusUpdateType } from "src/core/generated-graphql";

interface SimilarityJobMonitorOptions {
  onSimilarityJobComplete?: (sceneId: string) => void;
  refetch?: () => void;
}

export const useSimilarityJobMonitor = (options?: SimilarityJobMonitorOptions) => {
  const client = useApolloClient();
  const { data: jobsData } = useJobsSubscribeSubscription();
  const onCompleteRef = useRef(options?.onSimilarityJobComplete);
  const refetchRef = useRef(options?.refetch);

  // Update refs when options change
  useEffect(() => {
    onCompleteRef.current = options?.onSimilarityJobComplete;
    refetchRef.current = options?.refetch;
  }, [options?.onSimilarityJobComplete, options?.refetch]);

  useEffect(() => {
    if (!jobsData?.jobsSubscribe) return;

    const event = jobsData.jobsSubscribe;
    const job = event.job;

    // Check if this is a similarity job
    const isSimilarityJob = job.description?.includes("Recalculating similarities for scene");
    
    if (!isSimilarityJob) return;

    // Check if job was completed (removed or finished)
    const isJobComplete = 
      event.type === JobStatusUpdateType.Remove ||
      job.status === JobStatus.Finished ||
      job.status === JobStatus.Failed ||
      job.status === JobStatus.Cancelled;

    if (!isJobComplete) return;

    // Extract scene ID from job description
    const sceneIdMatch = job.description?.match(/scene (\d+)/);
    if (!sceneIdMatch) return;

    const sceneId = sceneIdMatch[1];
    
    // Evict similar scenes cache to force refetch
    client.cache.evict({
      fieldName: "findScene",
      args: { id: sceneId }
    });

    // Also trigger a refetch if available
    refetchRef.current?.();

    // Call the completion callback
    onCompleteRef.current?.(sceneId);
  }, [jobsData, client.cache]);
};
