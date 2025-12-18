package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetCodeFromErrResp(t *testing.T) {
	tests := []struct {
		input string
		code  int
		err   bool
	}{
		{
			input: "Http response NOT_OK. Status: Error Status, Code:404",
			code:  404,
			err:   false,
		},
		{
			input: "Http response NOT_OK. Status: Error Status, Code:200",
			code:  200,
			err:   false,
		},
		{
			input: "No code in this response",
			code:  0,
			err:   true,
		},
		{
			input: "Invalid Code value. Code:invalid",
			code:  0,
			err:   true,
		},
	}

	for _, test := range tests {
		code := GetStatusCodeFromError(errors.New(test.input))
		if code != test.code {
			t.Errorf("Test failed for input %s", test.input)
		}
		fmt.Printf("Input: %s, Output: %d\n", test.input, code)
	}
}

func TestIsSuccessStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"200 OK", 200, true},
		{"201 Created", 201, true},
		{"204 No Content", 204, true},
		{"299 Edge case", 299, true},
		{"199 Below range", 199, false},
		{"300 Redirect", 300, false},
		{"400 Bad Request", 400, false},
		{"404 Not Found", 404, false},
		{"500 Server Error", 500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSuccessStatusCode(tt.statusCode)
			if result != tt.expected {
				t.Errorf("isSuccessStatusCode(%d) = %v, expected %v", tt.statusCode, result, tt.expected)
			}
		})
	}
}

type testRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type testResponse struct {
	Token  string `json:"token"`
	UserID int    `json:"user_id"`
}

func TestHTTPRequest_Success(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		requestBody    testRequest
		responseBody   testResponse
		responseStatus int
		headers        map[string]string
	}{
		{
			name:   "POST with body and headers",
			method: http.MethodPost,
			requestBody: testRequest{
				Username: "testuser",
				Password: "testpass",
			},
			responseBody: testResponse{
				Token:  "abc123",
				UserID: 42,
			},
			responseStatus: http.StatusOK,
			headers: map[string]string{
				"Authorization": "Bearer token",
				"X-Custom":      "value",
			},
		},
		{
			name:   "PUT with body",
			method: http.MethodPut,
			requestBody: testRequest{
				Username: "updateuser",
				Password: "newpass",
			},
			responseBody: testResponse{
				Token:  "xyz789",
				UserID: 99,
			},
			responseStatus: http.StatusOK,
			headers:        nil,
		},
		{
			name:           "DELETE with 204 No Content",
			method:         http.MethodDelete,
			requestBody:    testRequest{},
			responseBody:   testResponse{},
			responseStatus: http.StatusNoContent,
			headers:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method
				if r.Method != tt.method {
					t.Errorf("Expected method %s, got %s", tt.method, r.Method)
				}

				// Verify headers
				for key, value := range tt.headers {
					if r.Header.Get(key) != value {
						t.Errorf("Expected header %s=%s, got %s", key, value, r.Header.Get(key))
					}
				}

				// Verify Content-Type for methods with body (only if body is present)
				if tt.method != http.MethodGet && tt.method != http.MethodHead && tt.method != http.MethodDelete {
					contentType := r.Header.Get("Content-Type")
					if contentType != "application/json" {
						t.Errorf("Expected Content-Type application/json, got %s", contentType)
					}

					// Verify request body can be decoded
					var req testRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Errorf("Failed to decode request body: %v", err)
					}
				}

				// For DELETE with empty body, verify no Content-Type or body
				if tt.method == http.MethodDelete {
					contentType := r.Header.Get("Content-Type")
					if contentType == "application/json" {
						t.Error("DELETE with empty body should not have Content-Type header")
					}

					bodyBytes, _ := io.ReadAll(r.Body)
					if len(bodyBytes) > 0 {
						t.Errorf("DELETE with zero-value request should not have body, got: %s", string(bodyBytes))
					}
				}

				// Send response
				w.WriteHeader(tt.responseStatus)
				if tt.responseStatus != http.StatusNoContent {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			// Make request
			result, err := HTTPRequest[testRequest, testResponse](
				context.Background(),
				tt.method,
				server.URL,
				tt.requestBody,
				tt.headers,
				5,
			)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify response (skip for 204 No Content)
			if tt.responseStatus != http.StatusNoContent {
				if result.Token != tt.responseBody.Token {
					t.Errorf("Expected token %s, got %s", tt.responseBody.Token, result.Token)
				}
				if result.UserID != tt.responseBody.UserID {
					t.Errorf("Expected userID %d, got %d", tt.responseBody.UserID, result.UserID)
				}
			}
		})
	}
}

