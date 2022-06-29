package koanf

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/knadh/koanf"
)

var Config = koanf.New(".")

func LoadConfig() error {
	profile := flag.String("profile", "local", "-profile=local")
	flag.Parse()
	// override profile value from env variable if available
	profileFromEnv := os.Getenv("ENROLL_PROFILE")
	if _, exist := os.LookupEnv("ENROLL_PROFILE"); !exist {
		return errors.New("[phil-enroll] ENROLL_PROFILE Not found")
	}

	if profileFromEnv != "" {
		profile = &profileFromEnv
	}
	log.Printf("[phil-enroll] Picking configuration: %s/%s.json", ".", *profile)

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
	os.Setenv("ENROLL_PROFILE", "config")
	os.Setenv("DEV", "XXXX")
	os.Setenv("DATA_DASHBOARD_ENDPOINT", "http://dataDashTest")
	LoadConfig()
	fmt.Println(Config.All())
	fmt.Println(Config.Get("placeholder.value"), " ===>>> ")
	fmt.Println(Config.Get("simple.value"), " ===>>> ")
	fmt.Println(Config.Get("placeHolderFallBack.value"), " ===>>> ")
}
