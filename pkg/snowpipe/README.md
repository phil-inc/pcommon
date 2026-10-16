## Snowpipe Streaming

A client for Snowpipe Streaming (high-performance architecture) over its REST API, plus the Snowflake SQL API for setup queries. It authenticates with a key-pair JWT that it signs itself.

Rows go to each pipe's Elastic Channel. A 200 from Snowflake means the rows are stored durably. Delivery is at-least-once and unordered: a retry can store a row twice, so give each row a stable id and deduplicate downstream.

## Prerequisites
* A Snowflake service user with an RSA public key set
* A role with USAGE on the database and schema, INSERT on the tables and OPERATE on the pipes
* Each table's pipe, for example `CREATE PIPE T AS COPY INTO T FROM TABLE(DATA_SOURCE(TYPE => 'STREAMING')) MATCH_BY_COLUMN_NAME = CASE_INSENSITIVE`

## Uses

```
go get github.com/phil-inc/pcommon
```

```go
c, err := snowpipe.New(snowpipe.Config{
	Account:    "myorg-myaccount",
	User:       "SVC_USER",
	PrivateKey: pemKey, // unencrypted PKCS#8 or PKCS#1
	Database:   "DB",
	Schema:     "RAW",
	NumWorkers: 4,
})
if err != nil {
	return err
}
defer c.Close(context.Background())

// One table.
err = c.AppendRows(ctx, "RX_ORDER", []map[string]any{{"id": "a1", "status": "SHIPPED"}})

// Several tables in one call; they are sent together.
err = c.AppendTables(ctx, map[string][]map[string]any{
	"ROR_EVENT":  {{"id": "e1"}},
	"RX_COMMENT": {{"id": "c1"}},
})
```

`AppendRows` and `AppendTables` return once Snowflake has stored every row, so a caller can commit its checkpoint right after. An error means some rows may not be stored. Retrying can store the others twice.
- `ErrTooLarge`: a row is over `MaxRow`. It is reported before anything is sent.
- `*StatusError`: a 4xx other than 401, 408 or 429. This usually means a missing pipe, a missing privilege or a bad request.
- `ctx.Err()`: the caller's deadline passed.

## What the client does

* **Worker pool.** `NumWorkers` goroutines (default 1) send for every pipe together. They start on the first append and stop at `Close`. Load never adds goroutines: requests get bigger and queues get longer instead. `MaxInflightPerPipe` (default 4) keeps one busy pipe from taking every worker.
* **Throughput.** Each worker sends one request at a time, about 2 requests/s at a ~0.5 s round trip. Size `NumWorkers` to the rate you need. In a 60 s test with 57 tables, throughput grew linearly: 1 worker did about 2.3k rows/s and 8 workers about 23.6k rows/s.
* **Requests.**
  * Rows queued for one pipe, including rows from concurrent calls, share one request.
  * Requests are compressed with zstd and sized to land just under Snowflake's 4 MB cap. A request still over the cap is split on a row boundary.
* **Retries.**
  * 408, 429, 5xx and network errors are retried with jittered backoff and `Retry-After`. Every attempt reuses one `requestId` with an increasing `retryCount`.
  * A 401 renews the scoped token.
  * Retries stop after `RetryFor` (default 1 minute), or once every call waiting on the request has returned.
* **Rows.** Rows are encoded and checked in the caller's goroutine before anything is sent.

## Timestamps

Do not stamp rows with a `CURRENT_TIMESTAMP()` column default. In a streaming pipe it is evaluated when the pipe compiles, not per row. Send the time from the client. If you need commit time, set `ROW_TIMESTAMP = TRUE` on the table and read `METADATA$ROW_LAST_COMMIT_TIME`.
