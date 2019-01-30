package proxy

import (
	"github.com/sirupsen/logrus"
	"path"
	"runtime"
)


type ContextHook struct {
}
func (hook ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
func (hook ContextHook) Fire(entry *logrus.Entry) error {
	if pc, file, line, ok := runtime.Caller(8); ok {
		funcName := runtime.FuncForPC(pc).Name()
		entry.Data["file"] = path.Base(file)
		entry.Data["func"] = path.Base(funcName)
		entry.Data["line"] = line
	}
	return nil
}


var log = logrus.New()

func init(){
	//log.SetReportCaller(true)
	log.AddHook(ContextHook{})
	log.Level = logrus.TraceLevel
}

