package koanf

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/knadh/koanf"
)

var Config = koanf.New(".")

func LoadConfig() error {
	profile := flag.String("profile", "local", "-profile=local")
	flag.Parse()
	// override profile value from env variable if available
	profileFromEnv := os.Getenv("TEST_PROFILE")
	if profileFromEnv != "" {
		profile = &profileFromEnv
	}
	fn := fmt.Sprintf("%s/%s.json", ".", *profile)
	err := LoadConfigFromFile(fn, Config)
	if err != nil {
		return err
	}
	err = LoadSystemEnvVariables(Config)
	if err != nil {
		return err
	}
	return nil
}

func TestGetStringOrDefaultInCommaSeparatorWithEnvValue(t *testing.T) {
	expected := "SYS.DEV"
	os.Setenv("TEST_PROFILE", "config")
	os.Setenv("DEV", expected)
	os.Setenv("DATA_DASHBOARD_ENDPOINT", "http://dataDashTest")
	LoadConfig()

	if Config.String("placeholder.value") != expected {
		t.Errorf("Expected value did not match with key %s\n", expected)
	}

	if Config.String("simple.value") != "dev" {
		t.Errorf("Expected value did not match with key %s\n", expected)
	}

	if Config.String("placeHolderFallBack.value") != expected {
		t.Errorf("Expected value did not match with key %s\n", expected)

	}
}
