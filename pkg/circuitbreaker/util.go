package circuitbreaker

import (
	"fmt"
	"net/url"
	"regexp"
	"time"
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

// parseOrDefault attempts to parse a value from Redis, returning the parsed value if successful.
// If parsing fails, it returns the provided default value.
func parseOrDefault[T any](parseFunc func(string) (T, error), value string, defaultValue T) T {
	parsedValue, err := parseFunc(value)
	if err != nil {
		return defaultValue
	}
	return parsedValue
}

// parseTimeOrDefault attempts to parse a time value from Redis. If parsing fails, it returns the provided default time.
func parseTimeOrDefault(layout, value string, defaultValue time.Time) time.Time {
	parsedTime, err := time.Parse(layout, value)
	if err != nil {
		return defaultValue
	}
	return parsedTime
}
