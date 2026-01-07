import React from "react";
import * as GQL from "src/core/generated-graphql";
import {
  useCardWidth,
  useContainerDimensions,
} from "../Shared/GridCard/GridCard";
import { GameCard } from "./GameCard";

interface IProps {
  games: GQL.SlimGameDataFragment[];
  selectedIds: Set<string>;
  zoomIndex: number;
  onSelectChange: (id: string, selected: boolean, shiftKey: boolean) => void;
}

const zoomWidths = [280, 340, 480, 640];

export const GameCardGrid: React.FC<IProps> = ({
  games,
  selectedIds,
  zoomIndex,
  onSelectChange,
}) => {
  const [containerRef, { width: containerWidth }] = useContainerDimensions();
  const cardWidth = useCardWidth(containerWidth, zoomIndex, zoomWidths);

  return (
    <div
      className="row justify-content-center game-card-grid"
      ref={containerRef}
    >
      {games.map((game) => (
        <GameCard
          key={game.id}
          game={game}
          cardWidth={cardWidth}
          zoomIndex={zoomIndex}
          selecting={selectedIds.size > 0}
          selected={selectedIds.has(game.id)}
          onSelectedChanged={(selected, shiftKey) =>
            onSelectChange(game.id, selected, shiftKey)
          }
        />
      ))}
    </div>
  );
};
