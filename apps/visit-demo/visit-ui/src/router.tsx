import React from "react";
import type { RouteObject } from "react-router";
import { RootRoute, loader as rootLoader } from "./routes/root.js";

export const routes: RouteObject[] = [
  {
    path: "/",
    loader: rootLoader,
    element: <RootRoute />,
  },
];
