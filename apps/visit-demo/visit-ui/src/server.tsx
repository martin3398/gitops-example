import express from "express";
import React from "react";
import { renderToString } from "react-dom/server";
import { App } from "./App.js";
import type { Response } from "express";
import path from "node:path";
import { fileURLToPath } from "node:url";

const listenAddr = process.env.LISTEN_ADDR ?? "3000";
const port = Number(listenAddr.replace(":", ""));
const gatewayBase = process.env.VISIT_GATEWAY_URL ?? "http://visit-gateway/api/v1";
const __dirname = path.dirname(fileURLToPath(import.meta.url));

const app = express();
app.use(express.static(__dirname));

app.get("/healthz", (_req, res) => {
  res.status(200).send("ok");
});

app.get("/", async (req, res) => {
  const { count, error } = await fetchCount();
  if (error) {
    renderPage(res, count, undefined, error, httpStatus.ServiceUnavailable);
    return;
  }
  renderPage(res, count);
});

async function fetchCount(): Promise<{ count: number; error?: string }> {
  try {
    const response = await fetch(`${gatewayBase}/visits/count`);
    if (!response.ok) {
      return { count: 0, error: "Count service unavailable" };
    }
    const payload = (await response.json()) as { data?: { count?: number } };
    return { count: typeof payload.data?.count === "number" ? payload.data.count : 0 };
  } catch {
    return { count: 0, error: "Gateway unreachable" };
  }
}

const httpStatus = {
  Ok: 200,
  BadRequest: 400,
  ServiceUnavailable: 503,
} as const;

function renderPage(res: Response, count: number, notice?: string, error?: string, status: number = httpStatus.Ok): void {
  const html = renderToString(<App count={count} notice={notice} error={error} />);
  res.setHeader("Content-Type", "text/html; charset=utf-8");
  res.status(status).send(`<!doctype html>${html}`);
}

app.listen(port, () => {
  console.log(`visit-ui listening on :${port}`);
});
