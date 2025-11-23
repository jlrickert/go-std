package toolkit

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"

	"github.com/jlrickert/cli-toolkit/mylog"
)

func getTookitLogger(ctx context.Context) *slog.Logger {
	lg := mylog.LoggerFromContext(ctx)
	stackTrace := getStackTrace(2)
	return lg.With(
		slog.String("package", "toolkit"),
		slog.String("stackTrace", stackTrace),
	)
}

// getStackTrace returns a formatted stack trace string, skipping the
// specified number of stack frames. skip=1 returns from the immediate
// caller, skip=2 skips one level, etc.
func getStackTrace(skip int) string {
	var buf strings.Builder
	pcs := make([]uintptr, 10)
	n := runtime.Callers(skip+1, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		fmt.Fprintf(&buf, "%s\n\t%s:%d\n", frame.Function,
			frame.File, frame.Line)
		if !more {
			break
		}
	}
	return buf.String()
}
