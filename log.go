package libra

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewSugar() *zap.SugaredLogger {
	encConf := zap.NewProductionEncoderConfig()
	encConf.EncodeTime = zapcore.ISO8601TimeEncoder
	logger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(encConf), os.Stdout, zap.InfoLevel))
	return logger.Sugar()
}
