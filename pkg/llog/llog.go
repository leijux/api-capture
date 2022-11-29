package llog

import (
	"changeme/pkg/config"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init(cfg *config.Config) {
	log.Logger = zerolog.
		New(zerolog.NewConsoleWriter()).
		With().
		Caller().
		Timestamp().
		Logger()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}
