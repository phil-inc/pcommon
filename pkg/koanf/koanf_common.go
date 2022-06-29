package koanf

import (
	"log"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
)

func LoadConfigFromFile(fn string, config *koanf.Koanf) error {
	fp := file.Provider(fn)
	err := config.Load(fp, json.Parser())
	if err != nil {
		return err
	}

	err = watchConfig(fp, config)
	if err != nil {
		return err
	}

	return nil
}

func LoadSystemEnvVariables(config *koanf.Koanf) error {
	jsonConfig := config.All()

	for k, v := range jsonConfig {
		strV := v.(string)
		if strings.HasPrefix(strV, "${") && strings.HasSuffix(strV, "}") {
			strV := strings.Trim(strV, "${")
			strV = strings.Trim(strV, "}")
			strVArr := strings.Split(strV, "|")
			if len(strVArr) == 1 {
				err := config.Load(env.Provider(strV, ".", func(s string) string {
					return k
				}), nil)

				if err != nil {
					return err
				}
			} else {
				err := config.Load(env.Provider(strVArr[0], ".", func(s string) string {
					if len(s) == 0 {
						return strVArr[1]
					} else {
						return k
					}
				}), nil)

				if err != nil {
					return err
				}
			}

		}
	}

	return nil
}

func watchConfig(fp *file.File, config *koanf.Koanf) error {
	err := fp.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Printf("watch error: %v", err)
			return
		}
		// Throw away the old config and load a fresh copy.
		log.Println("config changed. Reloading ...")
		config = koanf.New(".")
		config.Load(fp, json.Parser())
	})

	if err != nil {
		return err
	}

	return nil
}
