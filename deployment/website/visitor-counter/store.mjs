function numberAttribute(item, name) {
  const raw = item?.[name]?.N;
  if (typeof raw !== "string") return 0;
  const value = Number(raw);
  return Number.isFinite(value) && value >= 0 ? Math.floor(value) : 0;
}

function cancellationReasons(error) {
  return error?.CancellationReasons || error?.cancellationReasons || [];
}

function isDuplicateMarker(error) {
  if (error?.name !== "TransactionCanceledException") return false;
  const reasons = cancellationReasons(error);
  return reasons?.[0]?.Code === "ConditionalCheckFailed";
}

export function createDynamoStore({ client, commands, tableName }) {
  if (!client?.send) throw new TypeError("DynamoDB client is required");
  if (!commands?.GetItemCommand || !commands?.TransactWriteItemsCommand) throw new TypeError("DynamoDB commands are required");
  if (!tableName) throw new TypeError("visitor counter table name is required");

  async function getTotal() {
    const response = await client.send(new commands.GetItemCommand({
      TableName: tableName,
      Key: { pk: { S: "COUNTER" } },
      ConsistentRead: true,
      ProjectionExpression: "#total",
      ExpressionAttributeNames: { "#total": "total" }
    }));
    return numberAttribute(response?.Item, "total");
  }

  async function recordVisit(marker) {
    try {
      await client.send(new commands.TransactWriteItemsCommand({
        TransactItems: [
          {
            Put: {
              TableName: tableName,
              Item: {
                pk: { S: marker.key },
                expiresAt: { N: String(marker.expiresAt) }
              },
              ConditionExpression: "attribute_not_exists(pk)"
            }
          },
          {
            Update: {
              TableName: tableName,
              Key: { pk: { S: "COUNTER" } },
              UpdateExpression: "ADD #total :one",
              ExpressionAttributeNames: { "#total": "total" },
              ExpressionAttributeValues: { ":one": { N: "1" } }
            }
          }
        ]
      }));
    } catch (error) {
      if (!isDuplicateMarker(error)) throw error;
    }
    return getTotal();
  }

  return { getTotal, recordVisit };
}
