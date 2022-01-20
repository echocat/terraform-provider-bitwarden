package plugin

import (
	"fmt"
	log "github.com/echocat/slf4g"
	"github.com/echocat/slf4g/level"
	sdk "github.com/echocat/slf4g/sdk/bridge"
	"github.com/hashicorp/go-hclog"
	"io"
	log2 "log"
)

type Logger struct {
	log.Logger
}

func (this *Logger) logger(args ...interface{}) log.Logger {
	l := len(args)
	if (l % 2) != 0 {
		panic(fmt.Sprintf("Illegal amount of arguments: %d", l))
	}
	result := this.Logger
	for i := 0; i < l; i += 2 {
		result = result.With(args[i].(string), args[i+1])
	}
	return result
}

func (this *Logger) Trace(msg string, args ...interface{}) {
	this.logger(args...).Trace(msg)
}

func (this *Logger) Debug(msg string, args ...interface{}) {
	this.logger(args...).Debug(msg)
}

func (this *Logger) Info(msg string, args ...interface{}) {
	this.logger(args...).Info(msg)
}

func (this *Logger) Warn(msg string, args ...interface{}) {
	this.logger(args...).Warn(msg)
}

func (this *Logger) Error(msg string, args ...interface{}) {
	this.logger(args...).Error(msg)
}

func (this *Logger) IsTrace() bool {
	return this.Logger.IsTraceEnabled()
}

func (this *Logger) IsDebug() bool {
	return this.Logger.IsDebugEnabled()
}

func (this *Logger) IsInfo() bool {
	return this.Logger.IsInfoEnabled()
}

func (this *Logger) IsWarn() bool {
	return this.Logger.IsWarnEnabled()
}

func (this *Logger) IsError() bool {
	return this.Logger.IsErrorEnabled()
}

func (this *Logger) With(args ...interface{}) hclog.Logger {
	return &Logger{this.logger(args...)}
}

func (this *Logger) Named(string) hclog.Logger {
	panic("not implemented")
}

func (this *Logger) ResetNamed(string) hclog.Logger {
	panic("not implemented")
}

func (this *Logger) SetLevel(hclog.Level) {
	panic("not implemented")
}

func (this *Logger) StandardLogger(*hclog.StandardLoggerOptions) *log2.Logger {
	return sdk.NewWrapper(this.Logger, level.Info)
}

func (this *Logger) StandardWriter(*hclog.StandardLoggerOptions) io.Writer {
	panic("not implemented")
}
