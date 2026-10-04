#!/usr/bin/env python3
"""Verify an unpublished GoReleaser dist directory without executing artifacts."""
import hashlib
import json
from pathlib import Path
import sys
import tarfile
import zipfile


def verify(root):
    root = Path(root)
    checked = set()
    for line in (root / "checksums.txt").read_text().splitlines():
        digest, name = line.split(maxsplit=1)
        name = name.removeprefix("*")
        if Path(name).name != name or name in checked:
            raise ValueError("unsafe or duplicate checksum path")
        path = root / name
        if hashlib.sha256(path.read_bytes()).hexdigest() != digest:
            raise ValueError(f"checksum mismatch: {name}")
        checked.add(name)
    for target in ["linux_amd64", "linux_arm64", "darwin_amd64", "darwin_arm64", "windows_amd64"]:
        extension = ".zip" if target.startswith("windows") else ".tar.gz"
        target_name = target.replace("_", "-")
        candidates = list(root.glob(f"bwenv-*-{target_name}{extension}"))
        if len(candidates) != 1:
            raise ValueError(f"missing or ambiguous archive for {target}")
        archive = candidates[0]
        if extension == ".zip":
            with zipfile.ZipFile(archive) as file:
                names = file.namelist()
            binary = "bwenv.exe"
        else:
            with tarfile.open(archive) as file:
                if any(member.issym() or member.islnk() for member in file.getmembers()):
                    raise ValueError("unexpected archive link")
                names = file.getnames()
            binary = "bwenv"
        if not {binary, "LICENSE", "README.md"}.issubset(names):
            raise ValueError(f"incomplete archive: {archive.name}")
        if any(Path(name).is_absolute() or ".." in Path(name).parts for name in names):
            raise ValueError("unsafe archive path")
        sbom = root / (archive.name + ".sbom.json")
        data = json.loads(sbom.read_text())
        if data.get("bomFormat") != "CycloneDX" or not data.get("components"):
            raise ValueError(f"missing dependency inventory: {sbom.name}")
        if archive.name not in checked or sbom.name not in checked:
            raise ValueError("archive/SBOM not covered by checksums")
    for arch in ["amd64", "arm64"]:
        for extension in ["deb", "rpm"]:
            packages = list(root.glob(f"bwenv_*_{arch}.{extension}"))
            if len(packages) != 1 or packages[0].name not in checked:
                raise ValueError(f"missing checksummed {extension} package for {arch}")
    print(f"Verified five platform archives, five SBOMs and four Linux packages ({len(checked)} checksums)")


if __name__ == "__main__":
    try:
        verify(sys.argv[1])
    except (OSError, ValueError, KeyError, IndexError, tarfile.TarError, zipfile.BadZipFile) as error:
        print(f"Release verification failed: {error}", file=sys.stderr)
        sys.exit(1)
