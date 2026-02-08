package engine

import (
	"fmt"
	"io"
	"os"
	"time"
)

// DebugLevel defines verbosity
type DebugLevel int

const (
	DebugNone DebugLevel = iota
	DebugSQL
	DebugTrace
	DebugExplain
)

// DebugContext holds debug configuration
type DebugContext struct {
	Level  DebugLevel
	Writer io.Writer // Where to write (stdout, file, etc)

	// Future expansion
	EnableTiming    bool
	EnableProfiling bool
	ColorOutput     bool
}

// DefaultDebugContext for production
func DefaultDebugContext() *DebugContext {
	return &DebugContext{
		Level:       DebugNone,
		Writer:      os.Stdout,
		ColorOutput: true,
	}
}

// DebugContextFromEnv reads from environment
func DebugContextFromEnv() *DebugContext {
	level := DebugNone

	if os.Getenv("CHAMELEON_DEBUG") == "1" {
		level = DebugSQL
	}
	if os.Getenv("CHAMELEON_DEBUG") == "trace" {
		level = DebugTrace
	}
	if os.Getenv("CHAMELEON_DEBUG") == "explain" {
		level = DebugExplain
	}

	return &DebugContext{
		Level:       level,
		Writer:      os.Stdout,
		ColorOutput: true,
	}
}

// Log writes debug output
func (dc *DebugContext) Log(level DebugLevel, format string, args ...interface{}) {
	if dc.Level < level {
		return
	}

	var prefix string
	if dc.ColorOutput {
		prefix = colorPrefix(level)
	} else {
		prefix = textPrefix(level)
	}

	fmt.Fprintf(dc.Writer, prefix+format+"\n", args...)
}

// LogSQL logs generated SQL
func (dc *DebugContext) LogSQL(sql string) {
	if dc.Level < DebugSQL {
		return
	}

	if dc.ColorOutput {
		fmt.Fprintf(dc.Writer, "\n\033[36m[SQL]\033[0m\n%s\n\n", sql)
	} else {
		fmt.Fprintf(dc.Writer, "\n[SQL]\n%s\n\n", sql)
	}
}

// LogQuery logs full query trace
func (dc *DebugContext) LogQuery(sql string, duration time.Duration, rowCount int) {
	if dc.Level < DebugTrace {
		return
	}

	fmt.Fprintf(dc.Writer, "\n")
	fmt.Fprintf(dc.Writer, "┌─────────────────────────────────────\n")
	fmt.Fprintf(dc.Writer, "│ Query Trace\n")
	fmt.Fprintf(dc.Writer, "├─────────────────────────────────────\n")
	fmt.Fprintf(dc.Writer, "│ SQL:\n│   %s\n", sql)
	fmt.Fprintf(dc.Writer, "│ Duration: %v\n", duration)
	fmt.Fprintf(dc.Writer, "│ Rows: %d\n", rowCount)
	fmt.Fprintf(dc.Writer, "└─────────────────────────────────────\n\n")
}

func colorPrefix(level DebugLevel) string {
	switch level {
	case DebugSQL:
		return "\033[36m[DEBUG]\033[0m "
	case DebugTrace:
		return "\033[33m[TRACE]\033[0m "
	case DebugExplain:
		return "\033[35m[EXPLAIN]\033[0m "
	default:
		return ""
	}
}

func textPrefix(level DebugLevel) string {
	switch level {
	case DebugSQL:
		return "[DEBUG] "
	case DebugTrace:
		return "[TRACE] "
	case DebugExplain:
		return "[EXPLAIN] "
	default:
		return ""
	}
}
