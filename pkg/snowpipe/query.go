package snowpipe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type sqlResult struct {
	StatementHandle   string `json:"statementHandle"`
	ResultSetMetaData struct {
		PartitionInfo []json.RawMessage `json:"partitionInfo"`
	} `json:"resultSetMetaData"`
	Data [][]*string `json:"data"`
}

// Query runs one statement through the SQL API (POST /api/v2/statements) and returns every row,
// as text; a nil value is SQL NULL. args fill the ? placeholders. A statement still running
// after the server's wait (202) is polled; the rows of a large result come in partitions.
func (c *Client) Query(ctx context.Context, statement string, args ...string) ([][]*string, error) {
	req := map[string]any{
		"statement": statement,
		"timeout":   60,
		"database":  c.cfg.Database,
		"schema":    c.cfg.Schema,
		"warehouse": c.cfg.Warehouse,
		"role":      c.cfg.Role,
	}
	if len(args) > 0 {
		bindings := map[string]any{}
		for i, a := range args {
			bindings[strconv.Itoa(i+1)] = map[string]string{
				"type":  "TEXT",
				"value": a,
			}
		}
		req["bindings"] = bindings
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	res, status, err := c.statement(ctx, http.MethodPost, "/api/v2/statements", body)
	for err == nil && status == http.StatusAccepted {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
		res, status, err = c.statement(ctx, http.MethodGet, "/api/v2/statements/"+url.PathEscape(res.StatementHandle), nil)
	}
	if err != nil {
		return nil, err
	}

	rows := res.Data
	for p := 1; p < len(res.ResultSetMetaData.PartitionInfo); p++ {
		part, _, err := c.statement(ctx, http.MethodGet, fmt.Sprintf("/api/v2/statements/%s?partition=%d", url.PathEscape(res.StatementHandle), p), nil)
		if err != nil {
			return nil, err
		}
		rows = append(rows, part.Data...)
	}
	return rows, nil
}

// statement sends one SQL API request and decodes its 2xx response.
func (c *Client) statement(ctx context.Context, method, path string, body []byte) (sqlResult, int, error) {
	var res sqlResult
	contentType := ""
	var reader io.Reader // a nil *bytes.Reader would not be a nil io.Reader
	if body != nil {
		contentType = "application/json"
		reader = bytes.NewReader(body)
	}
	resp, err := c.control(ctx, method, path, contentType, reader)
	if err != nil {
		return res, 0, fmt.Errorf("sql api: %w", err)
	}
	data, err := readBody(resp)
	if err != nil {
		return res, resp.StatusCode, err
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return res, resp.StatusCode, fmt.Errorf("sql api: decode response: %w", err)
	}
	return res, resp.StatusCode, nil
}
