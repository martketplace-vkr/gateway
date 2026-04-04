package server

import "github.com/martketplace-vkr/pkg/build/components"

type Config struct {
	components.ComponentConfig `validate:"required"`
}