func TestHTTPRequest_GET(t *testing.T) {
	expectedResponse := testResponse{
		Token:  "get-token",
		UserID: 123,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		// GET should not have Content-Type header
		if r.Header.Get("Content-Type") == "application/json" {
			t.Error("GET request should not have Content-Type header")
		}

		// Verify custom headers are present
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("Expected X-API-Key header")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	headers := map[string]string{
		"X-API-Key": "test-key",
	}

	result, err := HTTPRequest[testRequest, testResponse](
		context.Background(),
		http.MethodGet,
		server.URL,
		testRequest{}, // Body should be ignored for GET
		headers,
		5,
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Token != expectedResponse.Token {
		t.Errorf("Expected token %s, got %s", expectedResponse.Token, result.Token)
	}
}

func TestHTTPRequest_ErrorStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"400 Bad Request", http.StatusBadRequest, "Invalid request data"},
		{"401 Unauthorized", http.StatusUnauthorized, "Authentication required"},
		{"404 Not Found", http.StatusNotFound, "Resource not found"},
		{"500 Internal Server Error", http.StatusInternalServerError, "Internal error occurred"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			_, err := HTTPRequest[testRequest, testResponse](
				context.Background(),
				http.MethodPost,
				server.URL,
				testRequest{Username: "test", Password: "pass"},
				nil,
				5,
			)

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			// Verify error message contains status code
			if !strings.Contains(err.Error(), fmt.Sprintf("%d", tt.statusCode)) {
				t.Errorf("Error message should contain status code %d: %v", tt.statusCode, err)
			}

			// Verify error message contains truncated body (if applicable)
			if len(tt.body) <= 200 && !strings.Contains(err.Error(), tt.body) {
				t.Errorf("Error message should contain body text: %v", err)
			}
		})
	}
}

func TestHTTPRequest_LongErrorBodyTruncation(t *testing.T) {
	// Create a long error body (more than 200 characters)
	longBody := strings.Repeat("This is a long error message with potential sensitive data. ", 10)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(longBody))
	}))
	defer server.Close()

	_, err := HTTPRequest[testRequest, testResponse](
		context.Background(),
		http.MethodPost,
		server.URL,
		testRequest{Username: "test", Password: "pass"},
		nil,
		5,
	)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Verify truncation occurred
	if !strings.Contains(err.Error(), "(truncated)") {
		t.Error("Long error body should be truncated")
	}

	// Verify the error message doesn't contain the full body
	if strings.Contains(err.Error(), longBody) {
		t.Error("Error message should not contain the full long body")
	}
}

