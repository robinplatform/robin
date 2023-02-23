package log

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"robinplatform.dev/internal/config"
)

// Lumberjack implements log file rotation
var writer io.Writer

func init() {
	channel := config.GetReleaseChannel()

	writer = &lumberjack.Logger{
		Filename:   path.Join(channel.GetPath(), "logs.db"),
		MaxSize:    512, // Megabytes
		MaxBackups: 1,
	}

	console := zerolog.ConsoleWriter{Out: os.Stderr}
	console.FormatFieldName = func(i interface{}) string {
		return ""
	}
	console.FormatFieldValue = func(i interface{}) string {
		return ""
	}
	console.FormatExtra = func(value map[string]interface{}, buf *bytes.Buffer) error {
		ctx := make(map[string]any, len(value))
		for key, value := range value {
			switch key {
			case "level", "time", "message":
				continue
			}
			ctx[key] = value
		}

		data, err := json.MarshalIndent(ctx, "\t", "\t")
		if err != nil {
			return err
		}

		buf.WriteString("\n\t")
		_, err = buf.Write(data)
		return err
	}

	log.Logger = log.Output(console)
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
