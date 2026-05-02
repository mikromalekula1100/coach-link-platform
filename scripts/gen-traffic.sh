#!/usr/bin/env bash
#
# Generate load against the API gateway so the Grafana dashboard fills with
# real RED metrics (Rate, Errors, Duration) for the live defense demo.
#
# Hits a mix of endpoints — including some that 401/404 on purpose — so the
# error-rate and latency panels show meaningful data, not just 200s.
#
# Usage:
#   bash scripts/gen-traffic.sh [iterations]   # default 200
#
set -uo pipefail

BASE="http://localhost:8080"
ITER="${1:-200}"
SUFFIX=$(date +%s | tail -c 6)

bold() { printf "\033[1m%s\033[0m\n" "$1"; }

bold "Генерация трафика: $ITER итераций против $BASE"
bold "Открой дашборд: http://localhost:3001  (Prometheus: http://localhost:9090)"

# Register one real user up front so authed endpoints return 200s too.
REG=$(curl -s -X POST "$BASE/api/v1/auth/register" -H "Content-Type: application/json" \
  -d "{\"login\":\"load-$SUFFIX\",\"email\":\"load$SUFFIX@t.test\",\"password\":\"password123\",\"full_name\":\"Load User\",\"role\":\"coach\"}")
TOK=$(python3 -c "import sys,json;print(json.load(sys.stdin)['access_token'])" <<<"$REG" 2>/dev/null || echo "")

hit() { curl -s -o /dev/null "$@"; }

for i in $(seq 1 "$ITER"); do
  # Healthy 200s across services
  hit "$BASE/health"
  hit "$BASE/api/v1/users/me" -H "Authorization: Bearer $TOK"
  hit "$BASE/api/v1/users/search?q=load&role=coach" -H "Authorization: Bearer $TOK"
  hit "$BASE/api/v1/training/assignments" -H "Authorization: Bearer $TOK"
  hit "$BASE/api/v1/notifications" -H "Authorization: Bearer $TOK"
  hit "$BASE/api/v1/analytics/overview" -H "Authorization: Bearer $TOK"

  # Deliberate 401 (no token) and 400 (bad search) so error panels populate
  hit "$BASE/api/v1/users/me"
  hit "$BASE/api/v1/users/search?q=a" -H "Authorization: Bearer $TOK"

  if (( i % 25 == 0 )); then bold "  ...$i / $ITER"; fi
done

bold "Готово. Метрики собраны Prometheus (scrape каждые 10 c)."
