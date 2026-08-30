#!/usr/bin/env bash
set -euo pipefail

REGION="${AWS_REGION:-ap-south-1}"
TABLE_NAME="${VISITOR_TABLE_NAME:-Sudharma-Website-Visitors}"
FUNCTION_NAME="${VISITOR_FUNCTION_NAME:-Sudharma-Website-Visitor-Counter}"
ROLE_NAME="${VISITOR_ROLE_NAME:-Sudharma-Website-Visitor-Counter-Role}"
API_NAME="${VISITOR_API_NAME:-Sudharma-Website-Visitor-Counter}"
POLICY_NAME="SudharmaWebsiteVisitorCounterPolicy"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BUILD_DIR="${REPO_ROOT}/build/website-visitor-counter"
CONFIG_FILE="${REPO_ROOT}/web/public/data/visitor-counter.json"
ZIP_FILE="${BUILD_DIR}/visitor-counter.zip"

mkdir -p "$BUILD_DIR"

if ! aws dynamodb describe-table --region "$REGION" --table-name "$TABLE_NAME" >/dev/null 2>&1; then
  aws dynamodb create-table \
    --region "$REGION" \
    --table-name "$TABLE_NAME" \
    --attribute-definitions AttributeName=pk,AttributeType=S \
    --key-schema AttributeName=pk,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --no-cli-pager >/dev/null
fi
aws dynamodb wait table-exists --region "$REGION" --table-name "$TABLE_NAME"
aws dynamodb update-time-to-live \
  --region "$REGION" \
  --table-name "$TABLE_NAME" \
  --time-to-live-specification Enabled=true,AttributeName=expiresAt \
  --no-cli-pager >/dev/null || true

TRUST_FILE="$(mktemp)"
POLICY_FILE="$(mktemp)"
trap 'rm -f "$TRUST_FILE" "$POLICY_FILE"' EXIT

cat >"$TRUST_FILE" <<'JSON'
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {"Service": "lambda.amazonaws.com"},
    "Action": "sts:AssumeRole"
  }]
}
JSON

if ! aws iam get-role --role-name "$ROLE_NAME" >/dev/null 2>&1; then
  aws iam create-role \
    --role-name "$ROLE_NAME" \
    --assume-role-policy-document "file://${TRUST_FILE}" \
    --description "Execution role for Sudharma website visitor counter" \
    --no-cli-pager >/dev/null
fi

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
TABLE_ARN="arn:aws:dynamodb:${REGION}:${ACCOUNT_ID}:table/${TABLE_NAME}"
cat >"$POLICY_FILE" <<JSON
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"],
      "Resource": "arn:aws:logs:${REGION}:${ACCOUNT_ID}:*"
    },
    {
      "Effect": "Allow",
      "Action": ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem", "dynamodb:TransactWriteItems"],
      "Resource": "${TABLE_ARN}"
    }
  ]
}
JSON
aws iam put-role-policy \
  --role-name "$ROLE_NAME" \
  --policy-name "$POLICY_NAME" \
  --policy-document "file://${POLICY_FILE}" \
  --no-cli-pager >/dev/null
ROLE_ARN="$(aws iam get-role --role-name "$ROLE_NAME" --query 'Role.Arn' --output text)"

rm -f "$ZIP_FILE"
(
  cd "$SCRIPT_DIR"
  zip -q -9 "$ZIP_FILE" index.mjs logic.mjs store.mjs
)

if ! aws lambda get-function --region "$REGION" --function-name "$FUNCTION_NAME" >/dev/null 2>&1; then
  # IAM role propagation can take a few seconds after first creation.
  sleep 8
  aws lambda create-function \
    --region "$REGION" \
    --function-name "$FUNCTION_NAME" \
    --runtime nodejs22.x \
    --handler index.handler \
    --role "$ROLE_ARN" \
    --zip-file "fileb://${ZIP_FILE}" \
    --timeout 5 \
    --memory-size 128 \
    --environment "Variables={TABLE_NAME=${TABLE_NAME}}" \
    --no-cli-pager >/dev/null
else
  aws lambda update-function-code \
    --region "$REGION" \
    --function-name "$FUNCTION_NAME" \
    --zip-file "fileb://${ZIP_FILE}" \
    --no-cli-pager >/dev/null
  aws lambda wait function-updated --region "$REGION" --function-name "$FUNCTION_NAME"
  aws lambda update-function-configuration \
    --region "$REGION" \
    --function-name "$FUNCTION_NAME" \
    --runtime nodejs22.x \
    --handler index.handler \
    --role "$ROLE_ARN" \
    --timeout 5 \
    --memory-size 128 \
    --environment "Variables={TABLE_NAME=${TABLE_NAME}}" \
    --no-cli-pager >/dev/null
fi
aws lambda wait function-active-v2 --region "$REGION" --function-name "$FUNCTION_NAME"
aws lambda wait function-updated-v2 --region "$REGION" --function-name "$FUNCTION_NAME"
FUNCTION_ARN="$(aws lambda get-function-configuration --region "$REGION" --function-name "$FUNCTION_NAME" --query FunctionArn --output text)"

