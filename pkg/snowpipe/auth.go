package snowpipe

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	jwtLife    = 59 * time.Minute // Snowflake allows at most an hour
	scopedLife = 30 * time.Minute
)

// auth signs key-pair JWTs for the account host and keeps the ingest host's scoped token.
type auth struct {
	cfg         *Config
	http        *http.Client
	base        string
	key         *rsa.PrivateKey
	fingerprint string

	// lock guards the fields below. It is a channel so a caller waiting behind a slow refresh
	// can give up when its context ends.
	lock     chan struct{}
	ingest   string
	scoped   string
	scopedAt time.Time
}

func parseKey(pemText string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemText))
	if block == nil {
		return nil, errors.New("not a PEM block")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("not an unencrypted PKCS#8 key")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA key")
	}
	return key, nil
}

// jwt signs a key-pair token. The account is the part before the first dot, so a locator with
// a region works too.
func (a *auth) jwt() (string, error) {
	account, _, _ := strings.Cut(strings.ToUpper(a.cfg.Account), ".")
	user := account + "." + strings.ToUpper(a.cfg.User)
	now := time.Now()
	claims, err := json.Marshal(map[string]any{
		"iss": user + "." + a.fingerprint,
		"sub": user,
		"iat": now.Unix(),
		"exp": now.Add(jwtLife).Unix(),
	})
	if err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding
	signed := enc.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." + enc.EncodeToString(claims)
	sum := sha256.Sum256([]byte(signed))
	sig, err := rsa.SignPKCS1v15(nil, a.key, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed + "." + enc.EncodeToString(sig), nil
}

// control sends a request to the account host with a fresh JWT.
func (a *auth) control(ctx context.Context, method, path, contentType string, body io.Reader) (*http.Response, error) {
	token, err := a.jwt()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Snowflake-Authorization-Token-Type", "KEYPAIR_JWT")
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return a.http.Do(req)
}

func (a *auth) hostname(ctx context.Context) (string, error) {
	resp, err := a.control(ctx, http.MethodGet, "/v2/streaming/hostname", "", nil)
	if err != nil {
		return "", fmt.Errorf("get ingest host: %w", err)
	}
	body, err := readBody(resp)
	if err != nil {
		return "", fmt.Errorf("get ingest host: %w", err)
	}
	// Snowflake documents that the ingest host uses dashes where the account has underscores.
	host := strings.ReplaceAll(strings.TrimSpace(string(body)), "_", "-")
	if host == "" {
		return "", errors.New("get ingest host: empty response")
	}
	return host, nil
}

// ingestToken returns the ingest base URL and a scoped token. stale is a token the ingest host
// rejected with 401; it is replaced unless another request already did so.
func (a *auth) ingestToken(ctx context.Context, stale string) (base, token string, err error) {
	select {
	case a.lock <- struct{}{}:
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
	defer func() { <-a.lock }()

	if a.scoped != "" && a.scoped != stale && time.Since(a.scopedAt) < scopedLife {
		return a.ingest, a.scoped, nil
	}
	if a.ingest == "" {
		host, err := a.hostname(ctx)
		if err != nil {
			return "", "", err
		}
		a.ingest = "https://" + host
	}
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"scope":      {strings.TrimPrefix(a.ingest, "https://")},
	}
	resp, err := a.control(ctx, http.MethodPost, "/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("get scoped token: %w", err)
	}
	body, err := readBody(resp)
	if err != nil {
		return "", "", fmt.Errorf("get scoped token: %w", err)
	}
	scoped, err := parseScopedToken(body)
	if err != nil {
		return "", "", fmt.Errorf("get scoped token: %w", err)
	}
	a.scoped, a.scopedAt = scoped, time.Now()
	return a.ingest, a.scoped, nil
}

// parseScopedToken accepts both {"token": ...}, as in the reference, and the bare token the
// tutorial shows.
func parseScopedToken(body []byte) (string, error) {
	token := strings.TrimSpace(string(body))
	if strings.HasPrefix(token, "{") {
		var v struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(body, &v); err != nil {
			return "", err
		}
		token = v.Token
	}
	if token == "" {
		return "", errors.New("empty token")
	}
	return token, nil
}
