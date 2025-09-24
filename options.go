package go11y

import (
	"log/slog"
)

func defaultOptions(cfg Configurator) *slog.HandlerOptions {
	ho := &slog.HandlerOptions{
		AddSource:   true,
		Level:       cfg.LogLevel(),
		ReplaceAttr: defaultReplacer(cfg.TrimModules(), cfg.TrimPaths()),
	}

	return ho
}