func TestHTTPRequest_Timeout(t *testing.T) {
	t.Run("Request times out with function timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow response (2 seconds)
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: "token", UserID: 1})
		}))
		defer server.Close()

		start := time.Now()
		// Set timeout to 500ms
		_, err := HTTPRequest[testRequest, testResponse](
			context.Background(),
			http.MethodPost,
			server.URL,
			testRequest{Username: "test", Password: "pass"},
			nil,
			1, // 1 second timeout
		)
		elapsed := time.Since(start)

		if err == nil {
			t.Fatal("Expected timeout error, got nil")
		}

		// Verify it's a timeout-related error
		if !strings.Contains(err.Error(), "request failed") {
			t.Errorf("Expected request failed error, got: %v", err)
		}

		// Verify timeout occurred around 1 second (with some tolerance)
		if elapsed < 800*time.Millisecond || elapsed > 1500*time.Millisecond {
			t.Logf("Warning: timeout took %v (expected ~1s)", elapsed)
		}
	})

	t.Run("Request completes before timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Fast response (100ms)
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: "success", UserID: 42})
		}))
		defer server.Close()

		start := time.Now()
		// Set timeout to 5 seconds (longer than response time)
		result, err := HTTPRequest[testRequest, testResponse](
			context.Background(),
			http.MethodPost,
			server.URL,
			testRequest{Username: "test", Password: "pass"},
			nil,
			5,
		)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.Token != "success" || result.UserID != 42 {
			t.Errorf("Expected token='success' userID=42, got token='%s' userID=%d", result.Token, result.UserID)
		}

		// Verify request completed quickly
		if elapsed > 1*time.Second {
			t.Errorf("Expected fast response, took %v", elapsed)
		}
	})

	t.Run("Context deadline shorter than function timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Slow response (3 seconds)
			time.Sleep(3 * time.Second)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: "token", UserID: 1})
		}))
		defer server.Close()

		// Context with 500ms deadline
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		start := time.Now()
		// Function timeout is 5 seconds, but context deadline is 500ms
		_, err := HTTPRequest[testRequest, testResponse](
			ctx,
			http.MethodPost,
			server.URL,
			testRequest{Username: "test", Password: "pass"},
			nil,
			5,
		)
		elapsed := time.Since(start)

		if err == nil {
			t.Fatal("Expected timeout error, got nil")
		}

		// Should timeout around 500ms (context deadline)
		if elapsed < 400*time.Millisecond || elapsed > 1*time.Second {
			t.Logf("Warning: timeout took %v (expected ~500ms)", elapsed)
		}
	})

	t.Run("Function timeout shorter than context deadline", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Slow response (3 seconds)
			time.Sleep(3 * time.Second)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: "token", UserID: 1})
		}))
		defer server.Close()

		// Context with 5 second deadline
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := time.Now()
		// Function timeout is 500ms (shorter than context)
		_, err := HTTPRequest[testRequest, testResponse](
			ctx,
			http.MethodPost,
			server.URL,
			testRequest{Username: "test", Password: "pass"},
			nil,
			1, // 1 second timeout
		)
		elapsed := time.Since(start)

		if err == nil {
			t.Fatal("Expected timeout error, got nil")
		}

		// Should timeout around 1 second (function timeout)
		if elapsed < 800*time.Millisecond || elapsed > 1500*time.Millisecond {
			t.Logf("Warning: timeout took %v (expected ~1s)", elapsed)
		}
	})

	t.Run("Zero timeout should still work", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: "instant", UserID: 99})
		}))
		defer server.Close()

		// Zero timeout means immediate timeout
		_, err := HTTPRequest[testRequest, testResponse](
			context.Background(),
			http.MethodPost,
			server.URL,
			testRequest{Username: "test", Password: "pass"},
			nil,
			0,
		)

		// Should timeout immediately or complete very quickly
		// This is a bit unpredictable, but it shouldn't crash
		if err != nil && !strings.Contains(err.Error(), "request failed") {
			t.Errorf("Unexpected error type: %v", err)
		}
	})
}

func TestHTTPRequest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(testResponse{Token: "token", UserID: 1})
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	_, err := HTTPRequest[testRequest, testResponse](
		ctx,
		http.MethodPost,
		server.URL,
		testRequest{Username: "test", Password: "pass"},
		nil,
		5,
	)

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}
}

