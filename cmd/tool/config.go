package main

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	WeatherKey       string `yaml:"weather-key" env:"WEATHER_KEY"`
	GeonamesUsername string `yaml:"geonames-username" env:"GEONAMES_USERNAME"`
}

func NewConfig(path string) Config {
	var cfg Config
	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		panic(err)
	}
	return cfg
}
