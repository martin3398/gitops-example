#!/usr/bin/env python3

from __future__ import annotations

import argparse
from pathlib import Path

import yaml


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Rewrite a kubeconfig server and context")
    parser.add_argument("--kubeconfig", required=True, help="Path to the kubeconfig file")
    parser.add_argument("--server", required=True, help="API server URL, including scheme")
    parser.add_argument("--context", default="kubernetes", help="Target cluster/context name")
    return parser.parse_args()


def ensure_item(items: list[dict], name: str, key: str) -> dict:
    for item in items:
        if item.get("name") == name:
            return item.setdefault(key, {})
    item = {"name": name, key: {}}
    items.append(item)
    return item[key]


def main() -> None:
    args = parse_args()
    kubeconfig_path = Path(args.kubeconfig)
    data = yaml.safe_load(kubeconfig_path.read_text())

    clusters = data.setdefault("clusters", [])
    contexts = data.setdefault("contexts", [])
    default_cluster = next((item["cluster"] for item in clusters if item.get("name") == "default"), {})

    cluster = ensure_item(clusters, args.context, "cluster")
    cluster["server"] = args.server
    if "certificate-authority-data" not in cluster and "certificate-authority-data" in default_cluster:
        cluster["certificate-authority-data"] = default_cluster["certificate-authority-data"]

    context = ensure_item(contexts, args.context, "context")
    context["cluster"] = args.context
    context["user"] = "default"

    data["current-context"] = args.context
    kubeconfig_path.write_text(yaml.safe_dump(data, sort_keys=False))


if __name__ == "__main__":
    main()
