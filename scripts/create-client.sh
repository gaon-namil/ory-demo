#!/usr/bin/env bash
set -euo pipefail

ADMIN_URL="${HYDRA_ADMIN_URL:-http://localhost:4445}"
CLIENT_ID="${CLIENT_ID:-demo-client}"
CLIENT_SECRET="${CLIENT_SECRET:-demo-secret}" # デモ用。本番ではSecret Storeへ。
REDIRECT_URI="${REDIRECT_URI:-http://localhost:8080/callback}"

echo "Creating OAuth2 client..."
curl -sS -X POST "${ADMIN_URL}/admin/clients" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"${CLIENT_ID}\",
    \"client_secret\": \"${CLIENT_SECRET}\",
    \"grant_types\": [\"authorization_code\", \"refresh_token\"],
    \"response_types\": [\"code\"],
    \"redirect_uris\": [\"${REDIRECT_URI}\"],
    \"scope\": \"openid offline_access\",
    \"token_endpoint_auth_method\": \"client_secret_basic\"
  }" | jq .
echo "Done."