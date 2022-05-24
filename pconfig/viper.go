package pconfig

import (
	"github.com/spf13/viper"
)

//viper used as the configuration framework
var vi = viper.New()

func GetString(key string) string {
	return vi.GetString(key)
}
