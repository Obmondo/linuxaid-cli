#!/usr/bin/env python3
"""Bump the linuxaid-cli version/checksums pinned in the LinuxAid Puppet control repo.

Usage: bump-linuxaid-hiera.py <path-to-LinuxAid-checkout> <version> <checksums.txt>

Updates, per architecture:
  - modules/enableit/common/data/common.yaml                    (amd64, also holds the version pin)
  - modules/enableit/common/data/architectures/armv7l.yaml       (armv7)
  - modules/enableit/common/data/architectures/aarch64.yaml      (arm64)

Keeps only the newest KEEP_VERSIONS checksum entries per file.
"""
import re
import sys
from pathlib import Path

KEEP_VERSIONS = 4

VERSION_KEY = "common::system::openvox::linuxaid_cli::version"
CHECKSUMS_KEY = "common::system::openvox::linuxaid_cli::checksums"

# path (relative to repo root) -> release asset arch suffix
FILES = {
    "modules/enableit/common/data/common.yaml": "amd64",
    "modules/enableit/common/data/architectures/armv7l.yaml": "armv7",
    "modules/enableit/common/data/architectures/aarch64.yaml": "arm64",
}


def parse_checksums(text):
    sums = {}
    for line in text.splitlines():
        line = line.strip()
        if not line:
            continue
        sha, name = line.split(None, 1)
        sums[name] = sha
    return sums


def sha_for_arch(sums, arch, checksums_path):
    suffix = f"_linux_{arch}.tar.gz"
    for name, sha in sums.items():
        if name.endswith(suffix):
            return sha
    raise SystemExit(f"no checksum ending in {suffix!r} found in {checksums_path}")


def bump_checksums_block(text, version, sha, keep=KEEP_VERSIONS):
    pattern = re.compile(
        r"(?P<header>^" + re.escape(CHECKSUMS_KEY) + r":\n)"
        r"(?P<entries>(?:^  \d+\.\d+\.\d+: [0-9a-f]+\n)+)",
        re.MULTILINE,
    )
    m = pattern.search(text)
    if not m:
        raise SystemExit(f"could not find {CHECKSUMS_KEY!r} block")

    entries = re.findall(r"^  (\d+\.\d+\.\d+): ([0-9a-f]+)\n", m.group("entries"), re.MULTILINE)
    entries = [(v, h) for v, h in entries if v != version]
    entries.insert(0, (version, sha))
    entries = entries[:keep]

    new_block = m.group("header") + "".join(f"  {v}: {h}\n" for v, h in entries)
    return text[: m.start()] + new_block + text[m.end() :]


def bump_version_line(text, version):
    pattern = re.compile(r"^(" + re.escape(VERSION_KEY) + r": ).*$", re.MULTILINE)
    if not pattern.search(text):
        return text
    return pattern.sub(lambda m: m.group(1) + version, text, count=1)


def main():
    if len(sys.argv) != 4:
        print(__doc__, file=sys.stderr)
        sys.exit(1)

    repo_root, version, checksums_path = Path(sys.argv[1]), sys.argv[2], sys.argv[3]
    sums = parse_checksums(Path(checksums_path).read_text())

    for rel_path, arch in FILES.items():
        path = repo_root / rel_path
        text = path.read_text()
        sha = sha_for_arch(sums, arch, checksums_path)
        text = bump_checksums_block(text, version, sha)
        text = bump_version_line(text, version)
        path.write_text(text)
        print(f"updated {rel_path} ({arch} -> {sha})")


if __name__ == "__main__":
    main()
