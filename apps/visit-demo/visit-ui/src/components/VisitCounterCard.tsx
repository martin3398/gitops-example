import React from "react";

type Props = {
  count: number;
  message: string;
  toneClassName: "ok" | "err";
  isQueueing?: boolean;
  isRefreshing?: boolean;
  onQueueOne?: () => void;
  onQueueTen?: () => void;
  onRefresh?: () => void;
};

export function VisitCounterCard({
  count,
  message,
  toneClassName,
  isQueueing = false,
  isRefreshing = false,
  onQueueOne,
  onQueueTen,
  onRefresh,
}: Props) {
  return (
    <main className="card">
      <h1>Visit Counter</h1>
      <p>Server-rendered count. Click queues a visit and re-renders with fresh data.</p>
      <div className="row">
        <button type="button" onClick={onQueueOne} disabled={isQueueing}>Register Visit</button>
        <button type="button" onClick={onQueueTen} disabled={isQueueing}>Register Visit x10</button>
        <button type="button" onClick={onRefresh} disabled={isRefreshing}>{isRefreshing ? "Refreshing..." : "Refresh"}</button>
        <span>
          Total visits: <strong className="count">{count}</strong>
        </span>
      </div>
      <p className={`msg ${toneClassName}`}>{message}</p>
    </main>
  );
}
