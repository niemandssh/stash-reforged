import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  useJobsSubscribeSubscription,
  JobStatus,
  JobStatusUpdateType,
} from "src/core/generated-graphql";
import { useToast } from "./Toast";
import { useIntl } from "react-intl";

interface ISimilarityJobMonitorOptions {
  onSimilarityJobComplete?: (sceneId: string) => void;
  refetch?: () => void;
}

export const useSimilarityJobMonitor = (
  options?: ISimilarityJobMonitorOptions
) => {
  const queryClient = useQueryClient();
  const { data: jobsData } = useJobsSubscribeSubscription();
  const onCompleteRef = useRef(options?.onSimilarityJobComplete);
  const refetchRef = useRef(options?.refetch);
  const Toast = useToast();
  const intl = useIntl();

  // Track processed jobs to avoid duplicate notifications
  const [processedJobs, setProcessedJobs] = useState<Set<string>>(new Set());

  // Clean up old processed jobs periodically to avoid memory leaks
  useEffect(() => {
    const cleanup = setInterval(() => {
      setProcessedJobs(new Set());
    }, 60000); // Clear every minute

    return () => clearInterval(cleanup);
  }, []);

  // Update refs when options change
  useEffect(() => {
    onCompleteRef.current = options?.onSimilarityJobComplete;
    refetchRef.current = options?.refetch;
  }, [options?.onSimilarityJobComplete, options?.refetch]);

  useEffect(() => {
    if (!jobsData?.jobsSubscribe) return;

    const { jobsSubscribe: event } = jobsData;
    const { job } = event;

    // Check if this is a similarity job
    const isSimilarityJob = job.description?.includes(
      "Recalculating similarities for scene"
    );

    if (!isSimilarityJob) return;

    // Extract scene ID from job description
    const sceneIdMatch = job.description?.match(/scene (\d+)/);
    if (!sceneIdMatch) return;

    const sceneId = sceneIdMatch[1];
    const jobKey = `${job.id}-${sceneId}`;

    // Check if we've already processed this job event
    if (processedJobs.has(jobKey)) return;

    // Check if job was just started
    const isJobStarted =
      event.type === JobStatusUpdateType.Add ||
      (event.type === JobStatusUpdateType.Update &&
        job.status === JobStatus.Running);

    // Check if job was completed (removed or finished)
    const isJobComplete =
      event.type === JobStatusUpdateType.Remove ||
      job.status === JobStatus.Finished ||
      job.status === JobStatus.Failed ||
      job.status === JobStatus.Cancelled;

    if (isJobStarted) {
      // Mark as processed and show notification about starting similarity recalculation
      setProcessedJobs((prev) => new Set(prev).add(jobKey));
      Toast.success(
        intl.formatMessage(
          { id: "toast.similarity_recalculation_started" },
          { sceneId }
        )
      );
    }

    if (isJobComplete) {
      // Mark as processed and show notification about completion
      setProcessedJobs((prev) => new Set(prev).add(jobKey));

      if (job.status === JobStatus.Finished) {
        Toast.success(
          intl.formatMessage(
            { id: "toast.similarity_recalculation_completed" },
            { sceneId }
          )
        );
      } else if (job.status === JobStatus.Failed) {
        Toast.error(
          intl.formatMessage(
            { id: "toast.similarity_recalculation_failed" },
            { sceneId }
          )
        );
      }

      // Invalidate similar scenes cache to force refetch
      queryClient.invalidateQueries({
        queryKey: ["scenes", "similar", sceneId],
      });

      // Also trigger a refetch if available
      refetchRef.current?.();

      // Call the completion callback
      onCompleteRef.current?.(sceneId);
    }
  }, [jobsData, queryClient, Toast, intl, processedJobs]);
};
