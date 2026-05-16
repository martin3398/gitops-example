import express from "express";
import React from "react";
import { renderToString } from "react-dom/server";
import { createStaticHandler, createStaticRouter, StaticRouterProvider, type StaticHandlerContext } from "react-router";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import { routes } from "./router.js";

const listenAddr = process.env.LISTEN_ADDR ?? "3000";
const port = Number(listenAddr.replace(":", ""));
const __dirname = path.dirname(fileURLToPath(import.meta.url));
const { query } = createStaticHandler(routes);
const htmlTemplate = readFileSync(path.join(__dirname, "template.html"), "utf-8");

const app = express();
app.use(express.static(__dirname));

app.get("/healthz", (_req, res) => {
  res.status(200).send("ok");
});

app.get("*", async (req, res) => {
  const request = createFetchRequest(req);
  const context = await query(request);

  if (context instanceof Response) {
    if (context.status >= 300 && context.status < 400) {
      const location = context.headers.get("Location");
      if (location) {
        res.redirect(context.status, location);
        return;
      }
    }
    res.status(context.status).send(await context.text());
    return;
  }

  const router = createStaticRouter(routes, context as StaticHandlerContext);
  const appHtml = renderToString(<StaticRouterProvider router={router} context={context} />);
  res.status(200).send(renderDocument(appHtml));
});

app.listen(port, () => {
  console.log(`visit-ui listening on :${port}`);
});

function renderDocument(appHtml: string): string {
  return htmlTemplate.replace("<!--APP_HTML-->", appHtml);
}

function createFetchRequest(req: express.Request): Request {
  const origin = `${req.protocol}://${req.get("host")}`;
  const url = new URL(req.originalUrl || req.url, origin);

  const headers = new Headers();
  for (const [key, value] of Object.entries(req.headers)) {
    if (Array.isArray(value)) {
      for (const item of value) {
        headers.append(key, item);
      }
    } else if (value) {
      headers.set(key, value);
    }
  }

  return new Request(url, {
    method: req.method,
    headers,
    signal: AbortSignal.timeout(30000),
  });
}
