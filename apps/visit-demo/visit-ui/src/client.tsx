import React from "react";
import { hydrateRoot } from "react-dom/client";
import { VisitCounterClient } from "./components/VisitCounterClient.js";

const root = document.getElementById("app-root");

if (!root) {
  throw new Error("missing app-root element");
}

const initialCount = Number.parseInt(root.dataset.initialCount ?? "0", 10);
const initialMessage = root.dataset.initialMessage ?? "";
const tone = root.dataset.initialTone === "err" ? "err" : "ok";

hydrateRoot(
  root,
  <VisitCounterClient
    initialCount={Number.isFinite(initialCount) ? initialCount : 0}
    initialMessage={initialMessage}
    initialTone={tone}
  />,
);
