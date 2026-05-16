import React from "react";
import { VisitActions } from "./VisitActions.js";

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
      <div className="layout-columns">
        <section className="panel panel-actions" aria-label="Visit actions">
          <VisitActions
            isQueueing={isQueueing}
            onQueueOne={onQueueOne ?? (() => undefined)}
            onQueueTen={onQueueTen ?? (() => undefined)}
          />
        </section>
        <section className="panel panel-status" aria-label="Visit status">
          <div className="status-head">
            <p className="count-row">
              Total visits: <strong className="count">{count}</strong>
            </p>
            <button
              className="icon-refresh"
              type="button"
              onClick={onRefresh}
              disabled={isRefreshing}
              aria-label="Refresh count"
              title="Refresh count"
            >
              <span className={`refresh-glyph ${isRefreshing ? "is-hidden" : ""}`}>↻</span>
              <span className={`spinner ${isRefreshing ? "is-visible" : ""}`} />
            </button>
          </div>
          <p className={`msg ${toneClassName}`}>{message}</p>
        </section>
      </div>
    </main>
  );
}
