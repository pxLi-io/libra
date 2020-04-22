package libra

import (
	"flag"
	"os"

	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Name = flag.String("libra.name", "localhost", "node name for libra member")
	Port = flag.Int("libra.port", 7946, "the port used for both UDP and TCP gossip")
)

type Config struct {
	*memberlist.Config

	Seeds []string
}

func NewSugar() *zap.SugaredLogger {
	encConf := zap.NewProductionEncoderConfig()
	encConf.EncodeTime = zapcore.ISO8601TimeEncoder
	logger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(encConf), os.Stdout, zap.InfoLevel))
	return logger.Sugar()
}
