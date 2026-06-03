package awscomm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var baseURL = "sdfh"
var serviceName = "test-service"
var serviceApiKey = "secret-key"

func TestNewClient(t *testing.T) {
	client := NewClient(baseURL, serviceName, serviceApiKey)

	assert.NotNil(t, client)
	assert.NotEmpty(t, client.baseURL)
	assert.NotEmpty(t, client.serviceName)
	assert.NotEmpty(t, client.serviceApiKey)
}

func TestSendSMS_ValidationErrors(t *testing.T) {
	client := NewClient(baseURL, serviceName, serviceApiKey)
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
			_, err := client.SendSMS(ctx, tt.request)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendVoiceCall_ValidationErrors(t *testing.T) {
	client := NewClient(baseURL, serviceName, serviceApiKey)
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *VoiceCallRequest
		expectError bool
	}{
		{
			name: "missing phone number",
			request: &VoiceCallRequest{
				CallbackURL: "https://example.com/callback",
				Payload: VoiceCallPayload{
					ToPhoneNumber: "",
					Message:       "Test message",
				},
			},
			expectError: true,
		},
		{
			name: "missing message and twiml",
			request: &VoiceCallRequest{
				CallbackURL: "https://example.com/callback",
				Payload: VoiceCallPayload{
					ToPhoneNumber: "+17609579111",
					Message:       "",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SendVoiceCall(ctx, tt.request)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendVoiceCall_AllowsTwiMLPayload(t *testing.T) {
	var captured VoiceCallRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/send/voice_mail", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"status":"QUEUED","comm_request_id":"voice-call-test","type":"voice_mail"}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient(server.URL, serviceName, serviceApiKey)
	resp, err := client.SendVoiceCall(context.Background(), &VoiceCallRequest{
		CallbackURL: "https://example.com/callback",
		Payload: VoiceCallPayload{
			ToPhoneNumber: "+17609579111",
			TwiML:         `<?xml version="1.0" encoding="UTF-8"?><Response><Say>Hello</Say></Response>`,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "voice-call-test", resp.CommRequestID)
	assert.Equal(t, "+17609579111", captured.Payload.ToPhoneNumber)
	assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?><Response><Say>Hello</Say></Response>`, captured.Payload.TwiML)
	assert.Empty(t, captured.Payload.Message)
}

func TestVoiceCallRequest_MarshalsTwiMLPayload(t *testing.T) {
	req := VoiceCallRequest{
		CallbackURL: "https://example.com/callback",
		Payload: VoiceCallPayload{
			ToPhoneNumber: "+18024712700",
			TwiML:         `<?xml version="1.0" encoding="UTF-8"?><Response><Say>Hello</Say></Response>`,
		},
		Metadata: map[string]any{
			"order_number": "1234-1234-1234",
		},
	}

	raw, err := json.Marshal(req)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))

	voicePayload := payload["payload"].(map[string]any)
	assert.Equal(t, "+18024712700", voicePayload["to_phone_number"])
	assert.Equal(t, req.Payload.TwiML, voicePayload["twiml"])
	assert.NotContains(t, voicePayload, "message")
}

func TestSendEmail_ValidationErrors(t *testing.T) {
	client := NewClient(baseURL, serviceName, serviceApiKey)
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
			_, err := client.SendEmail(ctx, tt.request)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendFax_ValidationErrors(t *testing.T) {
	client := NewClient(baseURL, serviceName, serviceApiKey)
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
				Metadata: map[string]any{
					"order_number": "1234-1234-1234",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SendFax(ctx, tt.request)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetPresignedURL_ValidationErrors(t *testing.T) {
	client := NewClient(baseURL, serviceName, serviceApiKey)
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
