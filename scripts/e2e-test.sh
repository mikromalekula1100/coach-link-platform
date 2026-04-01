#!/usr/bin/env bash
set -euo pipefail

BASE="http://localhost:8080"
PASS=0
FAIL=0
# Unique suffix to avoid login collisions across runs
SUFFIX=$(date +%s | tail -c 6)

green() { printf "\033[32m%s\033[0m\n" "$1"; }
red()   { printf "\033[31m%s\033[0m\n" "$1"; }
bold()  { printf "\033[1m%s\033[0m\n" "$1"; }

json() { python3 -c "import sys,json;print(json.load(sys.stdin)$1)" 2>/dev/null; }

check() {
  local desc="$1" actual="$2" expected="$3"
  if [ "$actual" = "$expected" ]; then
    green "  PASS: $desc"
    PASS=$((PASS + 1))
  else
    red "  FAIL: $desc (got '$actual', expected '$expected')"
    FAIL=$((FAIL + 1))
  fi
}

check_not_empty() {
  local desc="$1" actual="$2"
  if [ -n "$actual" ] && [ "$actual" != "null" ] && [ "$actual" != "None" ]; then
    green "  PASS: $desc"
    PASS=$((PASS + 1))
  else
    red "  FAIL: $desc (empty or null)"
    FAIL=$((FAIL + 1))
  fi
}

bold "========================================="
bold " CoachLink E2E Smoke Test"
bold "========================================="
echo ""

# --- Health Check ---
bold "1. Health Check"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/health")
check "Gateway health" "$STATUS" "200"

# --- Register Coach ---
bold "2. Register Coach"
CR=$(curl -s -X POST "$BASE/api/v1/auth/register" -H "Content-Type: application/json" \
  -d "{\"login\":\"e2e-coach-$SUFFIX\",\"email\":\"coach$SUFFIX@e2e.test\",\"password\":\"password123\",\"full_name\":\"E2E Coach\",\"role\":\"coach\"}")
CT=$(echo "$CR" | json "['access_token']")
CID=$(echo "$CR" | json "['user']['id']")
check_not_empty "Coach token" "$CT"
check_not_empty "Coach ID" "$CID"

# --- Register Athlete ---
bold "3. Register Athlete"
AR=$(curl -s -X POST "$BASE/api/v1/auth/register" -H "Content-Type: application/json" \
  -d "{\"login\":\"e2e-athlete-$SUFFIX\",\"email\":\"athlete$SUFFIX@e2e.test\",\"password\":\"password123\",\"full_name\":\"E2E Athlete\",\"role\":\"athlete\"}")
AT=$(echo "$AR" | json "['access_token']")
AID=$(echo "$AR" | json "['user']['id']")
check_not_empty "Athlete token" "$AT"
check_not_empty "Athlete ID" "$AID"

sleep 2  # wait for NATS user profile sync

# --- Login ---
bold "4. Login"
LR=$(curl -s -X POST "$BASE/api/v1/auth/login" -H "Content-Type: application/json" \
  -d "{\"login\":\"e2e-coach-$SUFFIX\",\"password\":\"password123\"}")
check_not_empty "Login returns token" "$(echo "$LR" | json "['access_token']")"

# --- Profile ---
bold "5. User Profile"
PR=$(curl -s "$BASE/api/v1/users/me" -H "Authorization: Bearer $AT")
ROLE=$(echo "$PR" | json "['role']")
check "Athlete role" "$ROLE" "athlete"

# --- Search ---
bold "6. Search Users"
SR=$(curl -s "$BASE/api/v1/users/search?q=e2e-coach-$SUFFIX&role=coach" -H "Authorization: Bearer $AT")
TOTAL=$(echo "$SR" | json "['pagination']['total_items']")
check "Found coach" "$TOTAL" "1"

# --- Connection Request ---
bold "7. Connection Request"
RR=$(curl -s -X POST "$BASE/api/v1/connections/request" -H "Authorization: Bearer $AT" \
  -H "Content-Type: application/json" -d "{\"coach_id\":\"$CID\"}")
