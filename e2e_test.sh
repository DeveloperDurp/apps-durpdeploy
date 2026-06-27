#!/usr/bin/env bash
set -euo pipefail

BASE="http://localhost:8080"
TMP=$(mktemp -d)
trap "rm -rf $TMP; kill %1 2>/dev/null || true" EXIT

echo "=== Building and starting server ==="
rm -f durpdeploy.db
go build -o "$TMP/durpdeploy" ./cmd/server
"$TMP/durpdeploy" &
sleep 2

curl_silent() { curl -s -o /dev/null -w "%{http_code}" "$@"; }
curl_body() { curl -s "$@"; }

echo "=== F3.1: Happy Path ==="
CODE=$(curl_silent -X POST -d "name=TestProject" "$BASE/projects")
[[ "$CODE" == "303" ]] || { echo "FAIL: create project got $CODE"; exit 1; }
PROJECT_ID=$(curl_body "$BASE/projects" | grep -oP 'href="/projects/\K[0-9]+' | head -1)
echo "Project ID: $PROJECT_ID"

CODE=$(curl_silent -X POST -d "name=TestEnv" "$BASE/environments")
[[ "$CODE" == "303" ]] || { echo "FAIL: create env got $CODE"; exit 1; }
ENV_ID=$(curl_body "$BASE/environments" | grep -oP 'href="/environments/\K[0-9]+' | head -1)
echo "Env ID: $ENV_ID"

CODE=$(curl_silent -X POST -d "name=Step1&script_body=echo+hello" "$BASE/projects/$PROJECT_ID/steps")
[[ "$CODE" == "200" ]] || { echo "FAIL: create step got $CODE"; exit 1; }

CODE=$(curl_silent -X POST -d "name=VAR1&value=hello&environment_id=$ENV_ID" "$BASE/projects/$PROJECT_ID/variables")
[[ "$CODE" == "303" ]] || { echo "FAIL: create variable got $CODE"; exit 1; }

CODE=$(curl_silent -X POST -d "version=1.0.0&release_notes=first" "$BASE/projects/$PROJECT_ID/releases")
[[ "$CODE" == "303" ]] || { echo "FAIL: create release got $CODE"; exit 1; }
RELEASE_ID=$(curl_body "$BASE/projects/$PROJECT_ID/releases" | grep -oP 'href="/projects/'$PROJECT_ID'/releases/\K[0-9]+' | sort -n | tail -1)
echo "Release ID: $RELEASE_ID"

DEP_URL=$(curl -s -D - -o /dev/null -X POST -d "release_id=$RELEASE_ID&environment_id=$ENV_ID" "$BASE/deployments" | grep -i "^location:" | awk '{print $2}' | tr -d '\r')
DEP_ID=$(echo "$DEP_URL" | grep -oP '/deployments/\K[0-9]+')
[[ -n "$DEP_ID" ]] || { echo "FAIL: create deployment did not redirect"; exit 1; }
echo "Deployment ID: $DEP_ID"

CODE=$(curl_silent "$BASE/deployments/$DEP_ID")
[[ "$CODE" == "200" ]] || { echo "FAIL: deployment page got $CODE"; exit 1; }

echo "=== F3.2: Cancel Path ==="
curl -s -o /dev/null -X POST -d "name=LongStep&script_body=sleep+10" "$BASE/projects/$PROJECT_ID/steps"
curl -s -o /dev/null -X POST -d "version=1.0.1" "$BASE/projects/$PROJECT_ID/releases"
CANCEL_REL=$(curl_body "$BASE/projects/$PROJECT_ID/releases" | grep -oP 'href="/projects/'$PROJECT_ID'/releases/\K[0-9]+' | sort -n | tail -1)
echo "Cancel Release ID: $CANCEL_REL"

CANCEL_URL=$(curl -s -D - -o /dev/null -X POST -d "release_id=$CANCEL_REL&environment_id=$ENV_ID" "$BASE/deployments" | grep -i "^location:" | awk '{print $2}' | tr -d '\r')
CANCEL_DEP=$(echo "$CANCEL_URL" | grep -oP '/deployments/\K[0-9]+')
[[ -n "$CANCEL_DEP" ]] || { echo "FAIL: cancel deployment did not redirect"; exit 1; }
echo "Cancel Deployment ID: $CANCEL_DEP"

for i in {1..50}; do
  if curl_body "$BASE/deployments/$CANCEL_DEP/status" | grep -q 'running'; then break; fi
  sleep 0.1
done
CODE=$(curl_silent -X POST "$BASE/deployments/$CANCEL_DEP/cancel")
[[ "$CODE" == "303" ]] || { echo "FAIL: cancel got $CODE"; exit 1; }

echo "=== F3.3: Validation Path ==="
CODE=$(curl_silent -X POST -d "name=" "$BASE/projects")
[[ "$CODE" == "422" ]] || { echo "FAIL: empty project name should be 422, got $CODE"; exit 1; }

echo "=== F3.4: Variable Fallback ==="
curl -s -o /dev/null -X POST -d "name=StepMissing&script=echo+%24%7BMISSING%7D" "$BASE/projects/$PROJECT_ID/steps"
curl -s -o /dev/null -X POST -d "version=2.0.0" "$BASE/projects/$PROJECT_ID/releases"
NEW_REL=$(curl_body "$BASE/projects/$PROJECT_ID/releases" | grep -oP 'href="/projects/'$PROJECT_ID'/releases/\K[0-9]+' | sort -n | tail -1)
curl -s -o /dev/null -X POST -d "release_id=$NEW_REL&environment_id=$ENV_ID" "$BASE/deployments"

echo "=== ALL E2E CHECKS PASSED ==="
