#!/usr/bin/env python3
"""Measure an explicitly selected project; persist only timing/count metadata."""
import argparse
import json
from pathlib import Path
import platform
import re
import shutil
import statistics
import subprocess
import sys
import tomllib


def measure(binary, project):
    result = subprocess.run([binary, "benchmark"], cwd=project, capture_output=True, text=True, timeout=180)
    if result.returncode:
        raise RuntimeError("Benchmark failed; check bwenv status/login in the selected project")
    def number(pattern):
        match = re.search(pattern, result.stdout, re.MULTILINE)
        if not match:
            raise RuntimeError("Unexpected benchmark report")
        return float(match.group(1))
    return {"total_ms": number(r"^total\s+([0-9.]+) ms$"),
            "provider_processes": int(number(r"^provider processes: (\d+)$")),
            "variables": int(number(r"^variables found: (\d+)$"))}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("project", type=Path)
    parser.add_argument("--binary", default="bwenv")
    parser.add_argument("--runs", type=int, default=5)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if not 1 <= args.runs <= 20:
        parser.error("--runs must be between 1 and 20")
    project = args.project.resolve()
    config = tomllib.loads((project / ".bwenv.toml").read_text())
    if config.get("activation", {}).get("disabled"):
        raise RuntimeError("Selected project is disabled; explicitly enable it before measuring")
    provider = config["provider"]
    cli = {"bitwarden": "bw", "1password": "op"}.get(provider)
    if not cli:
        raise RuntimeError("Unsupported provider for the measurement protocol")
    binary = shutil.which(args.binary)
    if not binary:
        raise RuntimeError("bwenv binary not found")
    version = subprocess.run([cli, "--version"], capture_output=True, text=True, timeout=20, check=True).stdout.strip()
    if not re.fullmatch(r"[0-9A-Za-z.+-]{1,80}", version):
        raise RuntimeError("Unexpected provider version output; refusing to record it")
    first = measure(binary, project)
    warm = [measure(binary, project) for _ in range(args.runs)]
    report = {"schema_version": 1, "os": platform.system(), "os_release": platform.release(),
              "architecture": platform.machine(), "provider": provider, "provider_cli_version": version,
              "selected_item_count": len(config["project"]["items"]) if config["project"].get("items") else None,
              "folder_id_fast_path": bool(config["project"].get("folder_id")),
              "first_authenticated_run": first, "warm_runs": warm,
              "warm_median_ms": statistics.median(run["total_ms"] for run in warm),
              "cold_definition": "First authenticated request; no vault/OS cache purge",
              "timing_gate": False}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2) + "\n")
    print(f"Saved timing/count metadata to {args.output}; warm median {report['warm_median_ms']:.1f} ms")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, RuntimeError, subprocess.SubprocessError):
        print("Measurement failed. Check the selected project, binary and authenticated provider session. No provider payload was recorded.", file=sys.stderr)
        sys.exit(1)
