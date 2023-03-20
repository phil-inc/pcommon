package ndc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test for converting Zoho SKU to NDC
func TestFromZohoSKU(t *testing.T) {
	// SKU length is 11
	assert.Equal(t, "59630-0580-90", FromZohoSKU("59630058090"))
	assert.Equal(t, "59630-0580-90", FromZohoSKU("59630-0580-90"))

	// SKU length is 12
	assert.Equal(t, "00378-8082-20", FromZohoSKU("303788082207"))
	assert.Equal(t, "51862-0462-60", FromZohoSKU("351862462605"))

	// default
	assert.Equal(t, "1234", FromZohoSKU("1234"))
}
