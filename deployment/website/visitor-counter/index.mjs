import {
  DynamoDBClient,
  GetItemCommand,
  TransactWriteItemsCommand
} from "@aws-sdk/client-dynamodb";
import { createHandler } from "./logic.mjs";
import { createDynamoStore } from "./store.mjs";

const TABLE_NAME = process.env.TABLE_NAME;
const client = new DynamoDBClient({});
const store = createDynamoStore({
  client,
  commands: { GetItemCommand, TransactWriteItemsCommand },
  tableName: TABLE_NAME
});

export const handler = createHandler({ store });
