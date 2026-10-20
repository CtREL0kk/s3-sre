package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const ServiceName = "myS3"

func New(level string) (*zap.SugaredLogger, error) {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return nil, err
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "timestamp"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encCfg),
		zapcore.AddSync(os.Stdout),
		zap.NewAtomicLevelAt(lvl),
	)

	zl := zap.New(core, zap.AddCaller()).
		With(zap.String("service.name", ServiceName))

	return zl.Sugar(), nil
}
