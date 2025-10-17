import React, { useState, useEffect } from "react";
import { Button } from "react-bootstrap";
import { FormattedMessage } from "react-intl";
import Mousetrap from "mousetrap";
import * as GQL from "src/core/generated-graphql";
import { MarkerWallPanel } from "src/components/Wall/WallPanel";
import { PrimaryTags } from "./PrimaryTags";
import { SceneMarkerForm } from "./SceneMarkerForm";
import { Icon } from "src/components/Shared/Icon";
import { faPlay, faStop } from "@fortawesome/free-solid-svg-icons";

interface ISceneMarkersPanelProps {
  sceneId: string;
  isVisible: boolean;
  onClickMarker: (marker: GQL.SceneMarkerDataFragment) => void;
  onPlayMarkers: (markers: GQL.SceneMarkerDataFragment[]) => void;
  onStopMarkers: () => void;
  playingTagId?: string;
  onPlayAllMarkers: (markers: GQL.SceneMarkerDataFragment[]) => void;
}

export const SceneMarkersPanel: React.FC<ISceneMarkersPanelProps> = ({
  sceneId,
  isVisible,
  onClickMarker,
  onPlayMarkers,
  onStopMarkers,
  playingTagId,
  onPlayAllMarkers,
}) => {
  const { data, loading } = GQL.useFindSceneMarkerTagsQuery({
    variables: { id: sceneId },
  });
  const [isEditorOpen, setIsEditorOpen] = useState<boolean>(false);
  const [editingMarker, setEditingMarker] =
    useState<GQL.SceneMarkerDataFragment>();

  // set up hotkeys
  useEffect(() => {
    if (!isVisible) return;

    Mousetrap.bind("n", () => onOpenEditor());

    return () => {
      Mousetrap.unbind("n");
    };
  });

  if (loading) return null;

  function onOpenEditor(marker?: GQL.SceneMarkerDataFragment) {
    setIsEditorOpen(true);
    setEditingMarker(marker ?? undefined);
  }

  const closeEditor = () => {
    setEditingMarker(undefined);
    setIsEditorOpen(false);
  };

  if (isEditorOpen)
    return (
      <SceneMarkerForm
        sceneID={sceneId}
        marker={editingMarker}
        onClose={closeEditor}
      />
    );

  const sceneMarkers = (
    data?.sceneMarkerTags.map((tag) => tag.scene_markers) ?? []
  ).reduce((prev, current) => [...prev, ...current], []);

  const hasRangeMarkers = sceneMarkers.some((m) => m.end_seconds != null);
  const isPlayingAll = playingTagId === "__ALL__";

  return (
    <div className="scene-markers-panel">
      <div className="d-flex mb-2">
        <Button onClick={() => onOpenEditor()}>
          <FormattedMessage id="actions.create_marker" />
        </Button>
        <Button
          className="ml-auto"
          disabled={!hasRangeMarkers}
          onClick={
            isPlayingAll ? onStopMarkers : () => onPlayAllMarkers(sceneMarkers)
          }
        >
          <Icon icon={isPlayingAll ? faStop : faPlay} className="mr-2" />
          <FormattedMessage id="actions.play_all" />
        </Button>
      </div>
      <div className="container">
        <PrimaryTags
          sceneMarkers={sceneMarkers}
          onClickMarker={onClickMarker}
          onEdit={onOpenEditor}
          onPlay={onPlayMarkers}
          onStop={onStopMarkers}
          playingTagId={playingTagId}
        />
      </div>
      <MarkerWallPanel
        markers={sceneMarkers}
        clickHandler={(e, marker) => {
          e.preventDefault();
          window.scrollTo(0, 0);
          onClickMarker(marker);
        }}
      />
    </div>
  );
};

export default SceneMarkersPanel;
