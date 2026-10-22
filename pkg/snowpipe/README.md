## Snowpipe Streaming Implementation
A client for Snowpipe Streaming (high-performance architecture) over its REST API. Rows go to each pipe's Elastic Channel. `AppendRows` and `AppendTables` return once Snowflake has stored the rows. Retries, token renewal, compression and request sizing happen inside.

Delivery is at-least-once: a retry can store a row twice, so give rows a stable id. The client does not panic: failures come back as errors, an `*AppendError` listing each failed pipe, and errors never hold row values.

## Prerequisites
* Snowflake account, user and unencrypted RSA private key (PKCS#8 or PKCS#1)
* A pipe per table
* A role with USAGE on the database and schema, INSERT on the table and OPERATE on the pipe

## Uses

To start working with this dependency, you need to retrieve the dependency in your Go project with the following command.

```
go get github.com/phil-inc/pcommon
```

Example Code:

```
package main

import (
	"context"
	"errors"
	"log"

	"github.com/phil-inc/pcommon/pkg/snowpipe"
)

func main() {
	client, err := snowpipe.New(snowpipe.Config{
		Account:    "myorg-myaccount",
		User:       "SVC_USER",
		PrivateKey: privateKeyPEM,
		Database:   "DB",
		Schema:     "RAW",
		NumWorkers: 4, // sender goroutines for all pipes; default 1
	})
	if err != nil {
		panic(err)
	}
	defer client.Close(context.Background())

	ctx := context.Background()
	err = client.AppendTables(ctx, map[string][]map[string]any{
		"ROR_EVENT":  {{"id": "e1", "operation_type": "insert"}},
		"RX_COMMENT": {{"id": "c1", "operation_type": "insert"}},
	})
	if err != nil {
		// some rows may not be stored; retrying can duplicate the others
		var ae *snowpipe.AppendError
		if errors.As(err, &ae) {
			for _, p := range ae.Pipes {
				log.Printf("%s: %d rows not stored: %v", p.Pipe, p.Rows, p.Err)
			}
		}
	}
}
```