RID=$(echo "$RR" | json "['id']")
RSTATUS=$(echo "$RR" | json "['status']")
check "Request status" "$RSTATUS" "pending"

# --- Accept Request ---
bold "8. Accept Request"
AR2=$(curl -s -X PUT "$BASE/api/v1/connections/requests/$RID/accept" -H "Authorization: Bearer $CT")
RSTATUS2=$(echo "$AR2" | json "['status']")
check "Accepted" "$RSTATUS2" "accepted"

# --- Athletes List ---
bold "9. Coach Athletes"
AL=$(curl -s "$BASE/api/v1/connections/athletes" -H "Authorization: Bearer $CT")
ATOTAL=$(echo "$AL" | json "['pagination']['total_items']")
check "Athletes count" "$ATOTAL" "1"

# --- Create Group ---
bold "10. Create Group"
GR=$(curl -s -X POST "$BASE/api/v1/groups" -H "Authorization: Bearer $CT" \
  -H "Content-Type: application/json" -d '{"name":"E2E Group"}')
GID=$(echo "$GR" | json "['id']")
check_not_empty "Group ID" "$GID"

# --- Add to Group ---
bold "11. Add Athlete to Group"
GM=$(curl -s -X POST "$BASE/api/v1/groups/$GID/members" -H "Authorization: Bearer $CT" \
  -H "Content-Type: application/json" -d "{\"athlete_id\":\"$AID\"}")
GMID=$(echo "$GM" | json "['athlete_id']")
check "Member added" "$GMID" "$AID"

# --- Create Template ---
bold "12. Create Template"
TR=$(curl -s -X POST "$BASE/api/v1/training/templates" -H "Authorization: Bearer $CT" \
  -H "Content-Type: application/json" -d '{"title":"E2E Template","description":"Template desc"}')
TID=$(echo "$TR" | json "['id']")
check_not_empty "Template ID" "$TID"

# --- Create Training Plan ---
bold "13. Create Training Plan"
PR2=$(curl -s -X POST "$BASE/api/v1/training/plans" -H "Authorization: Bearer $CT" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"E2E Training\",\"description\":\"Test training\",\"scheduled_date\":\"2026-04-10\",\"athlete_ids\":[\"$AID\"]}")
ASID=$(echo "$PR2" | json "['assignments'][0]['id']")
check_not_empty "Assignment ID" "$ASID"

# --- Athlete Sees Assignment ---
bold "14. Athlete Assignments"
AA=$(curl -s "$BASE/api/v1/training/assignments" -H "Authorization: Bearer $AT")
AATOTAL=$(echo "$AA" | json "['pagination']['total_items']")
check "Assignment count" "$AATOTAL" "1"

# --- Submit Report ---
bold "15. Submit Report"
RP=$(curl -s -X POST "$BASE/api/v1/training/assignments/$ASID/report" -H "Authorization: Bearer $AT" \
  -H "Content-Type: application/json" \
  -d '{"content":"E2E report","duration_minutes":30,"perceived_effort":5,"max_heart_rate":160,"distance_km":5.0}')
RPID=$(echo "$RP" | json "['id']")
check_not_empty "Report ID" "$RPID"

# --- Coach Views Report ---
bold "16. View Report"
VR=$(curl -s "$BASE/api/v1/training/assignments/$ASID/report" -H "Authorization: Bearer $CT")
VRID=$(echo "$VR" | json "['id']")
check "Report matches" "$VRID" "$RPID"

sleep 1

# --- Notifications ---
bold "17. Notifications"
NR=$(curl -s "$BASE/api/v1/notifications" -H "Authorization: Bearer $CT")
UNREAD=$(echo "$NR" | json "['unread_count']")
check_not_empty "Coach has notifications" "$UNREAD"

# --- Summary ---
echo ""
bold "========================================="
TOTAL=$((PASS + FAIL))
if [ "$FAIL" -eq 0 ]; then
  green " ALL $TOTAL TESTS PASSED"
else
  red " $FAIL/$TOTAL TESTS FAILED"
fi
bold "========================================="

exit $FAIL
