import React, { useEffect, useState } from "react";
import { VisitCounterCard } from "./VisitCounterCard.js";

type Props = {
  initialCount: number;
  initialMessage: string;
  initialTone: "ok" | "err";
};

export function VisitCounterClient({ initialCount, initialMessage, initialTone }: Props) {
  const [count, setCount] = useState(initialCount);
  const [message, setMessage] = useState(initialMessage);
  const [toneClassName, setToneClassName] = useState<"ok" | "err">(initialTone);
  const [isQueueing, setIsQueueing] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);

  useEffect(() => {
    const id = setInterval(() => {
      void refreshCount(false);
    }, 5000);
    return () => clearInterval(id);
  }, []);

  async function queueVisits(amount: number): Promise<void> {
    if (isQueueing) {
      return;
    }

    setIsQueueing(true);
    setMessage(`Queueing ${amount} visit${amount === 1 ? "" : "s"}...`);
    setToneClassName("ok");

    try {
      const response = await fetch(`/api/v1/visit-events?count=${amount}`, { method: "POST" });
      if (!response.ok) {
        setMessage("Queue request failed");
        setToneClassName("err");
        return;
      }

      const payload = (await response.json()) as { data?: { queued?: number } };
      const queued = typeof payload.data?.queued === "number" ? payload.data.queued : 0;
      if (queued !== amount) {
        setMessage(`${queued} of ${amount} visits queued`);
        setToneClassName("err");
      } else {
        setMessage(`${queued} visit${queued === 1 ? "" : "s"} queued`);
        setToneClassName("ok");
      }

      await refreshCount(false);
    } catch {
      setMessage("Gateway unreachable");
      setToneClassName("err");
    } finally {
      setIsQueueing(false);
    }
  }

  async function refreshCount(manual: boolean): Promise<void> {
    if (isRefreshing) {
      return;
    }

    setIsRefreshing(true);
    if (manual) {
      setMessage("Refreshing...");
      setToneClassName("ok");
    }

    try {
      const response = await fetch("/api/v1/visits/count");
      if (!response.ok) {
        setMessage("Count refresh failed");
        setToneClassName("err");
        return;
      }

      const payload = (await response.json()) as { data?: { count?: number } };
      const nextCount = typeof payload.data?.count === "number" ? payload.data.count : 0;
      setCount(nextCount);
      if (manual) {
        setMessage("Count refreshed");
        setToneClassName("ok");
      }
    } catch {
      setMessage("Gateway unreachable");
      setToneClassName("err");
    } finally {
      setIsRefreshing(false);
    }
  }

  return (
    <VisitCounterCard
      count={count}
      message={message}
      toneClassName={toneClassName}
      isQueueing={isQueueing}
      isRefreshing={isRefreshing}
      onQueueOne={() => {
        void queueVisits(1);
      }}
      onQueueTen={() => {
        void queueVisits(10);
      }}
      onRefresh={() => {
        void refreshCount(true);
      }}
    />
  );
}
