package main

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	APIKey        string   `mapstructure:"api_key"`
	Records       []Record `mapstructure:"dns_records"`
	CheckInterval int      `mapstructure:"check_interval"`
}

type Record struct {
	Domain   string   `mapstructure:"domain"`
	ZoneID   string   `mapstructure:"zone_id"`
	Proxied  bool     `mapstructure:"proxied"`
	IPs      []string `mapstructure:"ips"`
	Protocol string   `mapstructure:"protocol"`
	Port     int      `mapstructure:"port"`
}

func parseConfig() (Config, error) {
	var configPath string
	pflag.StringVarP(&configPath, "config", "c", "", "Path to config file")
	pflag.Parse()

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
	}

	var cfg Config

	err := viper.ReadInConfig()
	if err != nil {
		return cfg, fmt.Errorf("Error reading config: %v", err)
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return cfg, fmt.Errorf("Unable to decode config into struct, %v", err)
	}

	// Apply defaults if missing
	for i := range cfg.Records {
		if cfg.Records[i].Protocol == "" {
			cfg.Records[i].Protocol = "http"
		}
		if cfg.Records[i].Port == 0 {
			if cfg.Records[i].Protocol == "https" {
				cfg.Records[i].Port = 443
			} else {
				cfg.Records[i].Port = 80
			}
		}
	}
	return cfg, nil
}
