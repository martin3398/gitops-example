import express from "express";
import React from "react";
import { renderToString } from "react-dom/server";
import { App } from "./App.js";

const listenAddr = process.env.LISTEN_ADDR ?? "3000";
const port = Number(listenAddr.replace(":", ""));
const gatewayBase = process.env.VISIT_GATEWAY_URL ?? "http://visit-gateway/api";

const app = express();
app.use(express.urlencoded({ extended: false }));

app.get("/healthz", (_req, res) => {
  res.status(200).send("ok");
});

app.get("/", async (req, res) => {
  const notice = typeof req.query.notice === "string" ? req.query.notice : undefined;
  const error = typeof req.query.error === "string" ? req.query.error : undefined;
  const count = await fetchCount();
  const html = renderToString(<App count={count} notice={notice} error={error} />);
  res.setHeader("Content-Type", "text/html; charset=utf-8");
  res.status(200).send(`<!doctype html>${html}`);
});

app.post("/visit", async (_req, res) => {
  try {
    const response = await fetch(`${gatewayBase}/visits`, { method: "POST" });
    if (!response.ok) {
      res.redirect("/?error=Queueing+failed");
      return;
    }
    res.redirect("/?notice=Visit+queued");
  } catch {
    res.redirect("/?error=Gateway+unreachable");
  }
});

async function fetchCount(): Promise<number> {
  try {
    const response = await fetch(`${gatewayBase}/visits/count`);
    if (!response.ok) {
      return 0;
    }
    const payload = (await response.json()) as { count?: number };
    return typeof payload.count === "number" ? payload.count : 0;
  } catch {
    return 0;
  }
}

app.listen(port, () => {
  console.log(`visit-ui listening on :${port}`);
});
