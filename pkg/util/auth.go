package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/phil-inc/pcommon/pkg/network"
	"github.com/pkg/errors"
)

var ErrInvalidToken = errors.New("token is invalid")

type LocationData struct {
	CountryCode string  `json:"countryCode" bson:"countryCode,omitempty"`
	CountryName string  `json:"countryName" bson:"countryName,omitempty"`
	RegionCode  string  `json:"regionCode" bson:"regionCode,omitempty"`
	RegionName  string  `json:"regionName" bson:"regionName,omitempty"`
	City        string  `json:"city" bson:"city,omitempty"`
	Zip         string  `json:"zip" bson:"zip,omitempty"`
	Latitude    float64 `json:"latitude" bson:"latitude,omitempty"`
	Longitude   float64 `json:"longitude" bson:"longitude,omitempty"`
}

func ValidateToken(token string, publicKey string) (jwt.MapClaims, error) {

	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected method: %s", t.Header["alg"])
		}

		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))
		if err != nil {
			return nil, err
		}
		return key, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetHeaderValue retrieves the first value of the specified header key from *http.Request
func GetHeaderValue(r *http.Request, key string) string {
	headers := r.Header[key]
	if len(headers) > 0 {
		return headers[0]
	}

	return ""
}

// GetUserLocationDetailsUsingIP fetches location data for the provided IP address using ipstack
func GetUserLocationDetailsUsingIP(ip, accessKey string) (*LocationData, error) {

	serviceURL := fmt.Sprintf("http://api.ipstack.com/%s?access_key=%s", ip, accessKey)
	res, err := network.GetWithTimeout(serviceURL, nil, 5)
	if err != nil {
		return nil, err
	}

	locationDetails := LocationData{}

	err = json.Unmarshal(res, &locationDetails)
	if err != nil {
		return nil, err
	}

	return &locationDetails, nil
}

// GetRemoteIP retrieves the real IP address of the client from *http.Request
// It checks the "X-Forwarded-For" and "X-Real-Ip" headers and returns the first public IP found
func GetRemoteIP(r *http.Request) string {
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		addresses := strings.Split(r.Header.Get(h), ",")
		// match from right to left until we get a public address
		// that will be the address right before our proxy.
		for i := len(addresses) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(addresses[i])
			// header can contain spaces too, strip those out.
			realIP := net.ParseIP(ip)
			if !realIP.IsGlobalUnicast() || isPrivateSubnet(realIP) {
				// bad address, go to next
				continue
			}

			return ip
		}
	}

	return ""
}

// ipRange - a structure that holds the start and end of a range of ip addresses
type ipRange struct {
	start net.IP
	end   net.IP
}

// isPrivateSubnet - check to see if this ip is in a private subnet
func isPrivateSubnet(ipAddress net.IP) bool {
	// my use case is only concerned with ipv4 atm
	if ipCheck := ipAddress.To4(); ipCheck != nil {
		// iterate over all our ranges
		for _, r := range privateRanges {
			// check if this ip is in a private range
			if inRange(r, ipAddress) {
				return true
			}
		}
	}
	return false
}

// IP ranges to filter out private sub-nets, as well as multi-cast address space, and localhost address space
// Reference - https://whatismyipaddress.com/private-ip
var privateRanges = []ipRange{
	{
		start: net.ParseIP("10.0.0.0"),
		end:   net.ParseIP("10.255.255.255"),
	},
	{
		start: net.ParseIP("100.64.0.0"),
		end:   net.ParseIP("100.127.255.255"),
	},
	{
		start: net.ParseIP("172.16.0.0"),
		end:   net.ParseIP("172.31.255.255"),
	},
	{
		start: net.ParseIP("192.0.0.0"),
		end:   net.ParseIP("192.0.0.255"),
	},
	{
		start: net.ParseIP("192.168.0.0"),
		end:   net.ParseIP("192.168.255.255"),
	},
	{
		start: net.ParseIP("198.18.0.0"),
		end:   net.ParseIP("198.19.255.255"),
	},
}

// inRange - check to see if a given ip address is within a range given
func inRange(r ipRange, ipAddress net.IP) bool {
	// strcmp type byte comparison
	if bytes.Compare(ipAddress, r.start) >= 0 && bytes.Compare(ipAddress, r.end) < 0 {
		return true
	}
	return false
}
