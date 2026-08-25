#!/usr/bin/env python3
"""Generate Ansible inventory from `tofu output -json ansible_inventory`."""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
from typing import Any, Dict


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate Ansible inventory for the lab from OpenTofu JSON output."
    )
    parser.add_argument(
        "--input",
        required=True,
        help="Path to JSON containing the ansible_inventory object",
    )
    parser.add_argument(
        "--output",
        required=True,
        help="Path to write generated inventory YAML",
    )
    parser.add_argument(
        "--region",
        default="eu-central-1",
        help="Default AWS region value written as ansible_aws_ssm_region",
    )
    parser.add_argument(
        "--ssm-bucket",
        default=os.getenv("ANSIBLE_SSM_BUCKET", ""),
        help=(
            "S3 bucket used by Ansible aws_ssm connection plugin for module transfer "
            "(or set ANSIBLE_SSM_BUCKET env var)"
        ),
    )
    return parser.parse_args()


def load_inventory_payload(path: Path) -> Dict[str, Any]:
    data = json.loads(path.read_text(encoding="utf-8"))

    if "control_plane" in data and "workers" in data:
        return data

    if "ansible_inventory" in data:
        payload = data["ansible_inventory"]
        if isinstance(payload, dict) and "value" in payload:
            return payload["value"]
        if isinstance(payload, dict):
            return payload

    outputs = data.get("outputs")
    if isinstance(outputs, dict) and "ansible_inventory" in outputs:
        payload = outputs["ansible_inventory"]
        if isinstance(payload, dict) and "value" in payload:
            return payload["value"]

    raise ValueError(
        "Input JSON must be ansible_inventory object, tofu JSON output, or state JSON containing outputs.ansible_inventory"
    )


def format_host_block(hostname: str, values: Dict[str, Any], region: str) -> str:
    return (
        f"        {hostname}:\n"
        f"          ansible_host: {values['instance_id']}\n"
        f"          ansible_aws_ssm_region: {region}\n"
        f"          private_ip: {values['private_ip']}\n"
        f"          instance_id: {values['instance_id']}\n"
        f"          az: {values['az']}\n"
    )


def build_inventory(payload: Dict[str, Any], region: str, ssm_bucket: str) -> str:
    cp_nodes = payload.get("control_plane", {})
    worker_nodes = payload.get("workers", {})
    kube_api_internal_endpoint = payload.get("kube_api_internal_endpoint")
    kube_api_public_endpoint = payload.get("kube_api_public_endpoint")
    ingress_public_endpoint = payload.get("ingress_public_endpoint") or ""
    gateway_listener_port = payload.get("gateway_listener_port") or 30080

    if not isinstance(cp_nodes, dict) or not isinstance(worker_nodes, dict):
        raise ValueError("control_plane and workers must be maps")

    if not kube_api_internal_endpoint:
        if "cp-1" in cp_nodes and isinstance(cp_nodes["cp-1"], dict):
            kube_api_internal_endpoint = f"{cp_nodes['cp-1']['private_ip']}:6443"
        else:
            first_cp = sorted(cp_nodes.keys())[0]
            kube_api_internal_endpoint = f"{cp_nodes[first_cp]['private_ip']}:6443"

    if not kube_api_public_endpoint:
        kube_api_public_endpoint = kube_api_internal_endpoint

    lines = [
        "---",
        "all:",
        "  vars:",
        "    ansible_connection: amazon.aws.aws_ssm",
        "    ansible_aws_ssm_region: {}".format(region),
        "    ansible_aws_ssm_bucket_name: {}".format(ssm_bucket),
        "    kube_api_internal_endpoint: {}".format(kube_api_internal_endpoint),
        "    kube_api_public_endpoint: {}".format(kube_api_public_endpoint),
        "    ingress_public_endpoint: {}".format(ingress_public_endpoint),
        "    gateway_listener_port: {}".format(gateway_listener_port),
        "    ansible_shell_type: sh",
        "    ansible_shell_executable: /bin/sh",
        "  children:",
        "    control_plane:",
        "      hosts:",
    ]

    for host in sorted(cp_nodes.keys()):
        lines.append(format_host_block(host, cp_nodes[host], region).rstrip("\n"))

    lines.extend(
        [
            "    workers:",
            "      hosts:",
        ]
    )

    for host in sorted(worker_nodes.keys()):
        lines.append(format_host_block(host, worker_nodes[host], region).rstrip("\n"))

    lines.extend(
        [
            "    k8s_cluster:",
            "      children:",
            "        control_plane: {}",
            "        workers: {}",
        ]
    )

    return "\n".join(lines) + "\n"


def main() -> int:
    args = parse_args()
    input_path = Path(args.input)
    output_path = Path(args.output)

    if not args.ssm_bucket:
        raise ValueError(
            "Missing --ssm-bucket (or ANSIBLE_SSM_BUCKET env var). "
            "aws_ssm connection requires an S3 bucket for module transfer."
        )

    payload = load_inventory_payload(input_path)
    inventory_yaml = build_inventory(payload, args.region, args.ssm_bucket)

    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(inventory_yaml, encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
