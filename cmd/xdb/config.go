package main

import (
	"context"

	"github.com/gojekfarm/xtools/xload"
)

type Config struct {
	Addr string `env:"ADDR"`
}

func NewDefaultConfig() Config {
	cfg := Config{}

	cfg.Addr = ":8080"

	return cfg
}

func LoadConfig(ctx context.Context) (*Config, error) {
	cfg := NewDefaultConfig()
	err := xload.Load(ctx, &cfg, xload.SerialLoader(xload.OSLoader()))

	return &cfg, err
}