API_ID="$(aws apigatewayv2 get-apis --region "$REGION" --query "Items[?Name=='${API_NAME}'] | [0].ApiId" --output text)"
if [ -z "$API_ID" ] || [ "$API_ID" = "None" ]; then
  API_ID="$(aws apigatewayv2 create-api \
    --region "$REGION" \
    --name "$API_NAME" \
    --protocol-type HTTP \
    --cors-configuration '{"AllowOrigins":["*"],"AllowMethods":["GET","POST","OPTIONS"],"AllowHeaders":["content-type"],"MaxAge":86400}' \
    --query ApiId \
    --output text)"
fi

INTEGRATION_ID="$(aws apigatewayv2 get-integrations --region "$REGION" --api-id "$API_ID" --query "Items[?IntegrationUri=='${FUNCTION_ARN}'] | [0].IntegrationId" --output text)"
if [ -z "$INTEGRATION_ID" ] || [ "$INTEGRATION_ID" = "None" ]; then
  INTEGRATION_ID="$(aws apigatewayv2 create-integration \
    --region "$REGION" \
    --api-id "$API_ID" \
    --integration-type AWS_PROXY \
    --integration-uri "$FUNCTION_ARN" \
    --payload-format-version 2.0 \
    --query IntegrationId \
    --output text)"
fi

ROUTE_ID="$(aws apigatewayv2 get-routes --region "$REGION" --api-id "$API_ID" --query "Items[?RouteKey=='\$default'] | [0].RouteId" --output text)"
if [ -z "$ROUTE_ID" ] || [ "$ROUTE_ID" = "None" ]; then
  aws apigatewayv2 create-route \
    --region "$REGION" \
    --api-id "$API_ID" \
    --route-key '$default' \
    --target "integrations/${INTEGRATION_ID}" \
    --no-cli-pager >/dev/null
else
  aws apigatewayv2 update-route \
    --region "$REGION" \
    --api-id "$API_ID" \
    --route-id "$ROUTE_ID" \
    --target "integrations/${INTEGRATION_ID}" \
    --no-cli-pager >/dev/null
fi

STAGE_NAME="$(aws apigatewayv2 get-stages --region "$REGION" --api-id "$API_ID" --query "Items[?StageName=='\$default'] | [0].StageName" --output text)"
if [ -z "$STAGE_NAME" ] || [ "$STAGE_NAME" = "None" ]; then
  aws apigatewayv2 create-stage \
    --region "$REGION" \
    --api-id "$API_ID" \
    --stage-name '$default' \
    --auto-deploy \
    --default-route-settings ThrottlingBurstLimit=30,ThrottlingRateLimit=15 \
    --no-cli-pager >/dev/null
else
  aws apigatewayv2 update-stage \
    --region "$REGION" \
    --api-id "$API_ID" \
    --stage-name '$default' \
    --auto-deploy \
    --default-route-settings ThrottlingBurstLimit=30,ThrottlingRateLimit=15 \
    --no-cli-pager >/dev/null
fi

STATEMENT_ID="AllowSudharmaVisitorCounterApiInvoke"
SOURCE_ARN="arn:aws:execute-api:${REGION}:${ACCOUNT_ID}:${API_ID}/*/*"
if ! aws lambda get-policy --region "$REGION" --function-name "$FUNCTION_NAME" --query Policy --output text 2>/dev/null | grep -q "$STATEMENT_ID"; then
  aws lambda add-permission \
    --region "$REGION" \
    --function-name "$FUNCTION_NAME" \
    --statement-id "$STATEMENT_ID" \
    --action lambda:InvokeFunction \
    --principal apigateway.amazonaws.com \
    --source-arn "$SOURCE_ARN" \
    --no-cli-pager >/dev/null
fi

ENDPOINT="https://${API_ID}.execute-api.${REGION}.amazonaws.com"
for attempt in $(seq 1 20); do
  if BODY="$(curl -fsS --max-time 5 "$ENDPOINT" 2>/dev/null)" && printf '%s' "$BODY" | jq -e '.total | numbers' >/dev/null 2>&1; then
    break
  fi
  if [ "$attempt" -eq 20 ]; then
    echo "Visitor counter endpoint did not become healthy." >&2
    exit 1
  fi
  sleep 3
done

python3 - "$CONFIG_FILE" "$ENDPOINT" <<'PY'
import json, pathlib, sys
path = pathlib.Path(sys.argv[1])
path.parent.mkdir(parents=True, exist_ok=True)
path.write_text(json.dumps({"endpoint": sys.argv[2]}, indent=2) + "\n", encoding="utf-8")
PY

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  printf 'endpoint=%s\n' "$ENDPOINT" >> "$GITHUB_OUTPUT"
fi
printf 'Visitor counter endpoint configured.\n'
