import React, { useEffect, useState, useRef } from "react";
import { FormattedMessage } from "react-intl";
import { ListGroup, Form } from "react-bootstrap";
import { useHistory } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { queryFindTags } from "src/core/StashService";
import { ListFilterModel } from "src/models/list-filter/filter";
import { PoseTagIcon } from "./PoseTagIcon";
import { Icon } from "./Icon";
import { faCheck, faChevronDown, faChevronRight } from "@fortawesome/free-solid-svg-icons";

interface IPoseTagSelectorProps {
  selectedTagIds: string[];
  onSelectionChange: (tagIds: string[]) => void;
  disabled?: boolean;
}

export const PoseTagSelector: React.FC<IPoseTagSelectorProps> = ({
  selectedTagIds,
  onSelectionChange,
  disabled = false,
}) => {
  const history = useHistory();
  const [poseTags, setPoseTags] = useState<GQL.Tag[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCollapsed, setIsCollapsed] = useState(false);
  const STORAGE_KEY = 'pose-tag-selector-collapsed';
  const isMountedRef = useRef(true);

  // Load collapsed state from localStorage on component mount
  useEffect(() => {
    const savedState = localStorage.getItem(STORAGE_KEY);
    if (savedState !== null) {
      setIsCollapsed(JSON.parse(savedState));
    }
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  // Save collapsed state to localStorage when it changes
  const toggleCollapsed = () => {
    const newState = !isCollapsed;
    setIsCollapsed(newState);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(newState));
  };

  useEffect(() => {
    const loadPoseTags = async () => {
      try {
        setLoading(true);
        const filter = new ListFilterModel(GQL.FilterMode.Tags);
        filter.itemsPerPage = -1;
        filter.sortBy = "name";
        filter.sortDirection = GQL.SortDirectionEnum.Asc;

        const criterion = filter.makeCriterion("is_pose_tag");
        criterion.setFromSavedCriterion({
          modifier: GQL.CriterionModifier.Equals,
          value: "true"
        });
        filter.criteria.push(criterion);

        const result = await queryFindTags(filter);
        const loadedPoseTags = result.data.findTags.tags as unknown as GQL.Tag[];

        const sortedPoseTags = [...loadedPoseTags].sort((a, b) => {
          const aCount = a.scene_count || 0;
          const bCount = b.scene_count || 0;
          return bCount - aCount;
        });

        if (isMountedRef.current) {
          setPoseTags(sortedPoseTags);
        }
      } catch (error) {
        console.error("Error loading pose tags:", error);
      } finally {
        if (isMountedRef.current) {
          setLoading(false);
        }
      }
    };

    loadPoseTags();
  }, []);

  const handleTagToggle = (tagId: string) => {
    if (disabled) return;

    const validSelectedTagIds = selectedTagIds.filter(id => id && typeof id === 'string');
    const isSelected = validSelectedTagIds.includes(tagId);

    let newSelection;
    if (isSelected) {
      newSelection = validSelectedTagIds.filter(id => id !== tagId);
    } else {
      newSelection = [...validSelectedTagIds, tagId];
    }

    onSelectionChange(newSelection);
  };

  const handleImageClick = (e: React.MouseEvent, tagId: string) => {
    e.stopPropagation(); // Предотвращаем срабатывание handleTagToggle
    history.push(`/tags/${tagId}`);
  };

  if (loading) {
    return (
      <div className="pose-tag-selector">
        <Form.Label 
          className="pose-tag-selector-header"
          onClick={toggleCollapsed}
          style={{ cursor: 'pointer', userSelect: 'none' }}
        >
          <Icon 
            icon={isCollapsed ? faChevronRight : faChevronDown} 
            className="ml-0 mr-2"
          />
          <FormattedMessage id="pose_tags" />
        </Form.Label>
        <div className="text-center p-3">
          <FormattedMessage id="loading.generic" />
        </div>
      </div>
    );
  }

  return (
    <div className="pose-tag-selector">
      <Form.Label 
        className="pose-tag-selector-header"
        onClick={toggleCollapsed}
        style={{ cursor: 'pointer', userSelect: 'none' }}
      >
        <Icon 
          icon={isCollapsed ? faChevronRight : faChevronDown} 
          className="ml-0 mr-2"
        />
        <FormattedMessage id="pose_tags" />
      </Form.Label>
      <div 
        className={`pose-tag-list ${isCollapsed ? 'collapsed' : 'expanded'}`}
        style={{
          maxHeight: isCollapsed ? '0' : '225px',
          overflow: 'hidden',
          transition: 'max-height 0.3s ease-in-out'
        }}
      >
        <ListGroup variant="flush">
          {poseTags.map((tag) => {
            const isSelected = selectedTagIds.filter(id => id && typeof id === 'string').includes(tag.id);
            return (
              <ListGroup.Item
                key={tag.id}
                className={`pose-tag-item ${isSelected ? "selected" : ""} ${disabled ? "disabled" : ""}`}
                onClick={() => handleTagToggle(tag.id)}
                style={{ cursor: disabled ? "default" : "pointer" }}
              >
                <div className="d-flex align-items-center">
                  <div className="pose-tag-icon">
                    {tag.image_path ? (
                      <img
                        src={tag.image_path}
                        alt={tag.name}
                        className="pose-tag-image"
                        onClick={(e) => handleImageClick(e, tag.id)}
                        style={{ cursor: 'pointer' }}
                      />
                    ) : (
                      <PoseTagIcon className="pose-icon" />
                    )}
                  </div>
                  <div className="pose-tag-content flex-grow-1">
                    <div className="pose-tag-name">{tag.name}</div>
                    {tag.description && (
                      <div className="pose-tag-description text-muted small">
                        {tag.description}
                      </div>
                    )}
                  </div>
                  {isSelected && (
                    <div className="pose-tag-check">
                      <Icon icon={faCheck} className="text-success" />
                    </div>
                  )}
                </div>
              </ListGroup.Item>
            );
          })}
        </ListGroup>
        {poseTags.length === 0 && (
          <div className="text-center text-muted p-3">
            <FormattedMessage
              id="no_pose_tags_found"
            />
          </div>
        )}
      </div>
    </div>
  );
};
