package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	"robinplatform.dev/internal/config"
)

var (
	encoder         *json.Encoder
	debugNamespaces map[string]bool
)

func init() {
	channel := config.GetReleaseChannel()

	// Lumberjack implements log file rotation
	writer := &lumberjack.Logger{
		Filename:   path.Join(channel.GetPath(), "logs.db"),
		MaxSize:    512, // Megabytes
		MaxBackups: 1,
	}
	encoder = json.NewEncoder(writer)

	// Find all namespaces that should be logged in debug mode
	debugNamespaceStrings := strings.Split(os.Getenv("ROBIN_DEBUG"), ",")
	debugNamespaces = make(map[string]bool, len(debugNamespaceStrings))
	for _, namespace := range debugNamespaceStrings {
		debugNamespaces[namespace] = true
	}
}

type Level string

const (
	Debug Level = "debug"
	Info  Level = "info"
	Print Level = "print"
	Warn  Level = "warn"
	Error Level = "error"
)

type Ctx map[string]any

type Logger struct {
	namespace string
	color     int
}

func New(namespace string) Logger {
	return Logger{
		namespace: namespace,
		color:     randColor(namespace),
	}
}

func (logger *Logger) log(level Level, msg string, ctx Ctx) {
	if level == Debug && !debugNamespaces[logger.namespace] {
		return
	}

	log := make(map[string]any, 4+len(ctx))
	log["timestamp"] = time.Now().UnixMilli()
	log["level"] = string(level)
	log["namespace"] = logger.namespace
	log["message"] = msg

	for key, value := range ctx {
		log[key] = value
	}

	ctxBuf, err := json.MarshalIndent(ctx, "\t", "\t")
	if err != nil {
		panic(err)
	}

	levelStr := levelStrings[string(level)]
	fmt.Printf("%s %s %s\n\t%s\n", levelStr, color(logger.color, logger.namespace), msg, ctxBuf)
	encoder.Encode(log)
}

func (logger *Logger) Debug(msg string, ctx Ctx) {
	logger.log(Debug, msg, ctx)
}

func (logger *Logger) Print(msg string, ctx Ctx) {
	logger.log(Print, msg, ctx)
}

func (logger *Logger) Info(msg string, ctx Ctx) {
	logger.log(Info, msg, ctx)
}

func (logger *Logger) Warn(msg string, ctx Ctx) {
	logger.log(Warn, msg, ctx)
}

func (logger *Logger) Err(msg string, ctx Ctx) {
	logger.log(Error, msg, ctx)
}
