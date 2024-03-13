package main

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	APIKey        string   `mapstructure:"api_key"`
	Records       []Record `mapstructure:"dns_records"`
	CheckInterval int      `mapstructure:"check_interval"`
}

type Record struct {
	Domain  string   `mapstructure:"domain"`
	ZoneID  string   `mapstructure:"zone_id"`
	Proxied bool     `mapstructure:"proxied"`
	IPs     []string `mapstructure:"ips"`
}

func parseConfig() (Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	var cfg Config

	err := viper.ReadInConfig()
	if err != nil {
		return cfg, fmt.Errorf("Error reading config: %v", err)
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return cfg, fmt.Errorf("Unable to decode config into struct, %v", err)
	}

	return cfg, nil
}
