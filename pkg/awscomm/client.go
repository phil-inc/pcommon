package awscomm

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"

	"github.com/phil-inc/pcommon/pkg/network"
)

const (
	// MaxFaxFileSize is the maximum file size allowed for fax (20 MB)
	MaxFaxFileSize = 20 * 1024 * 1024 // 20 MB in bytes
)

// Client represents an AWS communication API client
type Client struct {
	baseURL       string
	serviceName   string
	serviceApiKey string
}

func NewClient(baseURL string, serviceName string, serviceApiKey string) *Client {
	return &Client{
		baseURL:       baseURL,
		serviceApiKey: serviceApiKey,
		serviceName:   serviceName,
	}
}

func (c Client) getAuthHeader() map[string]string {
	credential := fmt.Sprintf("%s:%s", c.serviceName, c.serviceApiKey)
	encoded := base64.StdEncoding.EncodeToString([]byte(credential))
	return map[string]string{
		"Authorization": fmt.Sprintf("Basic %s", encoded),
	}
}

func (c *Client) SendSMS(ctx context.Context, request *SMSRequest) (*Response, error) {
	if request.Payload.ToPhoneNumber == "" {
		return nil, NewError("to_phone_number is required")
	}

	if request.Payload.Message == "" {
		return nil, NewError("message is required")
	}

	if !slices.Contains([]string{SMS_FROM_TYPE_LONG_CODE, SMS_FROM_TYPE_SHORT_CODE, ""}, request.Payload.FromType) {
		return nil, NewError(
			"invalid FromType; accepted values: `short_code`, `long_code`, or empty",
		)
	}

	url, err := c.buildURL("/send/sms")
	if err != nil {
		return nil, err
	}
	return c.sendRequest(ctx, url, request)
}

func (c *Client) SendVoiceMail(ctx context.Context, request *VoiceMailRequest) (*Response, error) {
	if request.Payload.ToPhoneNumber == "" {
		return nil, NewError("to_phone_number is required")
	}

	if request.Payload.Message == "" {
		return nil, NewError("message is required")
	}

	url, err := c.buildURL("/send/voice_mail")
	if err != nil {
		return nil, err
	}
	return c.sendRequest(ctx, url, request)
}

func (c *Client) SendEmail(ctx context.Context, request *EmailRequest) (*Response, error) {
	if len(request.Payload.To) == 0 {
		return nil, NewError("at least one recipient is required")
	}

	if request.Payload.Subject == "" {
		return nil, NewError("subject is required")
	}

	if request.Payload.Text == "" && request.Payload.HTML == "" {
		return nil, NewError("either text or html content is required")
	}

	url, err := c.buildURL("/send/email")
	if err != nil {
		return nil, err
	}
	return c.sendRequest(ctx, url, request)
}

func (c *Client) SendFax(ctx context.Context, request *FaxRequest) (*Response, error) {
	if request.Payload.ToFaxNumber == "" {
		return nil, NewError("to_fax_number is required")
	}

	if request.Payload.FileURL == "" && request.Payload.StringData == "" {
		return nil, NewError("FileURL or StringData is required")
	}

	url, err := c.buildURL("/send/fax")
	if err != nil {
		return nil, err
	}
	return c.sendRequest(ctx, url, request)
}

