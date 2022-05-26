package google_sheets

//GoogleCreds google service account credentials
type GoogleCreds struct {
	Type            string `json:"type"`
	ProjectID       string `json:"project_id"`
	PrivateKeyID    string `json:"private_key_id"`
	PrivateKey      string `json:"private_key"`
	ClientID        string `json:"client_id"`
	ClientEmail     string `json:"client_email"`
	AuthURI         string `json:"auth_uri"`
	TokenURI        string `json:"token_uri"`
	ProviderCertURI string `json:"auth_provider_x509_cert_url"`
	ClientCertURI   string `json:"client_x509_cert_url"`
}
