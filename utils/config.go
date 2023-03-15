package utils

import "github.com/spf13/viper"

var config *viper.Viper

func InitConf(fileName string) (err error) {
	config = viper.New()
	config.AddConfigPath(GetCurrentPath() + "conf/")
	config.SetConfigName(fileName)
	config.SetConfigType("yaml")

	if err := config.ReadInConfig(); err != nil {
		return err
	} else {
		return nil
	}
}

func ReadConfStrOrDef(key, defaultVal string) string {
	var val string
	if val = config.GetString(key); val == "" {
		val = defaultVal
	}
	return val
}

func ReadConfIntOrDef(key string, defaultVal int) int {
	var val int
	if val = config.GetInt(key); val == 0 {
		val = defaultVal
	}
	return val
}

func ReadConfStr(key string) string {
	return config.GetString(key)
}

func WriteConf(key string, value interface{}) error {
	config.Set(key, value)
	if err := config.WriteConfig(); err != nil {
		return err
	}

	return nil
}

func ReadConfBool(key string) bool {
	return config.GetBool(key)
}