// SendFaxByContentBytes sends a fax using byte content (e.g., PDF bytes)
// Phaxio Legacy V2
// It uploads the content to S3 via presigned URL and then sends the fax
// WARNING: This function loads the entire file content into memory. For large files,
// this may consume significant memory. Maximum file size is 20 MB.
func (c *Client) SendFaxByContentBytes(ctx context.Context, toFaxNumber, callbackURL string, contentBytes []byte) (*Response, error) {
	if toFaxNumber == "" {
		return nil, NewError("to_fax_number is required")
	}

	if len(contentBytes) == 0 {
		return nil, NewError("content_bytes is required")
	}

	if len(contentBytes) > MaxFaxFileSize {
		return nil, NewError(fmt.Sprintf("file size exceeds maximum allowed size of 20 MB (got %d bytes)", len(contentBytes)))
	}

	presigned, err := c.GetPresignedURL(ctx, "pdf", "application/pdf")
	if err != nil {
		return nil, WrapError(err, "failed to get presigned URL")
	}

	// Upload content to S3 using PUT request
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, presigned.UploadURL, bytes.NewReader(contentBytes))
	if err != nil {
		return nil, WrapError(err, "failed to create upload request")
	}
	req.Header.Set("Content-Type", "application/pdf")

	// Use default http client for S3 upload (not using network.HTTPRequest since S3 doesn't return JSON)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, WrapError(err, "failed to upload content")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, NewError(fmt.Sprintf("upload failed with status %d: %s", resp.StatusCode, string(body)))
	}

	faxRequest := &FaxRequest{
		CallbackURL: callbackURL,
		Payload: FaxPayload{
			ToFaxNumber: toFaxNumber,
			FileURL:     presigned.FileURL,
		},
	}

	return c.SendFax(ctx, faxRequest)
}

// SendFaxByFileName sends a fax using a file from the local filesystem
// Phaxio Legacy V2
// It reads the file, uploads it to S3 via presigned URL, and then sends the fax
// WARNING: This function loads the entire file into memory (not chunked/streamed).
// For large files, this may consume significant memory. Maximum file size is 20 MB.
func (c *Client) SendFaxByFileName(ctx context.Context, toFaxNumber, callbackURL, fileName string) (*Response, error) {
	if toFaxNumber == "" {
		return nil, NewError("to_fax_number is required")
	}

	if fileName == "" {
		return nil, NewError("file_name is required")
	}

	contentBytes, err := os.ReadFile(fileName)
	if err != nil {
		return nil, WrapError(err, "failed to read file")
	}

	return c.SendFaxByContentBytes(ctx, toFaxNumber, callbackURL, contentBytes)
}

func (c *Client) GetPresignedURL(ctx context.Context, fileExtension, contentType string) (*PresignedURLResponse, error) {
	if fileExtension == "" {
		return nil, NewError("file_extension is required")
	}

	if contentType == "" {
		return nil, NewError("content_type is required")
	}

	url := fmt.Sprintf("%s/upload/presigned-url?file_extension=%s&content_type=%s", c.baseURL, fileExtension, contentType)
	headers := c.getAuthHeader()

	// Use empty struct for GET request (no body)
	type EmptyRequest struct{}
	response, err := network.HTTPRequest[EmptyRequest, PresignedURLResponse](
		ctx,
		http.MethodGet,
		url,
		EmptyRequest{},
		headers,
		30, // 30 second timeout
	)
	if err != nil {
		return nil, WrapError(err, "failed to get presigned URL")
	}

	return &response, nil
}

func (c *Client) buildURL(p string) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}

	base.Path = path.Join(base.Path, p)
	return base.String(), nil
}

func (c *Client) sendRequest(ctx context.Context, url string, payload interface{}) (*Response, error) {
	headers := c.getAuthHeader()

	// Use type assertion to determine the payload type and call HTTPRequest accordingly
	var response Response
	var err error

	switch req := payload.(type) {
	case *SMSRequest:
		response, err = network.HTTPRequest[*SMSRequest, Response](
			ctx, http.MethodPost, url, req, headers, 30,
		)
	case *VoiceMailRequest:
		response, err = network.HTTPRequest[*VoiceMailRequest, Response](
			ctx, http.MethodPost, url, req, headers, 30,
		)
	case *EmailRequest:
		response, err = network.HTTPRequest[*EmailRequest, Response](
			ctx, http.MethodPost, url, req, headers, 30,
		)
	case *FaxRequest:
		response, err = network.HTTPRequest[*FaxRequest, Response](
			ctx, http.MethodPost, url, req, headers, 30,
		)
	default:
		return nil, NewError("unsupported request type")
	}

	if err != nil {
		return nil, WrapError(err, "failed to send request")
	}

	return &response, nil
}
