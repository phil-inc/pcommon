package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/golang/mock/mockgen/model"
)

type ErrorObject struct {
	Status     string
	StatusCode int
	ErrorBody  string
}

//go:generate mockgen -destination=./mocks/RoundTripper.go -package=mocks net/http RoundTripper

//go:generate mockgen -destination=./mocks/HTTPClient.go -package=mocks github.com/phil-inc/pcommon/pkg/network HTTPClient
type HTTPClient interface {
	Get(url string) (resp *http.Response, err error)
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
	PostForm(url string, data url.Values) (resp *http.Response, err error)
	Head(url string) (resp *http.Response, err error)
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

var httpClient HTTPClient

func init() {
	httpClient = &http.Client{
		Timeout: time.Second * 60,
	}
}

// SetHttpClient - used primarily for testing, allows for mock tests
func SetHttpClient(c HTTPClient) {
	httpClient = c
}

// Get - GET request with headers
func Get(url string, headers map[string]string) ([]byte, error) {
	return HTTPGet(url, headers)
}

// GetWithTimeout - GET request with headers and timeout
func GetWithTimeout(url string, headers map[string]string, timeout int) ([]byte, error) {
	return HTTPGetWithTimeOut(url, headers, timeout)
}

func getNewRequest(method, url string, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// add headers
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	return req, nil
}

func parseGetResponse(res *http.Response, url string) ([]byte, error) {
	if res.Body != nil {
		defer res.Body.Close()
	}

	if res.StatusCode != 200 && res.StatusCode != 201 {
		return nil, fmt.Errorf("http response NOT_OK. Status: %s, Code:%d", res.Status, res.StatusCode)
	}

	return io.ReadAll(res.Body)
}

// PostWithTimeout - POST request with headers and custom timeout value
func PostWithTimeout(url string, body string, headers map[string]string, timeout int) ([]byte, error) {
	return HTTPPostWithTimeOut(url, body, headers, timeout)
}

func HTTPPostWithTimeOut(url string, body string, headers map[string]string, timeout int) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	reader := strings.NewReader(body)

	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return nil, err
	}

	// add headers
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return parsePostResponse(res, url)
}

func parsePostResponse(res *http.Response, url string) ([]byte, error) {
	if res.Body != nil {
		defer res.Body.Close()
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		errResp := fmt.Sprintf("Http response NOT_OK. Status: %s, Code:%d", res.Status, res.StatusCode)
		if res.Body != nil {
			resp, _ := io.ReadAll(res.Body)
			errResp = errResp + fmt.Sprintf(", Body: %s", resp)
		}

		return nil, errors.New(errResp)
	}

	return io.ReadAll(res.Body)
}

func HTTPGetWithTimeOut(url string, headers map[string]string, timeout int) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	req, err := getNewRequest("GET", url, headers)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return parseGetResponse(res, url)
}

// HTTPGet - makes a get request to the given URL and HTTP headers.
// it returns response data byte or error
func HTTPGet(url string, headers map[string]string) ([]byte, error) {
	req, err := getNewRequest("GET", url, headers)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return parseGetResponse(res, url)
}

// HTTPGetWithBasicAuth - makes a get request to the given URL and HTTP headers With Basic Auth
// it returns response data byte or error
func HTTPGetWithBasicAuth(url string, headers map[string]string, username, password string) ([]byte, error) {
	req, err := getNewRequest("GET", url, headers)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(username, password)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return parseGetResponse(res, url)
}

// HTTPDelete - makes a delete request to the given URL and HTTP headers.
// it returns response data byte or error
func HTTPDelete(url string, headers map[string]string) ([]byte, error) {
	req, err := getNewRequest("DELETE", url, headers)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	if res.StatusCode != 200 && res.StatusCode != 201 && res.StatusCode != 204 {
		return nil, fmt.Errorf("http response NOT_OK. Status: %s, Code:%d", res.Status, res.StatusCode)
	}

	return io.ReadAll(res.Body)
}

