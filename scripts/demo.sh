#!/usr/bin/env bash
#
# CoachLink end-to-end demonstration.
#
# Runs the full coach→athlete→report flow through the API gateway, then asks
# the AI service for a real training recommendation and prints the raw Ollama
# output. Unlike e2e-test.sh (which only asserts pass/fail), this script is
# meant to be SHOWN: every step echoes the actual request and response so the
# result is visible — including the generated AI recommendation text.
#
# Prereqs: `make up` (all services healthy, Ollama model pulled).
# Usage:   bash scripts/demo.sh
#
set -euo pipefail

BASE="http://localhost:8080"
SUFFIX=$(date +%s | tail -c 6)

green() { printf "\033[32m%s\033[0m\n" "$1"; }
cyan()  { printf "\033[36m%s\033[0m\n" "$1"; }
bold()  { printf "\033[1m%s\033[0m\n" "$1"; }
dim()   { printf "\033[2m%s\033[0m\n" "$1"; }

json() { python3 -c "import sys,json;print(json.load(sys.stdin)$1)" 2>/dev/null; }
pretty() { python3 -m json.tool 2>/dev/null || cat; }

step() {
  echo ""
  bold "─────────────────────────────────────────────────"
  bold " $1"
  bold "─────────────────────────────────────────────────"
}

bold "========================================="
bold " CoachLink — живая демонстрация работы"
bold "========================================="
dim  " Полный цикл: регистрация → связь тренер-спортсмен →"
dim  " план → отчёт → AI-рекомендация (реальный вывод LLM)"

# --- 1. Register coach + athlete ---
step "1. Регистрация тренера и спортсмена"
CR=$(curl -s -X POST "$BASE/api/v1/auth/register" -H "Content-Type: application/json" \
  -d "{\"login\":\"demo-coach-$SUFFIX\",\"email\":\"coach$SUFFIX@demo.test\",\"password\":\"password123\",\"full_name\":\"Демо Тренер\",\"role\":\"coach\"}")
CT=$(echo "$CR" | json "['access_token']")
CID=$(echo "$CR" | json "['user']['id']")
green "  Тренер зарегистрирован (id=$CID)"

AR=$(curl -s -X POST "$BASE/api/v1/auth/register" -H "Content-Type: application/json" \
  -d "{\"login\":\"demo-athlete-$SUFFIX\",\"email\":\"athlete$SUFFIX@demo.test\",\"password\":\"password123\",\"full_name\":\"Демо Спортсмен\",\"role\":\"athlete\"}")
AT=$(echo "$AR" | json "['access_token']")
AID=$(echo "$AR" | json "['user']['id']")
green "  Спортсмен зарегистрирован (id=$AID)"

sleep 2  # NATS user-profile sync

# --- 2. Connection request + accept ---
step "2. Спортсмен отправляет заявку, тренер принимает"
RR=$(curl -s -X POST "$BASE/api/v1/connections/request" -H "Authorization: Bearer $AT" \
  -H "Content-Type: application/json" -d "{\"coach_id\":\"$CID\"}")
RID=$(echo "$RR" | json "['id']")
green "  Заявка создана (status=$(echo "$RR" | json "['status']"))"
curl -s -X PUT "$BASE/api/v1/connections/requests/$RID/accept" -H "Authorization: Bearer $CT" >/dev/null
green "  Заявка принята — связь тренер↔спортсмен установлена"

# --- 3. Training plan ---
step "3. Тренер назначает план тренировки"
PR2=$(curl -s -X POST "$BASE/api/v1/training/plans" -H "Authorization: Bearer $CT" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Интервальная работа 8x400м\",\"description\":\"Разминка 15 мин, 8x400м через 200м трусцой, заминка\",\"scheduled_date\":\"2026-06-10\",\"athlete_ids\":[\"$AID\"]}")
ASID=$(echo "$PR2" | json "['assignments'][0]['id']")
green "  План назначен (assignment id=$ASID)"

# --- 4. Several reports (give the AI real data to analyse) ---
step "4. Спортсмен сдаёт отчёты о тренировках"
submit_report() {
  curl -s -X POST "$BASE/api/v1/training/assignments/$1/report" -H "Authorization: Bearer $AT" \
    -H "Content-Type: application/json" \
    -d "{\"content\":\"$2\",\"duration_minutes\":$3,\"perceived_effort\":$4,\"max_heart_rate\":$5,\"distance_km\":$6}" >/dev/null
}
submit_report "$ASID" "Выполнил полностью, ноги тяжёлые на последних отрезках" 52 8 182 6.4
green "  Отчёт сдан: 52 мин, RPE 8, max HR 182, 6.4 км"

# A couple more assignments+reports to build a history for the analytics/AI layer.
for i in 1 2; do
  P=$(curl -s -X POST "$BASE/api/v1/training/plans" -H "Authorization: Bearer $CT" \
    -H "Content-Type: application/json" \
    -d "{\"title\":\"Темповый бег $i\",\"description\":\"Темповый бег 5 км\",\"scheduled_date\":\"2026-06-1$i\",\"athlete_ids\":[\"$AID\"]}")
  AS=$(echo "$P" | json "['assignments'][0]['id']")
  submit_report "$AS" "Темп ровный, дыхание под контролем" $((40 + i * 3)) $((6 + i)) $((168 + i * 4)) 5.0
  green "  Отчёт сдан: темповый бег #$i"
done

sleep 1

# --- 5. Analytics summary ---
step "5. Аналитика по спортсмену (агрегаты для AI)"
STATS=$(curl -s "$BASE/api/v1/analytics/athletes/$AID/summary" -H "Authorization: Bearer $CT")
echo "$STATS" | pretty || true

# --- 6. AI recommendation — the visible result ---
step "6. AI-рекомендация (реальный ответ Ollama / gemma3:4b)"
cyan "  Запрос:"
dim  "  POST $BASE/api/v1/ai/athletes/$AID/recommendations"
echo ""
cyan "  Ответ (это может занять 30–180 c — локальная LLM):"
AIRESP=$(curl -s -X POST "$BASE/api/v1/ai/athletes/$AID/recommendations" \
  -H "Authorization: Bearer $CT" -H "Content-Type: application/json" -d '{}')

# Print the raw recommendation text + which model produced it.
echo ""
bold "  ── Рекомендация ───────────────────────────────"
echo "$AIRESP" | json "['content']" || echo "$AIRESP"
bold "  ───────────────────────────────────────────────"
MODEL=$(echo "$AIRESP" | json "['model']" || echo "?")
GEN=$(echo "$AIRESP" | json "['generated_at']" || echo "?")
dim "  model=$MODEL  generated_at=$GEN"

echo ""
bold "========================================="
green " Демонстрация завершена успешно."
dim  " Открыть метрики под нагрузкой: make grafana"
bold "========================================="
