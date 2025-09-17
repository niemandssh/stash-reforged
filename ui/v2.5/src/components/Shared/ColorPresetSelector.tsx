import React, { useState, useEffect } from "react";
import { Form, Button, Modal, Row, Col } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { useFindColorPresets, useColorPresetCreate, useColorPresetUpdate, useColorPresetDestroy } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";
import { ColorPreset } from "src/core/generated-graphql";

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
  const [editingPreset, setEditingPreset] = useState<ColorPreset | null>(null);

  const { data: presetsData, refetch: refetchPresets, loading: presetsLoading } = useFindColorPresets();
  const [createPreset] = useColorPresetCreate();
  const [updatePreset] = useColorPresetUpdate();
  const [destroyPreset] = useColorPresetDestroy();

  const presets = presetsData?.findColorPresets?.color_presets || [];


  const handlePresetClick = (preset: ColorPreset) => {
    onPresetSelect(preset);
    onColorSelect(preset.color);
  };

  const handleCreatePreset = async () => {
    if (!presetName.trim() || !selectedColor) return;

    try {
      await createPreset({
        variables: {
          input: {
            name: presetName.trim(),
            color: selectedColor,
          },
        },
      });
      setPresetName("");
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
            color: selectedColor,
          },
        },
      });
      setEditingPreset(null);
      setPresetName("");
      setShowPresetModal(false);
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
      }
    } catch (e) {
      Toast.error(e);
    }
  };

  const handleEditClick = (preset: ColorPreset) => {
    setEditingPreset(preset);
    setPresetName(preset.name);
    onColorSelect(preset.color);
    setShowPresetModal(true);
  };

  const handleModalClose = () => {
    setShowPresetModal(false);
    setEditingPreset(null);
    setPresetName("");
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
              <FormattedMessage id="loading" />
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
            <h6>
              <FormattedMessage id="color_preset.presets" /> ({presets.length})
            </h6>
            {presetsLoading ? (
              <div className="text-center py-3">
                <FormattedMessage id="loading" />
              </div>
            ) : presets.length > 0 ? (
              <div 
                className="d-flex flex-wrap"
                style={{ gap: "1rem" }}
              >
                {presets.map((preset, index) => (
                  <div 
                    key={preset.id} 
                    style={{ 
                      flex: "0 0 calc(50% - 0.5rem)",
                      minWidth: "300px"
                    }}
                  >
                    <div className="card h-100" style={{ margin: 0 }}>
                      <div className="card-body p-2">
                        <div className="d-flex align-items-center">
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
                          </div>
                          <div className="btn-group btn-group-sm">
                            <Button
                              variant="outline-primary"
                              size="sm"
                              onClick={() => handlePresetClick(preset)}
                              title={intl.formatMessage({ id: "actions.use" })}
                            >
                              <FormattedMessage id="actions.use" />
                            </Button>
                            <Button
                              variant="outline-success"
                              size="sm"
                              onClick={() => handleEditClick(preset)}
                              title={intl.formatMessage({ id: "actions.edit" })}
                            >
                              <FormattedMessage id="actions.edit" />
                            </Button>
                            <Button
                              variant="outline-danger"
                              size="sm"
                              onClick={() => handleDeletePreset(preset)}
                              title={intl.formatMessage({ id: "actions.delete" })}
                            >
                              <FormattedMessage id="actions.delete" />
                            </Button>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
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
                  value={selectedColor || "#bfccd6"}
                  onChange={(e) => onColorSelect(e.target.value)}
                  style={{ width: "60px", height: "38px" }}
                />
                <Form.Control
                  type="text"
                  name="preset_color_text"
                  className="text-input"
                  value={selectedColor || ""}
                  onChange={(e) => onColorSelect(e.target.value)}
                  placeholder="#000000"
                  style={{ flex: 1 }}
                />
              </div>
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
            disabled={!presetName.trim() || !selectedColor}
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
  
  // Обработка hex цветов
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
