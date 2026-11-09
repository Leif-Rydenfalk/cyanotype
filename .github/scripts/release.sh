#!/usr/bin/env bash
set -euo pipefail

test -s LICENSE || { echo 'Missing root LICENSE' >&2; exit 1; }

mkdir -p dist
dist_dir="$PWD/dist"
staging=$(mktemp -d "$dist_dir/.release.XXXXXXXX")
trap 'rm -rf "$staging"' EXIT

write_third_party_notices() {
  python3 - "$1" <<'PY'
import json
import re
import subprocess
import sys
from collections import defaultdict
from functools import cache
from pathlib import Path
from urllib.parse import quote

go_root, go_version = subprocess.check_output(
    ["go", "env", "GOROOT", "GOVERSION"], text=True
).splitlines()
go_root = Path(go_root).resolve()
toolchain = (f"Go runtime and standard library {go_version}",
             f"https://go.dev/dl/{go_version}.src.tar.gz")

# Include module and package-local notices, including Go's vendored packages.
notice_name = re.compile(
    r"(?:licen[cs]es?|copying|copyright|notices?|third[-_]party[-_]notices?|patents)"
    r"(?:[._-].*)?", re.IGNORECASE
)
license_name = re.compile(r"(?:licen[cs]es?|copying)(?:[._-].*)?", re.IGNORECASE)


@cache
def notice_files(directory):
    return tuple(
        path for path in sorted(directory.iterdir())
        if path.is_file() and notice_name.fullmatch(path.name)
    )


def escape_module(value):
    # The Go module proxy encodes uppercase letters as ! plus lowercase.
    return quote(re.sub(r"[A-Z]", lambda m: "!" + m[0].lower(), value), safe="/!")


packages = subprocess.check_output(["go", "list", "-deps", "-json", "."], text=True)
decoder = json.JSONDecoder()
notices = defaultdict(set)
components = set()
licensed = set()
offset = 0
nonspace = re.compile(r"\S")
while (match := nonspace.search(packages, offset)) is not None:
    package, offset = decoder.raw_decode(packages, match.start())
    if not package.get("Dir"):
        continue

    module = package.get("Module")
    if module:
        if module.get("Main"):
            continue
        module = module.get("Replace", module)
        if not module.get("Version") or not module.get("Dir"):
            raise SystemExit(f"Cannot locate versioned source for {package['ImportPath']}")
        name, version = module["Path"], module["Version"]
        component = (
            f"{name} {version}",
            f"https://proxy.golang.org/{escape_module(name)}/@v/{escape_module(version)}.zip",
        )
        root = Path(module["Dir"]).resolve()
    elif package.get("Standard") or package.get("Goroot"):
        component, root = toolchain, go_root
    else:
        raise SystemExit(f"Cannot locate source module for {package['ImportPath']}")

    components.add(component)
    directory = Path(package["Dir"]).resolve()
    if not directory.is_relative_to(root):
        raise SystemExit(f"Package directory outside its source root: {directory}")
    for parent in (directory, *directory.parents):
        for path in notice_files(parent):
            notices[path.read_bytes()].add((component[0], str(path.relative_to(root))))
            if license_name.fullmatch(path.name):
                licensed.add(component)
        if parent == root:
            break

if missing := components - licensed:
    raise SystemExit("Missing license files for: " + ", ".join(name for name, _ in sorted(missing)))

with Path(sys.argv[1]).open("wb") as output:
    output.write(b"Cyanotype CLI third-party notices\n\n")
    output.write(b"Corresponding upstream source archives are listed below.\n\n")
    for name, source in sorted(components):
        output.write(f"{name}\nSource: {source}\n\n".encode())
    for contents, locations in sorted(notices.items(), key=lambda entry: sorted(entry[1])):
        output.write(b"========================================================================\n")
        for name, path in sorted(locations):
            output.write(f"{name} — {path}\n".encode())
        output.write(b"\n")
        output.write(contents)
        output.write(b"\n" if contents.endswith(b"\n") else b"\n\n")
PY
}

targets=(
  linux/amd64
  linux/arm64
  darwin/amd64
  darwin/arm64
  windows/amd64
)

for target in "${targets[@]}"; do
  os="${target%/*}"
  arch="${target#*/}"
  name="cyanotype_${GITHUB_REF_NAME}_${os}_${arch}"

  exe="cyanotype"
  if [[ "$os" == "windows" ]]; then
    exe="cyanotype.exe"
  fi

  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    go build -trimpath \
      -ldflags="-s -w -X github.com/tychonis/cyanotype/internal/version.Version=${GITHUB_REF_NAME}" \
      -o "$staging/$exe" .

  cp LICENSE "$staging/LICENSE"
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    write_third_party_notices "$staging/THIRD_PARTY_NOTICES.txt"

  files=("$exe" LICENSE THIRD_PARTY_NOTICES.txt)

  if [[ "$os" == "windows" ]]; then
    (cd "$staging" && zip "$dist_dir/${name}.zip" "${files[@]}")
  else
    tar -C "$staging" -czf "$dist_dir/${name}.tar.gz" "${files[@]}"
  fi

  rm -f "$staging/$exe"
done

(cd dist && sha256sum *.tar.gz *.zip > checksums.txt)
