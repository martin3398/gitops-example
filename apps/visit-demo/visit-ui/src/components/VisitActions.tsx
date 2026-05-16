import React from "react";

type Props = {
  isQueueing: boolean;
  onQueueOne: () => void;
  onQueueTen: () => void;
};

export function VisitActions({ isQueueing, onQueueOne, onQueueTen }: Props) {
  return (
    <div className="actions-column">
      <button type="button" onClick={onQueueOne} disabled={isQueueing}>Register Visit</button>
      <button type="button" onClick={onQueueTen} disabled={isQueueing}>Register Visit x10</button>
    </div>
  );
}
