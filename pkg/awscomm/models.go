package awscomm

type SMSRequest struct {
	CallbackURL string     `json:"callback_url"`
	Payload     SMSPayload `json:"payload"`
}

type SMSPayload struct {
	ToPhoneNumber string `json:"to_phone_number"`
	Message       string `json:"message"`
}

type VoiceMailRequest struct {
	CallbackURL string           `json:"callback_url"`
	Payload     VoiceMailPayload `json:"payload"`
}

type VoiceMailPayload struct {
	ToPhoneNumber string `json:"to_phone_number"`
	Message       string `json:"message"`
}

type EmailRecipient struct {
	Email string `json:"email"`
	Type  string `json:"type"` // "to", "cc", "bcc"
}

type EmailAttachment struct {
	Type    string `json:"type"`    // MIME type
	Name    string `json:"name"`    // filename
	Content string `json:"content"` // base64 encoded content
}

type EmailRequest struct {
	CallbackURL string       `json:"callback_url"`
	Payload     EmailPayload `json:"payload"`
}

type EmailPayload struct {
	To                 []EmailRecipient  `json:"to"`
	Subject            string            `json:"subject"`
	HTML               string            `json:"html"`
	Text               string            `json:"text"`
	Attachments        []EmailAttachment `json:"attachments,omitempty"`
	Important          bool              `json:"important"`
	Merge              interface{}       `json:"merge,omitempty"`               // Merge data object
	MergeLanguage      string            `json:"merge_language,omitempty"`      // Template language (e.g., "mandrill")
	MergeVars          interface{}       `json:"merge_vars,omitempty"`          // Per-recipient merge variables
	FromName           string            `json:"from_name,omitempty"`           // Sender name
	FromEmail          string            `json:"from_email,omitempty"`          // Sender email address
	PreserveRecipients *bool             `json:"preserve_recipients,omitempty"` // Whether recipients can see each other
}

type FaxRequest struct {
	CallbackURL string     `json:"callback_url"`
	Payload     FaxPayload `json:"payload"`
}

type FaxPayload struct {
	ToFaxNumber    string `json:"to_fax_number"`
	FileURL        string `json:"file_url,omitempty"`
	StringData     string `json:"string_data,omitempty"`
	StringDataType string `json:"string_data_type,omitempty"`
	HeaderText     string `json:"header_text,omitempty"`
}

type Response struct {
	Status        string `json:"status"`          // e.g., "QUEUED"
	CommRequestID string `json:"comm_request_id"` // UUID
	Type          string `json:"type"`            // e.g., "sms", "email", "fax", "voice_mail"
}

// All error responses (400, 401, 404, 500) use the "message" field
type ErrorResponse struct {
	Message string `json:"message"`
}

type PresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileURL   string `json:"file_url"`
}
