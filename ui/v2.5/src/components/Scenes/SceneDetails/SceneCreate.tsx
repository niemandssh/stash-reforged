import React, { useEffect, useMemo, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useLocation } from "react-router-dom";
import { SceneEditPanel } from "./SceneEditPanel";
import * as GQL from "src/core/generated-graphql";
import { mutateCreateScene, useFindScene } from "src/core/StashService";
import ImageUtils from "src/utils/image";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";

const SceneCreate: React.FC = () => {
  const intl = useIntl();
  const Toast = useToast();

  const location = useLocation();
  const query = useMemo(() => new URLSearchParams(location.search), [location]);
  const scrapeUrl = query.get("scrapeUrl");

  // create scene from provided scene id if applicable
  const { data, loading } = useFindScene(query.get("from_scene_id") ?? "new");
  const [loadingCoverImage, setLoadingCoverImage] = useState(false);
  const [coverImage, setCoverImage] = useState<string>();

  const scene = useMemo(() => {
    if (data?.findScene) {
      return {
        ...data.findScene,
        paths: undefined,
        id: undefined,
      };
    }

    return {
      title: query.get("q") ?? "Untitled",
      urls: scrapeUrl ? [scrapeUrl] : undefined,
    };
  }, [data?.findScene, query, scrapeUrl]);

  useEffect(() => {
    async function fetchCoverImage() {
      const srcScene = data?.findScene;
      if (srcScene?.paths.screenshot) {
        setLoadingCoverImage(true);
        const imageData = await ImageUtils.imageToDataURL(
          srcScene.paths.screenshot
        );
        setCoverImage(imageData);
        setLoadingCoverImage(false);
      } else {
        setCoverImage(undefined);
      }
    }

    fetchCoverImage();
  }, [data?.findScene]);

  if (loading || loadingCoverImage) {
    return <LoadingIndicator />;
  }

  async function onSave(input: GQL.SceneCreateInput) {
    const fileID = query.get("file_id") ?? undefined;
    const result = await mutateCreateScene({
      ...input,
      file_ids: fileID ? [fileID] : undefined,
    });
    if (result.data?.sceneCreate?.id) {
      Toast.success(
        intl.formatMessage(
          { id: "toast.created_entity" },
          { entity: intl.formatMessage({ id: "scene" }).toLocaleLowerCase() }
        )
      );
      return result.data.sceneCreate.id;
    }
  }

  return (
    <div className="row new-view justify-content-center" id="create-scene-page">
      <div className="col-md-8">
        <h2>
          <FormattedMessage
            id="actions.create_entity"
            values={{ entityType: intl.formatMessage({ id: "scene" }) }}
          />
        </h2>
        <SceneEditPanel
          scene={scene}
          initialCoverImage={coverImage}
          isVisible
          isNew
          onSubmit={onSave}
        />
      </div>
    </div>
  );
};

export default SceneCreate;
