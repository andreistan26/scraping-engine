package cli

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type CliFlags struct {
	LogLevel string
}

type CliOptions struct {
	Logger *zap.SugaredLogger
}

type CliOption func(*CliOptions) error

func WithDefaultLogger(level string) CliOption {
	return func(copts *CliOptions) error {
		if copts.Logger != nil {
			return nil
		}

		defaultLevel := zapcore.DebugLevel
		
		switch level {
		case "info":
			defaultLevel = zapcore.InfoLevel
		case "warn":
			defaultLevel = zapcore.WarnLevel
		default:
			defaultLevel = zap.DebugLevel
		}

		encoderCfg := zap.NewDevelopmentEncoderConfig()
		encoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout(time.TimeOnly)

		config := zap.Config{
			Level:             zap.NewAtomicLevelAt(defaultLevel),
			Development:       true,
			DisableCaller:     true,
			DisableStacktrace: true,
			OutputPaths: []string{
				"stderr",
			},
			ErrorOutputPaths: []string{
				"stderr",
			},
			Encoding:      "console",
			EncoderConfig: encoderCfg,
		}

		copts.Logger = zap.Must(config.Build()).Sugar()

		return nil
	}
}
