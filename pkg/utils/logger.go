package utils

import (
	"fmt"
	glog "log"
	gsyslog "log/syslog"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/syslog"
)

var mu sync.Mutex
var loggers = make(map[string]*LogHandle)

var syslogHook *syslog.SyslogHook

func InitLoggers(logToSyslog bool) {
	if logToSyslog {
		var err error
		syslogHook, err = syslog.NewSyslogHook("", "", gsyslog.LOG_DEBUG, "")
		if err != nil {
			// we are the child process and we cannot connect to syslog,
			// probably because we are in a container without syslog
			// nothing much we can do here, printing to stderr doesn't work
			return
		}

		for _, l := range loggers {
			l.Hooks.Add(syslogHook)
		}
	}
}

type LogHandle struct {
	logrus.Logger

	name string
	Lvl  *logrus.Level
}

func (l *LogHandle) Format(e *logrus.Entry) ([]byte, error) {
	// Mon Jan 2 15:04:05 -0700 MST 2006
	timestamp := ""
	lvl := e.Level
	if l.Lvl != nil {
		lvl = *l.Lvl
	}

	if syslogHook == nil {
		const timeFormat = "2006/01/02 15:04:05.000000"

		timestamp = e.Time.Format(timeFormat) + " "
	}

	str := fmt.Sprintf("%v%v.%v %v",
		timestamp,
		l.name,
		strings.ToUpper(lvl.String()), e.Message)

	if len(e.Data) != 0 {
		str += " " + fmt.Sprint(e.Data)
	}

	str += "\n"
	return []byte(str), nil
}

// for aws.Logger
func (l *LogHandle) Log(args ...interface{}) {
	l.Debugln(args...)
}

func NewLogger(name string) *LogHandle {
	l := &LogHandle{name: name}
	l.Out = os.Stdout
	l.Formatter = l
	l.Level = logrus.InfoLevel
	l.Hooks = make(logrus.LevelHooks)
	if syslogHook != nil {
		l.Hooks.Add(syslogHook)
	}
	return l
}

func GetLogger(name string) *LogHandle {
	mu.Lock()
	defer mu.Unlock()

	if logger, ok := loggers[name]; ok {
		return logger
	}

	logger := NewLogger(name)
	loggers[name] = logger
	return logger
}

func GetStdLogger(l *LogHandle, lvl logrus.Level) *glog.Logger {
	return glog.New(l.WriterLevel(lvl), "", 0)
}
