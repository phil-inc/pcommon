package awscomm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var baseURL = "sdfh"
var apiKey = "hsdfh"

func TestNewClient(t *testing.T) {
	client := NewClient(baseURL, apiKey)

	assert.NotNil(t, client)
	assert.NotEmpty(t, client.baseURL)
	assert.NotEmpty(t, client.apiKey)
}

func TestSendSMS_ValidationErrors(t *testing.T) {
	client := NewClient(baseURL, apiKey)
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *SMSRequest
		expectError bool
	}{
		{
			name: "missing phone number",
			request: &SMSRequest{
				CallbackURL: "https://example.com/callback",
				Payload: SMSPayload{
					ToPhoneNumber: "",
					Message:       "Test message",
				},
			},
			expectError: true,
		},
		{
			name: "missing message",
			request: &SMSRequest{
				CallbackURL: "https://example.com/callback",
				Payload: SMSPayload{
					ToPhoneNumber: "+17609579111",
					Message:       "",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SendSMS(ctx, tt.request, nil)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendVoiceMail_ValidationErrors(t *testing.T) {
	client := NewClient(baseURL, apiKey)
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *VoiceMailRequest
		expectError bool
	}{
		{
			name: "missing phone number",
			request: &VoiceMailRequest{
				CallbackURL: "https://example.com/callback",
				Payload: VoiceMailPayload{
					ToPhoneNumber: "",
					Message:       "Test message",
				},
			},
			expectError: true,
		},
		{
			name: "missing message",
			request: &VoiceMailRequest{
				CallbackURL: "https://example.com/callback",
				Payload: VoiceMailPayload{
					ToPhoneNumber: "+17609579111",
					Message:       "",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SendVoiceMail(ctx, tt.request, nil)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendEmail_ValidationErrors(t *testing.T) {
	client := NewClient(baseURL, apiKey)
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *EmailRequest
		expectError bool
	}{
		{
			name: "missing recipients",
			request: &EmailRequest{
				CallbackURL: "https://example.com/callback",
				Payload: EmailPayload{
					To:      []EmailRecipient{},
					Subject: "Test",
					Text:    "Test message",
				},
			},
			expectError: true,
		},
		{
			name: "missing subject",
			request: &EmailRequest{
				CallbackURL: "https://example.com/callback",
				Payload: EmailPayload{
					To: []EmailRecipient{
						{Email: "test@example.com", Type: "to"},
					},
					Subject: "",
					Text:    "Test message",
				},
			},
			expectError: true,
		},
		{
			name: "missing content",
			request: &EmailRequest{
				CallbackURL: "https://example.com/callback",
				Payload: EmailPayload{
					To: []EmailRecipient{
						{Email: "test@example.com", Type: "to"},
					},
					Subject: "Test",
					Text:    "",
					HTML:    "",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SendEmail(ctx, tt.request, nil)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendFax_ValidationErrors(t *testing.T) {
	client := NewClient(baseURL, apiKey)
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *FaxRequest
		expectError bool
	}{
		{
			name: "missing fax number",
			request: &FaxRequest{
				CallbackURL: "https://example.com/callback",
				Payload: FaxPayload{
					ToFaxNumber: "",
					FileURL:     "s3://bucket/file.pdf",
				},
			},
			expectError: true,
		},
		{
			name: "missing file URL",
			request: &FaxRequest{
				CallbackURL: "https://example.com/callback",
				Payload: FaxPayload{
					ToFaxNumber: "+1234567890",
					FileURL:     "",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SendFax(ctx, tt.request, nil)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetPresignedURL_ValidationErrors(t *testing.T) {
	client := NewClient(baseURL, apiKey)
	ctx := context.Background()

	tests := []struct {
		name          string
		fileExtension string
		contentType   string
		expectError   bool
	}{
		{
			name:          "missing file extension",
			fileExtension: "",
			contentType:   "application/pdf",
			expectError:   true,
		},
		{
			name:          "missing content type",
			fileExtension: "pdf",
			contentType:   "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetPresignedURL(ctx, tt.fileExtension, tt.contentType)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestError(t *testing.T) {
	t.Run("simple error", func(t *testing.T) {
		err := NewError("test error")
		assert.Equal(t, "test error", err.Error())
		assert.Nil(t, err.Unwrap())
	})

	t.Run("wrapped error", func(t *testing.T) {
		originalErr := NewError("original")
		wrappedErr := WrapError(originalErr, "wrapped")
		assert.Contains(t, wrappedErr.Error(), "wrapped")
		assert.Contains(t, wrappedErr.Error(), "original")
		assert.NotNil(t, wrappedErr.Unwrap())
	})
}
