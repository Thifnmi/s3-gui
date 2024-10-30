//go:build !windows

package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type AppConfig struct {
	Region    string `json:"region"`
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

var (
	instance *AppConfig = nil
	once     sync.Once
)

func getEnv(key string, fallback interface{}) interface{} {
	var rValue interface{}
	value, exists := os.LookupEnv(key)
	if !exists {
		rValue = fallback
	} else {
		rValue = value
	}
	switch fallback.(type) {
	case int:
		if intValue, err := strconv.Atoi(value); err == nil {
			rValue = intValue
		} else {
			rValue = fallback
		}
	case time.Duration:
		if durationValue, err := time.ParseDuration(value); err == nil {
			rValue = durationValue
		} else {
			rValue = fallback
		}
	case bool:
		if boolValue, err := strconv.ParseBool(value); err == nil {
			rValue = boolValue
		} else {
			rValue = fallback
		}
	}
	return rValue
}

func InitConfig() *AppConfig {
	once.Do(
		func() {
			err := godotenv.Load(filepath.Join(os.Getenv("HOME"), "Documents", "s3-uploader", "config.json"))
			if err != nil {
				log.Printf("Error: %s", err)
			}

			instance = &AppConfig{
				Region:    getEnv("REGION", "us-east-1").(string),
				Endpoint:  getEnv("ENDPOINT", "https://s3.amazonaws.com").(string),
				AccessKey: getEnv("ACCESS_KEY", "access-key").(string),
				SecretKey: getEnv("SECRET_KEY", "secret-key").(string),
			}
		},
	)

	return instance
}
