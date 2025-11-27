package observability

import (
	"log/slog"
	"os"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func Logger() *slog.Logger {
	return logger
}