// HTTPFormPost makes a POST data to the given url with headers
func HTTPFormPost(url string, values url.Values, headers map[string]string) ([]byte, error) {
	rb := strings.NewReader(values.Encode())
	req, _ := http.NewRequest("POST", url, rb)

	// add headers
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return io.ReadAll(resp.Body)
	}

	body, _ := io.ReadAll(resp.Body)
	return body, fmt.Errorf("http response NOT_OK. Status: %s, Code:%d", resp.Status, resp.StatusCode)
}

func HTTPDataUpload(url, usrName, password string, body bytes.Buffer, headers map[string]string) ([]byte, error) {
	req, _ := http.NewRequest("POST", url, &body)
	req.SetBasicAuth(usrName, password)

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return io.ReadAll(resp.Body)
	}

	return nil, fmt.Errorf("http response NOT_OK. Status: %s, Code:%d", resp.Status, resp.StatusCode)
}

// HTTPJsonGet - sends JSON string data as get request
func HTTPJsonGet(url string, headers map[string]string) ([]byte, error) {
	return doHTTP(url, "GET", "", headers)
}

// HTTPJsonPost - sends JSON string data as post request
func HTTPJsonPost(url, jsonBody string, headers map[string]string) ([]byte, error) {
	return doHTTP(url, "POST", jsonBody, headers)
}

// HTTPJsonPut - sends JSON string data as put request
func HTTPJsonPut(url, jsonBody string, headers map[string]string) ([]byte, error) {
	return doHTTP(url, "PUT", jsonBody, headers)
}

// HTTPJsonPost - sends JSON string data as post request
// DEPRECATED - DO NOT USE
func HTTPJsonPostWithErrorObject(url, jsonBody string, headers map[string]string) ([]byte, *ErrorObject) {
	resp, errorCode, _ := httpSend(url, "POST", jsonBody, headers)
	return resp, errorCode
}

// HTTPJsonPut - sends JSON string data as put request
// DEPRECATED - DO NOT USE
func HTTPJsonPutWithErrorObject(url, jsonBody string, headers map[string]string) ([]byte, *ErrorObject) {
	resp, errorCode, _ := httpSend(url, "PUT", jsonBody, headers)
	return resp, errorCode
}

func doHTTP(url, method, body string, headers map[string]string) ([]byte, error) {
	reader := strings.NewReader(body)
	req, _ := http.NewRequest(method, url, reader)
	// add headers
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {

		errResp := fmt.Sprintf("Http response NOT_OK. Status: %s, Code:%d", res.Status, res.StatusCode)
		if res.Body != nil {
			resp, _ := io.ReadAll(res.Body)
			errResp = errResp + ", Body: " + string(resp)
		}

		return nil, errors.New(errResp)
	}

	return io.ReadAll(res.Body)
}

// DEPRECATED - DO NOT USE AND WILL BE DELETED
func httpSend(url, method, body string, headers map[string]string) ([]byte, *ErrorObject, error) {
	reader := strings.NewReader(body)
	req, _ := http.NewRequest(method, url, reader)
	// add headers
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {

		errResp := fmt.Sprintf("Http response NOT_OK. Status: %s, Code:%d", res.Status, res.StatusCode)
		if res.Body != nil {
			resp, _ := io.ReadAll(res.Body)
			errResp = errResp + fmt.Sprintf(", Body: %s", resp)
		}

		return nil, &ErrorObject{Status: res.Status, StatusCode: res.StatusCode, ErrorBody: errResp}, fmt.Errorf(errResp)
	}
	resp, error := io.ReadAll(res.Body)

	return resp, nil, error
}

