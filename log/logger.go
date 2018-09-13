package log

import (
	"os"
	"log"
	"sync"
	"fmt"
)

// log levels
const (
	DEBUG = uint(1)
	INFO	 = uint(2)
	ERROR = uint(3)
	NONE = uint(1<<10)
)

const (
	EnvLogfile = "LOG_FILE"
	DefaultLogFile = "app.log"
)

type Logger interface {
	// log message with DEBUG log level, only if log level is DEBUG or lower
	Debug(format string, args... interface{})
	// log message with INFO log level, only if log level is INFO or lower
	Info(format string, args... interface{})
	// log message with ERROR log level, only if log level is ERROR or lower
	Error(format string, args... interface{})
}

type logger struct {
	debug	*log.Logger
	info		*log.Logger
	err		*log.Logger
	prefix string
}
// application wide log level, can be changed dynamically
var logLevel = DEBUG
func SetLogLevel(level uint) {
	logLevel = level
}

func GetLogLevel() uint {
	return logLevel
}

var logfile *os.File
var lock sync.RWMutex
func GetLogFile() *os.File {
	return getLogFile()
}
func getLogFile() *os.File {
	if logfile == nil {
		lock.Lock()
		defer lock.Unlock()
		if logfile == nil {
			// use environment variable LOGFILE, if present
			if file, err := os.Create(os.Getenv(EnvLogfile)); err == nil {
				logfile = file
			} else {
				if logfile, err = os.Create(DefaultLogFile); err != nil {
					logfile = os.Stdout
				} 
			}
		}
	}
	return logfile
}

var appLogger *logger
var loggerLock sync.RWMutex
func AppLogger() Logger {
	if appLogger == nil {
		loggerLock.Lock()
		defer loggerLock.Unlock()
		if appLogger == nil {
			appLogger = &logger {
			debug: log.New(getLogFile(), "DEBUG|", log.LstdFlags|log.Lmicroseconds),
			info: log.New(getLogFile(),  "INFO |", log.LstdFlags|log.Lmicroseconds),
			err: log.New(getLogFile(),   "ERROR|", log.LstdFlags|log.Lmicroseconds),
			prefix: "|APP| ",
			}
		}
	}
	return appLogger 
}

func NewLogger(name interface{}) Logger {
	prefix := fmt.Sprintf("|%T| ",name)
	return &logger {
		debug: log.New(getLogFile(), "DEBUG|", log.LstdFlags|log.Lmicroseconds),
		info: log.New(getLogFile(),  "INFO |", log.LstdFlags|log.Lmicroseconds),
		err: log.New(getLogFile(),   "ERROR|", log.LstdFlags|log.Lmicroseconds),
		prefix: prefix,
	}
}

func (l *logger) Debug(format string, args... interface{}) {
	if logLevel > DEBUG {
		return
	}
	l.debug.Printf(l.prefix+format, args...)
}

func (l *logger) Info(format string, args... interface{}) {
	if logLevel > INFO {
		return
	}
	l.info.Printf(l.prefix+format, args...)
}

func (l *logger) Error(format string, args... interface{}) {
	if logLevel > ERROR {
		return
	}
	l.err.Printf(l.prefix+format, args...)
}
