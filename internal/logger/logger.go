package logger

import (
	"gowhatsmeow/internal/config"

	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewZapLogger initializes a zap.Logger based on the configuration
func NewZapLogger(cfg *config.Config) (*zap.Logger, error) {
	var zapCfg zap.Config
	if cfg.Logger.Encoding == "json" {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Logger.Level)); err != nil {
		level = zapcore.InfoLevel
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	return zapCfg.Build()
}

type ZapWAAdapter struct {
	sugar *zap.SugaredLogger
}

func NewZapWAAdapter(zapLog *zap.Logger) waLog.Logger {
	return &ZapWAAdapter{
		sugar: zapLog.Sugar().With("system", "whatsmeow"),
	}
}

func (z *ZapWAAdapter) Debugf(format string, args ...interface{}) {
	z.sugar.Debugf(format, args...)
}

func (z *ZapWAAdapter) Infof(format string, args ...interface{}) {
	z.sugar.Infof(format, args...)
}

func (z *ZapWAAdapter) Warnf(format string, args ...interface{}) {
	z.sugar.Warnf(format, args...)
}

func (z *ZapWAAdapter) Errorf(format string, args ...interface{}) {
	z.sugar.Errorf(format, args...)
}

func (z *ZapWAAdapter) Sub(module string) waLog.Logger {
	return &ZapWAAdapter{
		sugar: z.sugar.With("module", module),
	}
}
