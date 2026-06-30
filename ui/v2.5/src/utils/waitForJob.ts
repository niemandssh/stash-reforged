import { getClient } from "src/core/StashService";
import {
  FindJobDocument,
  JobStatus,
} from "src/core/generated-graphql";

export async function waitForJob(
  jobID: string | null | undefined
): Promise<void> {
  if (!jobID || jobID === "0") {
    return;
  }

  const client = getClient();

  for (;;) {
    const result = await client.query({
      query: FindJobDocument,
      variables: { input: { id: jobID } },
      fetchPolicy: "network-only",
    });

    const job = result.data?.findJob;
    if (!job) {
      return;
    }

    if (
      job.status === JobStatus.Finished ||
      job.status === JobStatus.Failed ||
      job.status === JobStatus.Cancelled
    ) {
      if (job.status === JobStatus.Failed) {
        throw new Error(job.error ?? "Job failed");
      }
      return;
    }

    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
}
