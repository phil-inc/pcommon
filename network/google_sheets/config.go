package google_sheets

//GoogleCreds google service account credentials
type GoogleCreds struct {
	Type            string
	ProjectID       string
	PrivateKeyID    string
	PrivateKey      string
	ClientID        string
	ClientEmail     string
	AuthURI         string
	TokenURI        string
	ProviderCertURI string
	ClientCertURI   string
}
