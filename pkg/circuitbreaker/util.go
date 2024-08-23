package circuitbreaker

import (
	"fmt"
	"net/url"
	"regexp"
)

// normalizeURL normalizes the given URL by replacing path parameters with a placeholder.
func normalizeURL(rawURL string) string {
	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Normalize path by replacing numeric or alphanumeric segments with a placeholder
	path := parsedURL.Path
	re := regexp.MustCompile(`\/\d+|\{\w+\}`)
	normalizedPath := re.ReplaceAllString(path, "/{param}")

	// Rebuild the URL with the normalized path
	normalizedURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, normalizedPath)

	return normalizedURL
}