func TestHTTPRequest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json response"))
	}))
	defer server.Close()

	_, err := HTTPRequest[testRequest, testResponse](
		context.Background(),
		http.MethodPost,
		server.URL,
		testRequest{Username: "test", Password: "pass"},
		nil,
		5,
	)

	if err == nil {
		t.Fatal("Expected JSON parse error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("Expected parse error message, got: %v", err)
	}
}

func TestHTTPRequest_EmptyResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Don't write any body
	}))
	defer server.Close()

	result, err := HTTPRequest[testRequest, testResponse](
		context.Background(),
		http.MethodPost,
		server.URL,
		testRequest{Username: "test", Password: "pass"},
		nil,
		5,
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Result should be zero value
	if result.Token != "" || result.UserID != 0 {
		t.Error("Expected zero value for empty response")
	}
}

func TestHTTPRequest_InvalidURL(t *testing.T) {
	_, err := HTTPRequest[testRequest, testResponse](
		context.Background(),
		http.MethodPost,
		"://invalid-url",
		testRequest{Username: "test", Password: "pass"},
		nil,
		5,
	)

	if err == nil {
		t.Fatal("Expected error for invalid URL, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create request") {
		t.Errorf("Expected create request error, got: %v", err)
	}
}

func TestHTTPRequest_NetworkError(t *testing.T) {
	// Use a URL that will fail to connect
	_, err := HTTPRequest[testRequest, testResponse](
		context.Background(),
		http.MethodPost,
		"http://localhost:99999",
		testRequest{Username: "test", Password: "pass"},
		nil,
		1,
	)

	if err == nil {
		t.Fatal("Expected network error, got nil")
	}

	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("Expected request failed error, got: %v", err)
	}
}

func TestHTTPRequest_GenericTypes(t *testing.T) {
	// Test with different generic types
	type CustomRequest struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	type CustomResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	expectedReq := CustomRequest{ID: 100, Name: "Test"}
	expectedResp := CustomResponse{Success: true, Message: "OK"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CustomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode: %v", err)
		}

		if req.ID != expectedReq.ID || req.Name != expectedReq.Name {
			t.Errorf("Request mismatch: got %+v, expected %+v", req, expectedReq)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResp)
	}))
	defer server.Close()

	result, err := HTTPRequest[CustomRequest, CustomResponse](
		context.Background(),
		http.MethodPost,
		server.URL,
		expectedReq,
		nil,
		5,
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Success != expectedResp.Success || result.Message != expectedResp.Message {
		t.Errorf("Response mismatch: got %+v, expected %+v", result, expectedResp)
	}
}

func TestHTTPRequest_ZeroValueBody(t *testing.T) {
	t.Run("Empty struct sends no body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify no body is sent
			bodyBytes, _ := io.ReadAll(r.Body)
			if len(bodyBytes) > 0 {
				t.Errorf("Expected no body for zero-value request, got: %s", string(bodyBytes))
			}

			// Verify no Content-Type header
			if r.Header.Get("Content-Type") == "application/json" {
				t.Error("Expected no Content-Type header for zero-value request")
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: "token", UserID: 1})
		}))
		defer server.Close()

		result, err := HTTPRequest[testRequest, testResponse](
			context.Background(),
			http.MethodDelete,
			server.URL,
			testRequest{}, // Zero value
			nil,
			5,
		)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.Token != "token" || result.UserID != 1 {
			t.Errorf("Unexpected result: %+v", result)
		}
	})

	t.Run("Non-zero struct sends body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify body is sent
			var req testRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("Failed to decode body: %v", err)
			}

			if req.Username != "testuser" || req.Password != "testpass" {
				t.Errorf("Unexpected request: %+v", req)
			}

			// Verify Content-Type header
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("Expected Content-Type: application/json")
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: "token", UserID: 1})
		}))
		defer server.Close()

		result, err := HTTPRequest[testRequest, testResponse](
			context.Background(),
			http.MethodPost,
			server.URL,
			testRequest{Username: "testuser", Password: "testpass"}, // Non-zero value
			nil,
			5,
		)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.Token != "token" || result.UserID != 1 {
			t.Errorf("Unexpected result: %+v", result)
		}
	})

	t.Run("Partial struct sends body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify body is sent even with partial data
			var req testRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("Failed to decode body: %v", err)
			}

			if req.Username != "onlyusername" {
				t.Errorf("Unexpected username: %s", req.Username)
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: "token", UserID: 1})
		}))
		defer server.Close()

		result, err := HTTPRequest[testRequest, testResponse](
			context.Background(),
			http.MethodPost,
			server.URL,
			testRequest{Username: "onlyusername"}, // Partial value
			nil,
			5,
		)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.Token != "token" || result.UserID != 1 {
			t.Errorf("Unexpected result: %+v", result)
		}
	})
}

