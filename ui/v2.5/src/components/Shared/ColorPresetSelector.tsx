import React, { useState, useEffect } from "react";
import { Form, Button, Modal, Row, Col } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { useFindColorPresets, useColorPresetCreate, useColorPresetUpdate, useColorPresetDestroy } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";
import { ColorPreset } from "src/core/generated-graphql";
import { faGripLines } from "@fortawesome/free-solid-svg-icons";
import { Icon } from "src/components/Shared/Icon";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core';
import { restrictToVerticalAxis } from '@dnd-kit/modifiers';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import {
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

interface SortablePresetCardProps {
  preset: ColorPreset;
  onEdit: (preset: ColorPreset) => void;
  onDelete: (preset: ColorPreset) => void;
  onUse: (preset: ColorPreset) => void;
}

const SortablePresetCard: React.FC<SortablePresetCardProps> = ({
  preset,
  onEdit,
  onDelete,
  onUse,
}) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: preset.id });

  const dragHandleRef = React.useRef<HTMLDivElement>(null);

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
    zIndex: isDragging ? 1000 : 'auto',
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="w-100 mb-2"
    >
      <div className="card p-2 h-100" style={{ margin: 0 }}>
        <div className="card-body pl-1 pr-2 p-0">
          <div className="d-flex align-items-center">
            <div
              ref={dragHandleRef}
              className="drag-handle mr-3"
              style={{ cursor: 'grab', flexShrink: 0 }}
              {...attributes}
              {...listeners}
            >
              <Icon icon={faGripLines} />
            </div>
            <div
              className="color-preview mr-3"
              style={{
                width: "40px",
                height: "40px",
                backgroundColor: preset.color,
                border: `2px solid ${preset.color}`,
                borderRadius: "4px",
                flexShrink: 0,
              }}
            />
            <div className="flex-grow-1">
              <h6 className="mb-1">{preset.name}</h6>
              <small className="text-muted">{preset.color}</small>
              <div className="text-muted small">
                <FormattedMessage id="color_preset.sort" />: {preset.sort}
              </div>
            </div>
            <div className="btn-group btn-group-sm ml-2">
              <Button
                variant="outline-primary"
                size="sm"
                onClick={(e) => {
                  e.stopPropagation();
                  onUse(preset);
                }}
                title="Use"
              >
                <FormattedMessage id="actions.use" />
              </Button>
              <Button
                variant="outline-success"
                size="sm"
                onClick={(e) => {
                  e.stopPropagation();
                  onEdit(preset);
                }}
                title="Edit"
              >
                <FormattedMessage id="actions.edit" />
              </Button>
              <Button
                variant="outline-danger"
                size="sm"
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete(preset);
                }}
                title="Delete"
              >
                <FormattedMessage id="actions.delete" />
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

interface ColorPresetSelectorProps {
  selectedColor: string;
  onColorSelect: (color: string) => void;
  onPresetSelect: (preset: ColorPreset) => void;
}

