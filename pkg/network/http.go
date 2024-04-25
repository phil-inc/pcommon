package network

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ErrorObject struct {
	Status     string
	StatusCode int
	ErrorBody  string
}

var httpClient = &http.Client{
	Timeout: time.Second * 90,
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

	return ioutil.ReadAll(res.Body)
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

	return ioutil.ReadAll(res.Body)
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
		return ioutil.ReadAll(resp.Body)
	}

	body, _ := ioutil.ReadAll(resp.Body)
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
		return ioutil.ReadAll(resp.Body)
	}

	return nil, fmt.Errorf("http response NOT_OK. Status: %s, Code:%d", resp.Status, resp.StatusCode)
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
			resp, _ := ioutil.ReadAll(res.Body)
			errResp = errResp + fmt.Sprintf(", Body: %s", resp)
		}

		return nil, fmt.Errorf(errResp)
	}

	return ioutil.ReadAll(res.Body)
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
			resp, _ := ioutil.ReadAll(res.Body)
			errResp = errResp + fmt.Sprintf(", Body: %s", resp)
		}

		return nil, &ErrorObject{Status: res.Status, StatusCode: res.StatusCode, ErrorBody: errResp}, fmt.Errorf(errResp)
	}
	resp, error := ioutil.ReadAll(res.Body)

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
			resp, _ := ioutil.ReadAll(res.Body)
			errResp = errResp + fmt.Sprintf(", Body: %s", resp)
		}

		return nil, fmt.Errorf(errResp)
	}

	return ioutil.ReadAll(res.Body)
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
