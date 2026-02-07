import * as GQL from "src/core/generated-graphql";
import {
  evictQueries,
  getClient,
  studioMutationImpactedQueries,
} from "src/core/StashService";

export const useUpdatePerformer = () => {
  const [updatePerformer] = GQL.usePerformerUpdateMutation({
    onError: (errors: any) => errors,
    errorPolicy: "all",
  } as any);

  const updatePerformerHandler = (input: GQL.PerformerUpdateInput) =>
    updatePerformer({
      variables: {
        input,
      },
      update: (store: any, updatedPerformer: any) => {
        if (!updatedPerformer.data?.performerUpdate) return;

        updatedPerformer.data.performerUpdate.stash_ids.forEach((id: any) => {
          store.writeQuery({
            query: GQL.FindPerformersDocument,
            variables: {
              performer_filter: {
                stash_id_endpoint: {
                  stash_id: id.stash_id,
                  endpoint: id.endpoint,
                  modifier: GQL.CriterionModifier.Equals,
                },
              },
            },
            data: {
              findPerformers: {
                count: 1,
                performers: [updatedPerformer.data!.performerUpdate!],
                __typename: "FindPerformersResultType",
              },
            },
          });
        });
      },
    } as any);

  return updatePerformerHandler;
};

export const useUpdateStudio = () => {
  const [updateStudio] = GQL.useStudioUpdateMutation({
    onError: (errors: any) => errors,
    errorPolicy: "all",
  } as any);

  const updateStudioHandler = (input: GQL.StudioUpdateInput) =>
    updateStudio({
      variables: {
        input,
      },
      update: (store: any, updatedStudio: any) => {
        if (!updatedStudio.data?.studioUpdate) return;

        if (updatedStudio.data?.studioUpdate?.parent_studio) {
          const ac = getClient() as any;
          evictQueries(ac.cache, studioMutationImpactedQueries);
        } else {
          updatedStudio.data.studioUpdate.stash_ids.forEach((id: any) => {
            store.writeQuery({
              query: GQL.FindStudiosDocument,
              variables: {
                studio_filter: {
                  stash_id_endpoint: {
                    stash_id: id.stash_id,
                    endpoint: id.endpoint,
                    modifier: GQL.CriterionModifier.Equals,
                  },
                },
              },
              data: {
                findStudios: {
                  count: 1,
                  studios: [updatedStudio.data!.studioUpdate!],
                  __typename: "FindStudiosResultType",
                },
              },
            });
          });
        }
      },
    } as any);

  return updateStudioHandler;
};
