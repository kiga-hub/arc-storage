package kafka

import (
	"github.com/kiga-hub/arc-storage/pkg/config"
	arcKafka "github.com/kiga-hub/arc/kafka"
	"github.com/kiga-hub/arc/logging"
)

// Option is a function that will set up option.
type Option func(opts *Server)

func loadOptions(options ...Option) *Server {
	opts := &Server{}
	for _, option := range options {
		option(opts)
	}
	if opts.logger == nil {
		opts.logger = new(logging.NoopLogger)
	}
	if opts.config == nil {
		opts.config = config.GetConfig()
	}
	return opts
}

// WithLogger -
func WithLogger(logger logging.ILogger) Option {
	return func(opts *Server) {
		opts.logger = logger
	}
}

// WithConfig -
func WithConfig(c *config.ArcConfig) Option {
	return func(opts *Server) {
		opts.config = c
	}
}

// WithKafka -
func WithKafka(k interface{}) Option {
	return func(opts *Server) {
		if k != nil {
			opts.kafka = k.(*arcKafka.Kafka)
		}
	}
}