// HTTPMultipartPost - Sends multipart data as POST request
func HTTPMultipartPost(url string, body, headers map[string]string) ([]byte, error) {
	reqBody := &bytes.Buffer{}

	writer := multipart.NewWriter(reqBody)
	// set formdata
	for k, v := range body {
		fw, _ := writer.CreateFormField(k)
		fw.Write([]byte(v))
	}
	writer.Close()

	req, _ := http.NewRequest("POST", url, bytes.NewReader(reqBody.Bytes()))

	// set content type multipart/form-data
	req.Header.Set("Content-Type", writer.FormDataContentType())
	//add headers
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {

		errResp := fmt.Sprintf("Http response NOT_OK. Status: %s, Code:%d", res.Status, res.StatusCode)
		if res.Body != nil {
			resp, _ := io.ReadAll(res.Body)
			errResp = errResp + fmt.Sprintf(", Body: %s", resp)
		}

		return nil, errors.New(errResp)
	}

	return io.ReadAll(res.Body)
}

// GetStatusCodeFromError return status code for the http header
func GetStatusCodeFromError(err error) int {
	re := regexp.MustCompile(`Code:(\d+)`)
	matches := re.FindStringSubmatch(err.Error())

	if len(matches) < 2 {
		return 0
	}

	codeStr := matches[1]
	code, err := strconv.Atoi(codeStr)
	if err != nil {
		return 0
	}

	return code
}

// HTTPRequest performs a type-safe HTTP request with generic request and response types.
// This function provides compile-time type checking for both request and response structures,
// making it easier to work with strongly-typed APIs.
//
// Type Parameters:
//   - Req: The type of the request body. Must be JSON-serializable.
//   - Res: The type of the response body. Must be JSON-deserializable.
//
// Parameters:
//   - ctx: Context for the request, allowing for cancellation and deadline control
//   - method: HTTP method (GET, POST, PUT, DELETE, etc.)
//   - url: The target URL for the request
//   - request: The request body that will be marshaled to JSON (ignored for GET and HEAD)
//   - headers: HTTP headers to include in the request (can be nil)
//   - timeoutSeconds: Request timeout in seconds
//
// Returns:
//   - Res: The parsed response body of type Res
//   - error: Any error encountered during the request or response processing
//
// Example usage:
//
//	type LoginRequest struct {
//	    Username string `json:"username"`
//	    Password string `json:"password"`
//	}
//
//	type LoginResponse struct {
//	    Token string `json:"token"`
//	    UserID int  `json:"user_id"`
//	}
//
//	headers := map[string]string{
//	    "Authorization": "Bearer token123",
//	    "X-Custom-Header": "value",
//	}
//
//	resp, err := HTTPRequest[LoginRequest, LoginResponse](
//	    context.Background(),
//	    http.MethodPost,
//	    "https://api.example.com/login",
//	    LoginRequest{Username: "user", Password: "pass"},
//	    headers,
//	    30,
//	)
func HTTPRequest[Req any, Res any](ctx context.Context, method, url string, request Req, headers map[string]string, timeoutSeconds int) (Res, error) {
	var result Res
	var body io.Reader

	// Marshal request to JSON if method supports body and request is not nil
	// Methods like GET typically don't have a body
	if method != http.MethodGet && method != http.MethodHead {
		requestBodyBytes, err := json.Marshal(request)
		if err != nil {
			return result, fmt.Errorf("[HTTP] failed to marshal request: %w", err)
		}
		body = bytes.NewReader(requestBodyBytes)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return result, fmt.Errorf("[HTTP] failed to create request for %s %s: %w", method, url, err)
	}

	// Set content type for methods with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Make HTTP call with timeout
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("[HTTP] request failed for %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("[HTTP] failed to read response body from %s %s: %w", method, url, err)
	}

	// Check HTTP status - success codes vary by method
	if !isSuccessStatusCode(resp.StatusCode) {
		return result, fmt.Errorf("[HTTP] request to %s %s returned error status %d: %s", method, url, resp.StatusCode, string(respBody))
	}

	// Unmarshal response if body is not empty
	if len(respBody) > 0 {
		err = json.Unmarshal(respBody, &result)
		if err != nil {
			return result, fmt.Errorf("[HTTP] failed to parse response from %s %s: %w", method, url, err)
		}
	}

	return result, nil
}

// isSuccessStatusCode checks if the HTTP status code represents a successful response.
// Success codes are in the 2xx range (200-299).
func isSuccessStatusCode(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}
