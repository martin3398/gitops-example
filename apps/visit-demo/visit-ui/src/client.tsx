import React from "react";
import { hydrateRoot } from "react-dom/client";
import { createBrowserRouter } from "react-router";
import { RouterProvider } from "react-router/dom";
import { routes } from "./router.js";

declare global {
  interface Window {
    __staticRouterHydrationData?: unknown;
  }
}

const root = document.getElementById("app-root");

if (!root) {
  throw new Error("missing app-root element");
}

const router = createBrowserRouter(routes, {
  hydrationData: window.__staticRouterHydrationData as any,
});

hydrateRoot(root, <RouterProvider router={router} />);
