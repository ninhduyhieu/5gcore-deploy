package logctx

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	DEFAULT_LOG_LEVEL logrus.Level = logrus.InfoLevel
)

type Fields map[string]interface{}

var _logger *loggerWrapper

type loggerWrapper struct {
	*logrus.Entry
	instance *logrus.Logger
}

func init() {
	instance := logrus.New()
	instance.SetLevel(logrus.Level(DEFAULT_LOG_LEVEL))
	_logger = &loggerWrapper{
		instance: instance,
		Entry:    instance.WithField("logger", "generic"), //default root
	}
}

// initialize logger, set fields for application root writer
func Init(f *os.File, level string, fields logrus.Fields) (err error) {
	var logLevel logrus.Level
	if len(level) > 0 {
		if logLevel, err = logrus.ParseLevel(level); err != nil {
			return fmt.Errorf("Failed to parse log level: %+v", err)
		}
	} else {
		logLevel = DEFAULT_LOG_LEVEL
	}
	_logger.instance.SetLevel(logLevel)
	if f != nil {
		_logger.instance.SetOutput(f)
	}
	_logger.Entry = _logger.instance.WithFields(logrus.Fields(fields)) //updated root
	return
}

func Entry(fields logrus.Fields) *logrus.Entry {
	if len(fields) == 0 {
		return _logger.Entry
	}
	return _logger.WithFields(fields)
}

// new writer inheriting fields from the application root logger
func WithFields(fields Fields) LogWriter {
	if len(fields) == 0 {
		return &writer{
			Entry: _logger.Entry,
		}
	}
	return &writer{
		Entry: _logger.WithFields(logrus.Fields(fields)),
	}
}

type writer struct {
	*logrus.Entry
}

// add fields to writer
func (w *writer) WithFields(fields Fields) LogWriter {
	return &writer{
		Entry: w.Entry.WithFields(logrus.Fields(fields)),
	}
}

type LogWriter interface {
	Trace(args ...interface{})
	Debug(args ...interface{})
	Print(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Warning(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Panic(args ...interface{})

	Tracef(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Printf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})

	Traceln(args ...interface{})
	Debugln(args ...interface{})
	Println(args ...interface{})
	Infoln(args ...interface{})
	Warnln(args ...interface{})
	Warningln(args ...interface{})
	Errorln(args ...interface{})
	Fatalln(args ...interface{})
	Panicln(args ...interface{})

	WithFields(Fields) LogWriter
}
