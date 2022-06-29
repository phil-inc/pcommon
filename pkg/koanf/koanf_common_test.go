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
	LoadConfig()
	if GetConfigValue("placeholder.value", Config) != expected {
		t.Errorf("Expected value did not match with key %s\n", expected)
	}

	if GetConfigValue("simple.value", Config) != "dev" {
		t.Errorf("Expected value did not match with key %s\n", expected)
	}

	if GetConfigValue("placeHolderFallBack.value", Config) != expected {
		t.Errorf("Expected value did not match with key %s\n", expected)
	}

	if GetConfigValue("cors.allowedOrigin", Config) != "http://dataDashTest,https://login.default.microsoftonline.com" {
		t.Errorf("Expected value did not match with key %s\n", expected)
	}
}

func TestGetStringOrDefaultInCommaSeparatorMissingEnvEnvValue(t *testing.T) {
	os.Setenv("TEST_PROFILE", "config")
	os.Setenv("DATA_DEV", "http://dataDashTest")
	LoadConfig()
	if GetConfigValue("placeholder.value", Config) != "" {
		t.Errorf("Expected value did not match with key %s\n", "")
	}

	if GetConfigValue("simple.value", Config) != "dev" {
		t.Errorf("Expected value did not match with key %s\n", "")
	}

	if GetConfigValue("placeHolderFallBack.value", Config) != "default" {
		t.Errorf("Expected value did not match with key %s\n", "")
	}
	fmt.Println(GetConfigValue("cors.allowedOrigin", Config))

	if GetConfigValue("cors.allowedOrigin", Config) != "http://dataDashTest,https://login.default.microsoftonline.com" {
		t.Errorf("Expected value did not match with key %s\n", "")
	}
}
