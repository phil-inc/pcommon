package koanf

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStringOrDefaultInCommaSeparatorWithEnvValue(t *testing.T) {
	expected := "SYS.DEV"
	os.Setenv("TEST_PROFILE", "config")
	os.Setenv("DEV", expected)
	os.Setenv("DATA_DEV", "http://dataDashTest")
	LoadConfig("TEST_PROFILE")
	if GetConfigValue("placeholder.value") != expected {
		t.Errorf("Expected value did not match with key %s\n", expected)
	}

	if GetConfigValue("simple.value") != "dev" {
		t.Errorf("Expected value did not match with key %s\n", expected)
	}

	if GetConfigValue("placeHolderFallBack.value") != expected {
		t.Errorf("Expected value did not match with key %s\n", expected)
	}

	if GetConfigValue("cors.allowedOrigin") != "http://dataDashTest,https://login.default.microsoftonline.com" {
		t.Errorf("Expected value did not match with key %s\n", expected)
	}
}

func TestGetStringOrDefaultInCommaSeparatorMissingEnvEnvValue(t *testing.T) {
	os.Setenv("TEST_PROFILE", "config")
	os.Setenv("DATA_DEV", "http://dataDashTest")
	LoadConfig("TEST_PROFILE")
	if GetStringConfigValue("placeholder.value") != "" {
		t.Errorf("Expected value did not match with key %s\n", "")
	}

	if GetStringConfigValue("simple.value") != "dev" {
		t.Errorf("Expected value did not match with key %s\n", "")
	}

	if GetStringConfigValue("placeHolderFallBack.value") != "default" {
		t.Errorf("Expected value did not match with key %s\n", "")
	}

	if GetStringConfigValue("cors.allowedOrigin") != "http://dataDashTest,https://login.default.microsoftonline.com" {
		t.Errorf("Expected value did not match with key %s\n", "")
	}
}

func TestNonStringDataTypes(t *testing.T) {
	os.Setenv("TEST_PROFILE", "config")
	LoadConfig("TEST_PROFILE")
	if !GetBooleanConfigValue("booleanDataType.value", false) {
		t.Errorf("Expected value did not match with key %s\n", "true")
	}

	if GetNumberConfigValue("numberDataType.value", 0) != 1 {
		t.Errorf("Expected value did not match with key %d\n", 1)
	}

	if !CheckValueExist("floatDataType.value") {
		t.Errorf("Expected value did not match with key %d\n", 1)
	}

	if CheckValueExist("floatDataType.someBS") {
		t.Errorf("Expected value did not match with key %d\n", 1)
	}

	if GetFloatConfigValue("floatDataType.value", 0.0) != 1.5 {
		t.Errorf("Expected value did not match with key %d\n", 1)
	}
}

func TestLoadConfigWithProvider(t *testing.T) {
	os.Setenv("TEST_PROFILE", "config")
	LoadConfig("TEST_PROFILE")
	mp := map[string]interface{}{
		"str": map[string]string{
			"k1": "value",
		},
		"strs": map[string][]string{
			"k1": {"value"},
		},
		"iface": map[string]interface{}{
			"k2": "value",
		},
		"ifaces": map[string][]interface{}{
			"k2": {"value"},
		},
		"ifaces2": map[string]interface{}{
			"k2": []interface{}{"value"},
		},
		"ifaces3": map[string]interface{}{
			"k2": []string{"value"},
		},
	}

	LoadConfigWithProvider(mp, ".")

	assert.Equal(t, map[string]string{"k1": "value"}, Config.StringMap("str"), "types don't match")
	assert.Equal(t, map[string]string{"k2": "value"}, Config.StringMap("iface"), "types don't match")
	assert.Equal(t, map[string][]string{"k1": {"value"}}, Config.StringsMap("strs"), "types don't match")
	assert.Equal(t, map[string][]string{"k2": {"value"}}, Config.StringsMap("ifaces"), "types don't match")
	assert.Equal(t, map[string][]string{"k2": {"value"}}, Config.StringsMap("ifaces2"), "types don't match")
	assert.Equal(t, map[string][]string{"k2": {"value"}}, Config.StringsMap("ifaces3"), "types don't match")
}
