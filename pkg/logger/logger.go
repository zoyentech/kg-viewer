// Package logger is a thin printf-style logging interface backed by
// the standard library so the service has zero external log deps.
package logger

import "log"

// Logger is the interface consumed by handlers and services.
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

type stdLogger struct{}

// New returns a stdlib-backed Logger.
func New() Logger { return stdLogger{} }

func (stdLogger) Debugf(f string, a ...any) { log.Printf("DEBUG: "+f, a...) }
func (stdLogger) Infof(f string, a ...any)  { log.Printf("INFO:  "+f, a...) }
func (stdLogger) Warnf(f string, a ...any)  { log.Printf("WARN:  "+f, a...) }
func (stdLogger) Errorf(f string, a ...any) { log.Printf("ERROR: "+f, a...) }
func (stdLogger) Fatalf(f string, a ...any) { log.Fatalf("FATAL: "+f, a...) }
