package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

type Config interface{}

func prepare(name string) (*viper.Viper, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	envPrefix := fmt.Sprintf("PLANTD_%s", strings.ToUpper(name))
	envConfig := fmt.Sprintf("%s_CONFIG", envPrefix)

	config := viper.New()

	file := os.Getenv(envConfig)
	if file == "" {
		config.SetConfigName(name)
		config.AddConfigPath(".")
		config.AddConfigPath(fmt.Sprintf("%s/.config/plantd", home))
		config.AddConfigPath("/etc/plantd")
	} else {
		var extension string
		regex := regexp.MustCompile("((y(a)?ml)|json|toml)$")
		base := filepath.Base(file)
		if regex.Match([]byte(base)) {
			// strip the file type for viper
			parts := strings.Split(filepath.Base(file), ".")
			base = strings.Join(parts[:len(parts)-1], ".")
			extension = parts[len(parts)-1]
		} else {
			return nil, errors.New("configuration does not support that extension type")
		}
		config.SetConfigName(base)
		config.SetConfigType(extension)
		config.SetConfigFile(file)
		config.AddConfigPath(filepath.Dir(file))
	}

	return config, nil
}

func LoadConfigWithDefaults(name string, c interface{}, defaults map[string]interface{}) error {
	envPrefix := fmt.Sprintf("PLANTD_%s", strings.ToUpper(name))

	config, err := prepare(name)
	if err != nil {
		return err
	}

	err = config.ReadInConfig()
	if err != nil {
		return err
	}

	config.SetEnvPrefix(envPrefix)
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AutomaticEnv()

	for key, value := range defaults {
		fmt.Printf("Setting default: %s = %s\n", key, value)
		config.SetDefault(key, value)
	}

	err = config.Unmarshal(&c)
	if err != nil {
		return err
	}

	return nil
}

// LoadConfig reads in a configuration file from a set of locations and
// deserializes it into a Config instance.
func LoadConfig(name string, c interface{}) error {
	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	envPrefix := fmt.Sprintf("PLANTD_%s", strings.ToUpper(name))
	envConfig := fmt.Sprintf("%s_CONFIG", envPrefix)

	config := viper.New()

	file := os.Getenv(envConfig)
	if file == "" {
		config.SetConfigName(name)
		config.AddConfigPath(".")
		config.AddConfigPath(fmt.Sprintf("%s/.config/plantd", home))
		config.AddConfigPath("/etc/plantd")
	} else {
		var extension string
		regex := regexp.MustCompile("((y(a)?ml)|json|toml)$")
		base := filepath.Base(file)
		if regex.Match([]byte(base)) {
			// strip the file type for viper
			parts := strings.Split(filepath.Base(file), ".")
			base = strings.Join(parts[:len(parts)-1], ".")
			extension = parts[len(parts)-1]
		} else {
			return errors.New("configuration does not support that extension type")
		}
		config.SetConfigName(base)
		config.SetConfigType(extension)
		config.SetConfigFile(file)
		config.AddConfigPath(filepath.Dir(file))
	}

	err = config.ReadInConfig()
	if err != nil {
		return err
	}

	config.SetEnvPrefix(envPrefix)
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AutomaticEnv()

	err = config.Unmarshal(&c)
	if err != nil {
		return err
	}

	return nil
}

// MarshalConfig converts a Config instance to a JSON string.
func MarshalConfig(c Config) (string, error) {
	bytes, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
