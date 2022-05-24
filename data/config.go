package data

//GoogleCreds google service account credentials
type GoogleCreds struct {
	Type            string `json:"type" bson:"type"`
	ProjectID       string `json:"project_id" bson:"project_id"`
	PrivateKeyID    string `json:"private_key_id" bson:"private_key_id"`
	PrivateKey      string `json:"private_key" bson:"private_key"`
	ClientID        string `json:"client_id" bson:"client_id"`
	ClientEmail     string `json:"client_email" bson:"client_email"`
	AuthURI         string `json:"auth_uri" bson:"auth_uri"`
	TokenURI        string `json:"token_uri" bson:"token_uri"`
	ProviderCertURI string `json:"auth_provider_x509_cert_url" bson:"auth_provider_x509_cert_url"`
	ClientCertURI   string `json:"client_x509_cert_url" bson:"client_x509_cert_url"`
}