export const ColorPresetSelector: React.FC<ColorPresetSelectorProps> = ({
  selectedColor,
  onColorSelect,
  onPresetSelect,
}) => {
  const intl = useIntl();
  const Toast = useToast();

  const [showPresetModal, setShowPresetModal] = useState(false);
  const [presetName, setPresetName] = useState("");
  const [presetSort, setPresetSort] = useState(1);
  const [presetColor, setPresetColor] = useState("");
  const [editingPreset, setEditingPreset] = useState<ColorPreset | null>(null);

  const { data: presetsData, refetch: refetchPresets, loading: presetsLoading } = useFindColorPresets();
  const [createPreset] = useColorPresetCreate();
  const [updatePreset] = useColorPresetUpdate();
  const [destroyPreset] = useColorPresetDestroy();

  const presets = presetsData?.findColorPresets?.color_presets || [];

  // Initialize presetColor and presetSort when opening modal for creation
  React.useEffect(() => {
    if (showPresetModal && !editingPreset) {
      if (!presetColor && selectedColor) {
        setPresetColor(selectedColor);
      }
      if (!presetSort || presetSort === 1) {
        setPresetSort(presets.length + 1);
      }
    }
  }, [showPresetModal, editingPreset, presetColor, presetSort, selectedColor, presets.length]);

  // Sensors for drag & drop
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  // Handle drag end
  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      const oldIndex = presets.findIndex((preset) => preset.id === active.id);
      const newIndex = presets.findIndex((preset) => preset.id === over.id);

      const reorderedPresets = arrayMove(presets, oldIndex, newIndex);

      // Update sort values for all presets
      const updatePromises = reorderedPresets.map((preset, index) => {
        const newSort = index + 1;
        if (preset.sort !== newSort) {
          return updatePreset({
            variables: {
              input: {
                id: preset.id,
                sort: newSort,
              },
            },
          });
        }
        return null;
      }).filter(Boolean);

      try {
        await Promise.all(updatePromises);
        refetchPresets();
        Toast.success(intl.formatMessage({ id: "toast.updated_entity" }, { entity: "Color presets order" }));
      } catch (e) {
        Toast.error(e);
      }
    }
  };

  const handlePresetClick = (preset: ColorPreset) => {
    onPresetSelect(preset);
    onColorSelect(preset.color);
  };

  const handleCreatePreset = async () => {
    if (!presetName.trim() || !presetColor) return;

    try {
      await createPreset({
        variables: {
          input: {
            name: presetName.trim(),
            color: presetColor,
            sort: presetSort,
          },
        },
      });
      setPresetName("");
      setPresetSort(1);
      setPresetColor("");
      setShowPresetModal(false);
      refetchPresets();
      Toast.success(intl.formatMessage({ id: "color_preset.created" }));
    } catch (e) {
      Toast.error(e);
    }
  };

  const handleEditPreset = async () => {
    if (!editingPreset || !presetName.trim()) return;

    try {
      await updatePreset({
        variables: {
          input: {
            id: editingPreset.id,
            name: presetName.trim(),
            color: presetColor,
            sort: presetSort,
          },
        },
      });
      refetchPresets();
      Toast.success(intl.formatMessage({ id: "color_preset.updated" }));
    } catch (e) {
      Toast.error(e);
    }
  };

  const handleDeletePreset = async (preset: ColorPreset) => {
    if (!confirm(intl.formatMessage({ id: "color_preset.confirm_delete" }, { name: preset.name }))) {
      return;
    }

    try {
      await destroyPreset({
        variables: {
          input: {
            id: preset.id,
          },
        },
      });
      refetchPresets();
      Toast.success(intl.formatMessage({ id: "color_preset.deleted" }));

      // Reset form to creation mode if deleting the currently edited preset
      if (editingPreset && editingPreset.id === preset.id) {
        setEditingPreset(null);
        setPresetName("");
        setPresetColor("");
        setPresetSort(1);
      }
    } catch (e) {
      Toast.error(e);
    }
  };

  const handleEditClick = (preset: ColorPreset) => {
    setEditingPreset(preset);
    setPresetName(preset.name);
    setPresetSort(preset.sort);
    setPresetColor(preset.color);
    setShowPresetModal(true);
  };

  const handleModalClose = () => {
    setShowPresetModal(false);
    setEditingPreset(null);
    setPresetName("");
    setPresetSort(0);
    setPresetColor("");
  };

  return (
    <>
      <div className="color-preset-selector">
        {/* Color presets */}
        <div className="color-presets mb-3">
          <small className="text-muted d-block mb-2">
            <FormattedMessage id="color_preset.presets" />:
          </small>
          {presetsLoading ? (
            <div className="text-muted">
              <FormattedMessage id="loading.generic" />
            </div>
          ) : presets.length > 0 ? (
            <div className="d-flex flex-wrap gap-1">
              {presets.map((preset) => (
                <div
                  key={preset.id}
                  className="color-preset-item d-flex align-items-center"
                  style={{
                    backgroundColor: preset.color,
                    color: getContrastColor(preset.color),
                    border: `1px solid ${preset.color}`,
                    borderRadius: "4px",
                    padding: "4px 8px",
                    fontSize: "0.875rem",
                    cursor: "pointer",
                    marginRight: "4px",
                    marginBottom: "4px",
                  }}
                  onClick={() => handlePresetClick(preset)}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    handleEditClick(preset);
                  }}
                  title={`${preset.name} (${preset.color})`}
                >
                  <span>{preset.name}</span>
                </div>
              ))}
              <Button
                variant="outline-secondary"
                size="sm"
                onClick={() => setShowPresetModal(true)}
                title={intl.formatMessage({ id: "color_preset.manage" })}
                style={{ marginBottom: "4px" }}
              >
                <FormattedMessage id="color_preset.manage" />
              </Button>
            </div>
          ) : (
            <div className="d-flex align-items-center">
              <div className="text-muted mr-3">
                <FormattedMessage id="color_preset.no_presets" />
              </div>
              <Button
                variant="outline-secondary"
                size="sm"
                onClick={() => setShowPresetModal(true)}
                title={intl.formatMessage({ id: "color_preset.manage" })}
              >
                <FormattedMessage id="color_preset.manage" />
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Modal for managing presets */}
      <Modal show={showPresetModal} onHide={handleModalClose} size="lg">
        <Modal.Header closeButton>
          <Modal.Title>
            <FormattedMessage id="color_preset.manage" />
          </Modal.Title>
        </Modal.Header>
        <Modal.Body>
          {/* List of existing presets */}
          <div className="mb-4">
            <div className="d-flex justify-content-between align-items-center mb-3">
              <h6>
                <FormattedMessage id="color_preset.presets" /> ({presets.length})
              </h6>
              <small className="text-muted">
                <FormattedMessage id="color_preset.drag_to_reorder" />
              </small>
            </div>
            {presetsLoading ? (
              <div className="text-center py-3">
                <FormattedMessage id="loading.generic" />
              </div>
            ) : presets.length > 0 ? (
              <DndContext
                sensors={sensors}
                collisionDetection={closestCenter}
                modifiers={[restrictToVerticalAxis]}
                onDragEnd={handleDragEnd}
              >
                <SortableContext
                  items={presets.map(p => p.id)}
                  strategy={verticalListSortingStrategy}
                >
                  <div className="d-flex flex-column">
                    {presets.map((preset) => (
                      <SortablePresetCard
                        key={preset.id}
                        preset={preset}
                        onEdit={handleEditClick}
                        onDelete={handleDeletePreset}
                        onUse={handlePresetClick}
                      />
                    ))}
                  </div>
                </SortableContext>
              </DndContext>
            ) : (
              <div className="text-center py-4 text-muted">
                <FormattedMessage id="color_preset.no_presets" />
              </div>
            )}
          </div>

          <hr className="mb-4" />

          {/* Form for creating/editing preset */}
          <div>
            <div className="d-flex justify-content-between align-items-center mb-3">
              <h6 className="mb-0">
                <FormattedMessage id={editingPreset ? "color_preset.edit" : "color_preset.create"} />
              </h6>
              {editingPreset && (
                <Button
                  variant="outline-primary"
                  size="sm"
                  onClick={() => {
                    setEditingPreset(null);
                    setPresetName("");
                    setPresetSort(1);
                    setPresetColor("");
                  }}
                  title={intl.formatMessage({ id: "color_preset.create" })}
                >
                  <FormattedMessage id="color_preset.create" />
                </Button>
              )}
            </div>
            <Form.Group className="mb-3">
              <Form.Label>
                <FormattedMessage id="color_preset.name" />
              </Form.Label>
              <Form.Control
                type="text"
                name="preset_name"
                className="text-input"
                value={presetName}
                onChange={(e) => setPresetName(e.target.value)}
                placeholder={intl.formatMessage({ id: "color_preset.name_placeholder" })}
              />
            </Form.Group>
            <Form.Group className="mb-3">
              <Form.Label>
                <FormattedMessage id="color_preset.color" />
              </Form.Label>
              <div className="d-flex align-items-center">
                <Form.Control
                  type="color"
                  name="preset_color"
                  className="text-input mr-2"
                  value={presetColor || "#bfccd6"}
                  onChange={(e) => setPresetColor(e.target.value)}
                  style={{ width: "60px", height: "38px" }}
                />
                <Form.Control
                  type="text"
                  name="preset_color_text"
                  className="text-input"
                  value={presetColor || ""}
                  onChange={(e) => setPresetColor(e.target.value)}
                  placeholder="#000000"
                  style={{ flex: 1 }}
                />
              </div>
            </Form.Group>
            <Form.Group className="mb-3">
              <Form.Label>
                <FormattedMessage id="color_preset.sort" />
              </Form.Label>
              <Form.Control
                type="number"
                name="preset_sort"
                className="text-input"
                value={presetSort}
                onChange={(e) => setPresetSort(parseInt(e.target.value) || 0)}
                placeholder="0"
              />
            </Form.Group>
          </div>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={handleModalClose}>
            <FormattedMessage id="actions.close" />
          </Button>
          <Button
            variant="primary"
            onClick={editingPreset ? handleEditPreset : handleCreatePreset}
            disabled={!presetName.trim() || !presetColor}
          >
            <FormattedMessage id={editingPreset ? "actions.save" : "actions.create"} />
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

// Utility for determining text color based on background color
function getContrastColor(backgroundColor: string): string {
  if (!backgroundColor) return "#000000";

  let r = 0, g = 0, b = 0;

  if (backgroundColor.startsWith("#")) {
    const hex = backgroundColor.replace("#", "");
    if (hex.length === 3) {
      r = parseInt(hex[0] + hex[0], 16);
      g = parseInt(hex[1] + hex[1], 16);
      b = parseInt(hex[2] + hex[2], 16);
    } else if (hex.length === 6) {
      r = parseInt(hex.substr(0, 2), 16);
      g = parseInt(hex.substr(2, 2), 16);
      b = parseInt(hex.substr(4, 2), 16);
    }
  }

  const brightness = (r * 299 + g * 587 + b * 114) / 1000;
  return brightness > 128 ? "#000000" : "#ffffff";
}
