package koanf

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
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
	err := LoadConfigFromFile(fn)
	if err != nil {
		return err
	}

	return nil
}

func LoadConfigFromFile(fn string) error {
	fp := file.Provider(fn)
	err := Config.Load(fp, json.Parser())
	if err != nil {
		return err
	}

	err = watchConfig(fp)
	if err != nil {
		return err
	}

	return nil
}

func watchConfig(fp *file.File) error {
	err := fp.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Printf("watch error: %v", err)
			return
		}
		// Throw away the old config and load a fresh copy.
		log.Println("config changed. Reloading ...")
		Config = koanf.New(".")
		Config.Load(fp, json.Parser())
	})

	if err != nil {
		return err
	}

	return nil
}

// Use this method if your variable has to fetch data from system properties
func GetConfigValue(key string) interface{} {
	return replaceSysVars(key)
}

func GetBooleanConfigValue(key string, def bool) bool {
	v, e := strconv.ParseBool(replaceSysVars(key))
	if e == nil {
		return v
	} else {
		fmt.Println("Error converting", e)
		return def
	}
}

func GetNumberConfigValue(key string, def int64) int64 {
	v, e := strconv.ParseInt(replaceSysVars(key), 0, 0)
	if e == nil {
		return v
	} else {
		fmt.Println("Error converting", e)
		return def
	}
}

func GetFloatConfigValue(key string, def float64) float64 {
	v, e := strconv.ParseFloat(replaceSysVars(key), 0)
	if e == nil {
		return v
	} else {
		fmt.Println("Error converting", e)
		return def
	}
}

func replaceSysVars(key string) string {
	value := Config.String(key)
	re := regexp.MustCompile(`\${[^}]+}`)
	return re.ReplaceAllStringFunc(value, replaceSysVarsHelper)
}

func replaceSysVarsHelper(value string) string {
	value = strings.Replace(value, "${", "", 1)
	value = strings.Replace(value, "}", "", 1)
	parts := strings.Split(value, "|")
	v := ""
	if val := helper(fmt.Sprintf("${%s}", parts[0])); val != "" {
		v = val
	} else if len(parts) > 1 {
		v = parts[1]
	}
	return v
}

func helper(strV string) string {
	if strings.HasPrefix(strV, "${") && strings.HasSuffix(strV, "}") {
		mainKey := strings.TrimLeft(strings.TrimRight(strV, "}"), "${")
		keySplitter := strings.Split(mainKey, "|")
		if len(keySplitter) >= 2 {
			configValue := os.ExpandEnv(fmt.Sprintf("${%s}", keySplitter[0]))
			if len(configValue) == 0 {
				return keySplitter[1]
			}
			return configValue
		} else {
			return os.ExpandEnv(strV)
		}

	}

	return strV
}
