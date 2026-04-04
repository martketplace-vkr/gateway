package main

import (
	"context"
	"log"

	"github.com/martketplace-vkr/gateway/config"
	// _ "github.com/martketplace-vkr/gateway/docs"
	"github.com/martketplace-vkr/gateway/internal/app"

	cfgloader "github.com/martketplace-vkr/pkg/config"
)

// @title		Marketplace VKR API
// @version	1.0
// @host		localhost:8000
// @BasePath	/
func main() {
	ctx := context.Background()
	cfg := &config.Config{}

	if err := cfgloader.LoadConfig(ctx, cfg); err != nil {
		log.Fatal(err.Error())
	}

	if err := app.Run(ctx, cfg); err != nil {
		log.Fatal(err.Error())
	}
}
