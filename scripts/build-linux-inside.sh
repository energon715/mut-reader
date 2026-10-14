#!/usr/bin/env bash
set -euo pipefail
cd /work
[ "$(dpkg --print-architecture)" = amd64 ]
dpkg --compare-versions "$(pkg-config --modversion webkit2gtk-4.1)" ge 2.54.0 || { echo 'WebKitGTK >= 2.54.0 required; refusing to bundle an older engine' >&2; exit 1; }
(cd frontend && npm ci && npm run check && npm run build && npm audit --audit-level=high)
go vet -tags webkit2_41 ./...
go test -race -tags webkit2_41 ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -tags webkit2_41 ./...
go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0
wails generate module -tags webkit2_41
wails build -tags webkit2_41 -skipbindings -o mut
go build -trimpath -o build/bin/vaultctl ./cmd/vaultctl
python3 scripts/collect-licenses.py
python3 scripts/package-linux.py
python3 scripts/package-appimage.py
python3 scripts/collect-runtime-sources.py
python3 scripts/finalize-release.py
python3 scripts/verify-package.py
