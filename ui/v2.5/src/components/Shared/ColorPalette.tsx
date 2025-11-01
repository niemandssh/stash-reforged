import React, { useState, useEffect } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useFindTagColors } from "src/core/StashService";

interface IColorPaletteProps {
  onColorSelect?: (color: string) => void;
}

export const ColorPalette: React.FC<IColorPaletteProps> = ({ onColorSelect }) => {
  const [colors, setColors] = useState<string[]>([]);
  const [sortBy] = useState<"count" | "color">("color");

  const { data: colorsData } = useFindTagColors();

  useEffect(() => {
    console.log("ColorPalette: colorsData", colorsData);
    if (colorsData?.findTagColors) {
      console.log("ColorPalette: Found colors", colorsData.findTagColors);
      setColors(colorsData.findTagColors);
    }
  }, [colorsData]);

  const sortedColors = [...colors].sort((a, b) => {
    if (sortBy === "color") {
      return a.localeCompare(b);
    } else {
      // For color sorting, we'll just use alphabetical
      return a.localeCompare(b);
    }
  });

  const handleColorClick = (color: string) => {
    if (onColorSelect) {
      onColorSelect(color);
    }
  };

  if (sortedColors.length === 0) {
    return null;
  }

  return (
    <div className="color-palette">
      <small className="text-muted d-block mb-2">
        <FormattedMessage id="color_palette.title" />:
      </small>

      <div className="color-palette-grid" style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fill, minmax(30px, 1fr))",
        gap: "6px",
        maxWidth: "100%",
        marginBottom: "1rem"
      }}>
          {sortedColors.map((color) => (
            <div
              key={color}
              className="color-palette-item"
              style={{
                backgroundColor: color,
                border: `2px solid ${color}`,
                borderRadius: "3px",
                width: "30px",
                height: "30px",
                cursor: onColorSelect ? "pointer" : "default",
                position: "relative",
                transition: "transform 0.2s ease, box-shadow 0.2s ease",
              }}
              onClick={() => handleColorClick(color)}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = "scale(1.1)";
                e.currentTarget.style.boxShadow = "0 4px 8px rgba(0,0,0,0.3)";
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = "scale(1)";
                e.currentTarget.style.boxShadow = "none";
              }}
              title={color}
            >
              {/* Показываем цветовой код при наведении */}
              <div 
                className="color-tooltip"
                style={{
                  position: "absolute",
                  bottom: "-25px",
                  left: "50%",
                  transform: "translateX(-50%)",
                  backgroundColor: "rgba(0,0,0,0.8)",
                  color: "white",
                  padding: "2px 6px",
                  borderRadius: "3px",
                  fontSize: "10px",
                  whiteSpace: "nowrap",
                  opacity: 0,
                  transition: "opacity 0.2s ease",
                  pointerEvents: "none",
                  zIndex: 1000,
                }}
              >
                {color}
              </div>
            </div>
          ))}
      </div>
    </div>
  );
};
