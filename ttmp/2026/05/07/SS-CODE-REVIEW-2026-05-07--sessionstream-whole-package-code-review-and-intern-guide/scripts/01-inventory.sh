#!/usr/bin/env bash
set -euo pipefail
printf '== go packages ==\n'
go list ./...
printf '\n== non-generated source files ==\n'
rg --files -g '!dist/**' -g '!ttmp/**' | sort
printf '\n== top line counts ==\n'
find . -path './ttmp' -prune -o -path './dist' -prune -o -type f \( -name '*.go' -o -name '*.js' -o -name '*.css' -o -name '*.html' -o -name '*.md' -o -name '*.proto' \) -print0 | xargs -0 wc -l | sort -nr | head -80
printf '\n== TODO/FIXME/deprecated markers ==\n'
rg -n 'TODO|FIXME|Deprecated|deprecated|legacy|obsolete|HACK|XXX|panic\(' . --glob '!ttmp/**' --glob '!dist/**' || true
printf '\n== recent commits ==\n'
git log --oneline --decorate -30