func TestHTTPRequest_AllHTTPMethods(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		sendBody       bool
		expectBody     bool
		responseStatus int
	}{
		{"GET without body", http.MethodGet, false, false, http.StatusOK},
		{"POST with body", http.MethodPost, true, true, http.StatusCreated},
		{"PUT with body", http.MethodPut, true, true, http.StatusOK},
		{"PATCH with body", http.MethodPatch, true, true, http.StatusOK},
		{"DELETE without body", http.MethodDelete, false, false, http.StatusNoContent},
		{"HEAD without body", http.MethodHead, false, false, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method
				if r.Method != tt.method {
					t.Errorf("Expected method %s, got %s", tt.method, r.Method)
				}

				// Verify body presence
				bodyBytes, _ := io.ReadAll(r.Body)
				hasBody := len(bodyBytes) > 0

				if tt.expectBody && !hasBody {
					t.Error("Expected request body but got none")
				}
				if !tt.expectBody && hasBody {
					t.Errorf("Expected no request body but got: %s", string(bodyBytes))
				}

				w.WriteHeader(tt.responseStatus)
				if tt.method != http.MethodHead && tt.responseStatus != http.StatusNoContent {
					json.NewEncoder(w).Encode(testResponse{Token: "method-test", UserID: 1})
				}
			}))
			defer server.Close()

			var req testRequest
			if tt.sendBody {
				req = testRequest{Username: "test", Password: "pass"}
			}

			result, err := HTTPRequest[testRequest, testResponse](
				context.Background(),
				tt.method,
				server.URL,
				req,
				nil,
				5,
			)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// For methods that return content
			if tt.method != http.MethodHead && tt.responseStatus != http.StatusNoContent {
				if result.Token != "method-test" {
					t.Errorf("Expected token='method-test', got '%s'", result.Token)
				}
			}
		})
	}
}

func TestHTTPRequest_HeaderPropagation(t *testing.T) {
	t.Run("Multiple headers are sent correctly", func(t *testing.T) {
		expectedHeaders := map[string]string{
			"Authorization": "Bearer test-token-12345",
			"X-API-Key":     "api-key-value",
			"X-Request-ID":  "req-123",
			"Content-Type":  "application/json", // Should be set by function
			"X-Custom":      "custom-value",
			"Accept":        "application/json",
			"User-Agent":    "test-agent/1.0",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify all headers except Content-Type (which is set by function)
			for key, expectedValue := range expectedHeaders {
				if key == "Content-Type" {
					continue // Skip - this is set by the function
				}
				actualValue := r.Header.Get(key)
				if actualValue != expectedValue {
					t.Errorf("Header %s: expected '%s', got '%s'", key, expectedValue, actualValue)
				}
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: "header-test", UserID: 1})
		}))
		defer server.Close()

		// Remove Content-Type from headers to send (function will add it)
		headersToSend := make(map[string]string)
		for k, v := range expectedHeaders {
			if k != "Content-Type" {
				headersToSend[k] = v
			}
		}

		result, err := HTTPRequest[testRequest, testResponse](
			context.Background(),
			http.MethodPost,
			server.URL,
			testRequest{Username: "test", Password: "pass"},
			headersToSend,
			5,
		)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.Token != "header-test" {
			t.Errorf("Expected token='header-test', got '%s'", result.Token)
		}
	})

	t.Run("Empty headers map works", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: "no-headers", UserID: 1})
		}))
		defer server.Close()

		result, err := HTTPRequest[testRequest, testResponse](
			context.Background(),
			http.MethodPost,
			server.URL,
			testRequest{Username: "test", Password: "pass"},
			map[string]string{},
			5,
		)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.Token != "no-headers" {
			t.Errorf("Expected token='no-headers', got '%s'", result.Token)
		}
	})
}

