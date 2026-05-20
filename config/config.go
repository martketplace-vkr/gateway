package config

import (
	serverCmp "github.com/martketplace-vkr/gateway/internal/app/cmps/server"

	"github.com/martketplace-vkr/pkg/server/http"
	"github.com/martketplace-vkr/pkg/utils/duration"
)

type GRPCClient struct {
	Host    string           `validate:"required"`
	Timeout duration.Seconds `validate:"required" default:"5"`
}

type Config struct {
	HTTP         http.Config      `validate:"required"`
	ServerCmp    serverCmp.Config `validate:"required"`
	Auth         GRPCClient       `validate:"required"`
	Cart         GRPCClient       `validate:"required"`
	Catalog      GRPCClient       `validate:"required"`
	Order        GRPCClient       `validate:"required"`
	Analytics    GRPCClient       `validate:"required"`
	User         GRPCClient       `validate:"required"`
	Balance      GRPCClient       `validate:"required"`
	Media        GRPCClient       `validate:"required"`
	Review       GRPCClient       `validate:"required"`
	MockProvider MockProviderConfig
}

type MockProviderConfig struct {
	InternalURL   string `default:"http://mock-payment-provider:8010"`
	WebhookSecret string `default:"dev-mock-provider-secret"`
}
