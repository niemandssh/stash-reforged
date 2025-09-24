import React from "react";
import { FormattedMessage } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { PoseTagIcon } from "src/components/Shared/PoseTagIcon";

interface IPoseTagsDisplayProps {
  scene: GQL.SceneDataFragment;
}

export const PoseTagsDisplay: React.FC<IPoseTagsDisplayProps> = ({ scene }) => {

  const poseTags = [...scene.tags]
    .filter(tag => tag.is_pose_tag)
    .sort((a, b) => {
      const aCount = a.scene_count || 0;
      const bCount = b.scene_count || 0;
      return bCount - aCount;
    });

  if (poseTags.length === 0) {
    return null;
  }

  const handlePoseTagClick = (tagId: string) => {
    window.open(`/tags/${tagId}`, '_blank');
  };

  const handleImageClick = (e: React.MouseEvent, tagId: string) => {
    e.stopPropagation();
    handlePoseTagClick(tagId);
  };

  return (
    <>
      <div className="mt-3 mb-3">
        <h4>
          <FormattedMessage
            id="pose_tags"
            defaultMessage="Pose Tags"
            values={{ count: poseTags.length }}
          />
        </h4>
      </div>
      <div className="pose-tags-display">
        {poseTags.map((tag) => (
          <div
            key={tag.id}
            className="pose-tag-item-display"
            onClick={() => handlePoseTagClick(tag.id)}
            style={{ cursor: 'pointer' }}
          >
            <div className="pose-tag-icon-display">
              {tag.image_path ? (
                <img 
                  src={tag.image_path} 
                  alt={tag.name}
                  className="pose-tag-image-display"
                  onClick={(e) => handleImageClick(e, tag.id)}
                />
              ) : (
                <PoseTagIcon className="pose-icon-display" />
              )}
            </div>
            <div className="pose-tag-content-display">
              <div className="pose-tag-name-display">{tag.name}</div>
            </div>
          </div>
        ))}
      </div>
    </>
  );
};
