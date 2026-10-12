#!/bin/sh
set -eu
cd "$(dirname "$0")"

swag init -g cmd/data-migration/main.go --parseInternal --outputTypes yaml -o docs

python3 - <<'PY'
from pathlib import Path

path = Path("docs/swagger.yaml")
document = path.read_text(encoding="utf-8")
for route, summary in (
    ("/api/methods", "Список опубликованных методов"),
    ("/api/methods/feed", "Метод из ленты"),
):
    start = document.index(f"  {route}:\n")
    end = document.find("\n  /", start + 1)
    if end == -1:
        end = len(document)
    block = document[start:end]
    required = f"      security:\n      - BearerAuth: []\n      summary: {summary}"
    optional = required.replace("      - BearerAuth: []", "      - {}\n      - BearerAuth: []")
    if block.count(required) != 1:
        raise SystemExit(f"Cannot locate GET security for {route}")
    document = document[:start] + block.replace(required, optional, 1) + document[end:]
path.write_text(document, encoding="utf-8")
PY
