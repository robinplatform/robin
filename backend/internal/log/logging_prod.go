//go:build prod

package log

func (l *Logger) Debug(msg string, ctx Ctx) {
	l.zero.Debug().Interface("data", ctx).Msg(msg)
}

func (l *Logger) Info(msg string, ctx Ctx) {
	l.zero.Info().Interface("data", ctx).Msg(msg)
}

func (l *Logger) Print(msg string, ctx Ctx) {
	log.Info().Fields(ctx).Msg(msg)
	l.zero.Info().Interface("data", ctx).Msg(msg)
}

func (l *Logger) Warn(msg string, ctx Ctx) {
	l.zero.Warn().Interface("data", ctx).Msg(msg)
	log.Warn().Fields(ctx).Msg(msg)
}

func (l *Logger) Err(e error, msg string, ctx Ctx) {
	l.zero.Err(e).Interface("data", ctx).Msg(msg)
	log.Warn().Fields(ctx).Msg(msg)
}
