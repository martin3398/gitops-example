import React, { useEffect, useState } from "react";
import { useLoaderData, useRevalidator, type LoaderFunctionArgs } from "react-router";
import { VisitCounterCard } from "../components/VisitCounterCard.js";
import { useVisitFeedback } from "../hooks/useVisitFeedback.js";

type LoaderData = {
  count: number;
  queued: number | null;
  queueStatus: "ok" | "unavailable";
  error?: string;
};

export async function loader({ request }: LoaderFunctionArgs): Promise<LoaderData> {
  try {
    const response = await fetch(`${getApiBase(request)}/visits/count`);
    if (!response.ok) {
      return { count: 0, queued: null, queueStatus: "unavailable", error: "Count service unavailable" };
    }
    const payload = (await response.json()) as {
      data?: {
        count?: number;
        queued?: number | null;
        queue?: { status?: string };
      };
    };
    return {
      count: typeof payload.data?.count === "number" ? payload.data.count : 0,
      queued: typeof payload.data?.queued === "number" ? payload.data.queued : null,
      queueStatus: payload.data?.queue?.status === "ok" ? "ok" : "unavailable",
    };
  } catch {
    return { count: 0, queued: null, queueStatus: "unavailable", error: "Gateway unreachable" };
  }
}

export function RootRoute() {
  const data = useLoaderData() as LoaderData;
  const revalidator = useRevalidator();
  const [isQueueing, setIsQueueing] = useState(false);
  const { message, toneClassName, showOk, showError } = useVisitFeedback(data.error ?? "", data.error ? "err" : "ok");

  useEffect(() => {
    if (data.error) {
      showError(data.error);
    }
  }, [data.error, showError]);

  useEffect(() => {
    const id = setInterval(() => {
      revalidator.revalidate();
    }, 5000);
    return () => clearInterval(id);
  }, [revalidator]);

  async function queueVisits(amount: number): Promise<void> {
    if (isQueueing) {
      return;
    }

    setIsQueueing(true);
    showOk(`Queueing ${amount} visit${amount === 1 ? "" : "s"}...`);

    try {
      const response = await fetch(`/api/v1/visit-events?count=${amount}`, { method: "POST" });
      if (!response.ok) {
        showError("Queue request failed");
        return;
      }

      const payload = (await response.json()) as { data?: { queued?: number } };
      const queued = typeof payload.data?.queued === "number" ? payload.data.queued : 0;
      if (queued !== amount) {
        showError(`${queued} of ${amount} visits queued`);
      } else {
        showOk(`${queued} visit${queued === 1 ? "" : "s"} queued`);
      }
    } catch {
      showError("Gateway unreachable");
    } finally {
      setIsQueueing(false);
    }
  }

  return (
    <VisitCounterCard
      count={data.count}
      queued={data.queued}
      queueStatus={data.queueStatus}
      message={message}
      toneClassName={toneClassName}
      isQueueing={isQueueing}
      isRefreshing={revalidator.state === "loading"}
      onQueueOne={() => {
        void queueVisits(1);
      }}
      onQueueTen={() => {
        void queueVisits(10);
      }}
      onRefresh={() => {
        revalidator.revalidate();
      }}
    />
  );
}

function getApiBase(request: Request): string {
  if (typeof window !== "undefined") {
    return "/api/v1";
  }
  const envBase = process.env.VISIT_GATEWAY_URL;
  if (envBase) {
    return envBase;
  }
  const url = new URL(request.url);
  return `${url.protocol}//${url.host}/api/v1`;
}
