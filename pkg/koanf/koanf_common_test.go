package koanf

import (
	"fmt"
	"os"
	"testing"
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
	if GetConfigValue("placeholder.value") != "" {
		t.Errorf("Expected value did not match with key %s\n", "")
	}

	if GetConfigValue("simple.value") != "dev" {
		t.Errorf("Expected value did not match with key %s\n", "")
	}

	if GetConfigValue("placeHolderFallBack.value") != "default" {
		t.Errorf("Expected value did not match with key %s\n", "")
	}
	fmt.Println(GetConfigValue("cors.allowedOrigin"))

	if GetConfigValue("cors.allowedOrigin") != "http://dataDashTest,https://login.default.microsoftonline.com" {
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