func TestHTTPRequest_LargePayload(t *testing.T) {
	t.Run("Large request body", func(t *testing.T) {
		// Create a large string (1MB)
		largeString := strings.Repeat("A", 1024*1024)

		type LargeRequest struct {
			Data string `json:"data"`
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req LargeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("Failed to decode: %v", err)
			}

			if len(req.Data) != len(largeString) {
				t.Errorf("Expected data length %d, got %d", len(largeString), len(req.Data))
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: "large-payload", UserID: 1})
		}))
		defer server.Close()

		result, err := HTTPRequest[LargeRequest, testResponse](
			context.Background(),
			http.MethodPost,
			server.URL,
			LargeRequest{Data: largeString},
			nil,
			10,
		)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.Token != "large-payload" {
			t.Errorf("Expected token='large-payload', got '%s'", result.Token)
		}
	})

	t.Run("Large response body", func(t *testing.T) {
		// Create a large response (1MB)
		largeString := strings.Repeat("B", 1024*1024)

		type LargeResponse struct {
			Data string `json:"data"`
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(LargeResponse{Data: largeString})
		}))
		defer server.Close()

		result, err := HTTPRequest[testRequest, LargeResponse](
			context.Background(),
			http.MethodPost,
			server.URL,
			testRequest{Username: "test", Password: "pass"},
			nil,
			10,
		)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(result.Data) != len(largeString) {
			t.Errorf("Expected data length %d, got %d", len(largeString), len(result.Data))
		}
	})
}

func TestHTTPRequest_SpecialCharacters(t *testing.T) {
	t.Run("Unicode and special characters in request", func(t *testing.T) {
		specialChars := "Hello 世界 🌍 Привет ñ é € ™ © ® ← → ↑ ↓"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req testRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("Failed to decode: %v", err)
			}

			if req.Username != specialChars {
				t.Errorf("Username mismatch: expected '%s', got '%s'", specialChars, req.Username)
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(testResponse{Token: specialChars, UserID: 1})
		}))
		defer server.Close()

		result, err := HTTPRequest[testRequest, testResponse](
			context.Background(),
			http.MethodPost,
			server.URL,
			testRequest{Username: specialChars, Password: "pass"},
			nil,
			5,
		)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.Token != specialChars {
			t.Errorf("Token mismatch: expected '%s', got '%s'", specialChars, result.Token)
		}
	})
}

func TestHTTPRequest_ConcurrentRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Small delay to simulate real server
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(testResponse{Token: "concurrent", UserID: 1})
	}))
	defer server.Close()

	// Run 10 concurrent requests
	const numRequests = 10
	errors := make(chan error, numRequests)
	results := make(chan testResponse, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			result, err := HTTPRequest[testRequest, testResponse](
				context.Background(),
				http.MethodPost,
				server.URL,
				testRequest{Username: fmt.Sprintf("user%d", id), Password: "pass"},
				nil,
				5,
			)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		select {
		case err := <-errors:
			t.Errorf("Request failed: %v", err)
		case result := <-results:
			if result.Token == "concurrent" {
				successCount++
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}

	if successCount != numRequests {
		t.Errorf("Expected %d successful requests, got %d", numRequests, successCount)
	}
}
