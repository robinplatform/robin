package log

import (
	"io"
	"os"
	"path"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"robin.dev/internal/config"
)

// Lumberjack implements log file rotation
var writer io.Writer

func init() {
	// If we fail to get a release channel, we are defaulting to
	// the stable channel anyways, so select that for logging too
	channel, _ := config.GetReleaseChannel()

	writer = &lumberjack.Logger{
		Filename:   path.Join(channel.GetPath(), "logs.db"),
		MaxSize:    512, // Megabytes
		MaxBackups: 1,
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

type Ctx map[string]any
type Logger struct {
	zero      zerolog.Logger
	namespace string
}

func New(namespace string) Logger {
	return Logger{
		zero:      log.Output(writer),
		namespace: namespace,
	}
}
