#!/usr/bin/env bash
set -u
B=http://localhost:8080
code() { printf '%-55s -> %s\n' "$2" "$1"; }

c=$(curl -s -o /dev/null -w '%{http_code}' "$B/health"); code "$c" "GET  /health"

r=$(curl -s -w '\n%{http_code}' -X POST "$B/users" -d '{"name":"alice"}')
c=$(echo "$r" | tail -1); body=$(echo "$r" | sed '$d')
code "$c" "POST /users (create alice)"
USERID=$(echo "$body" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
echo "   user_id=$USERID"

c=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$B/users" -d '{"name":"ab"}')
code "$c" "POST /users (too short, expect 400)"

START=$(date -u -v+1H '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -u -d '+1 hour' '+%Y-%m-%dT%H:%M:%SZ')
END=$(date -u -v+2H '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -u -d '+2 hour' '+%Y-%m-%dT%H:%M:%SZ')
r=$(curl -s -w '\n%{http_code}' -X POST "$B/timeslots" \
  -d "{\"user_id\":\"$USERID\",\"duration_mins\":30,\"availability\":[{\"availability_time_start\":\"$START\",\"availability_time_to\":\"$END\"}]}")
c=$(echo "$r" | tail -1); body=$(echo "$r" | sed '$d')
code "$c" "POST /timeslots (60min window, 30min slots)"
echo "   $body"
TS1=$(echo "$body" | sed -n 's/.*\["\([^"]*\)".*/\1/p')

c=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$B/timeslots" \
  -d "{\"user_id\":\"$USERID\",\"duration_mins\":90,\"availability\":[{\"availability_time_start\":\"$START\",\"availability_time_to\":\"$END\"}]}")
code "$c" "POST /timeslots (duration 90, expect 400)"

DS=$(date -u -v-1H '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -u -d '-1 hour' '+%Y-%m-%dT%H:%M:%SZ')
DE=$(date -u -v+1d '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -u -d '+1 day' '+%Y-%m-%dT%H:%M:%SZ')
r=$(curl -s -w '\n%{http_code}' "$B/timeslots/allotted?user_id=$USERID&date_start=$DS&date_end=$DE")
c=$(echo "$r" | tail -1); body=$(echo "$r" | sed '$d')
code "$c" "GET  /timeslots/allotted"
echo "   $body"

c=$(curl -s -o /dev/null -w '%{http_code}' -X PATCH "$B/timeslots/$TS1" \
  -d "{\"time_start\":\"$START\",\"time_end\":\"$END\",\"duration_mins\":20}")
code "$c" "PATCH /timeslots/{id} (unbooked, expect 204)"

c=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$B/timeslots/meeting" \
  -d "{\"id\":\"$TS1\",\"title\":\"Standup\",\"descr\":\"daily\",\"url\":\"http://meet/x\",\"attendees\":[\"a@x.com\",\"b@x.com\"]}")
code "$c" "POST /timeslots/meeting (book TS1)"

c=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$B/timeslots/meeting" -d "{\"id\":\"$TS1\",\"title\":\"dup\"}")
code "$c" "POST /timeslots/meeting (double-book, expect 409)"

c=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE "$B/timeslots/$TS1")
code "$c" "DELETE /timeslots/{id} (booked, expect 409)"

c=$(curl -s -o /dev/null -w '%{http_code}' -X PATCH "$B/timeslots/$TS1" \
  -d "{\"time_start\":\"$START\",\"time_end\":\"$END\",\"duration_mins\":15}")
code "$c" "PATCH /timeslots/{id} (booked, expect 409)"

r=$(curl -s -w '\n%{http_code}' "$B/timeslots/calendar?user_id=$USERID")
c=$(echo "$r" | tail -1); body=$(echo "$r" | sed '$d')
code "$c" "GET  /timeslots/calendar"
echo "   $body"
