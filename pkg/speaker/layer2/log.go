package layer2

import (
	"regexp"

	"k8s.io/klog/v2"
)

const (
	logRegexpLevel = `\[(?P<level>[A-Z]+)\]`
	logRegexpMsg   = `(?P<msg>.*)`
)

var (
	logRegexp = regexp.MustCompile(logRegexpLevel + logRegexpMsg)
)

func subexps(line []byte) map[string]string {
	m := logRegexp.FindSubmatch(line)
	if len(m) == 0 {
		return map[string]string{}
	}

	result := map[string]string{}
	for i, name := range logRegexp.SubexpNames() {
		if i > 0 && i <= len(m) {
			result[name] = string(m[i])
		}
	}
	return result
}

type logWriter struct{}

func (l logWriter) Write(p []byte) (n int, err error) {
	result := subexps(p)
	keyvals := []interface{}{}
	if msg, ok := result["msg"]; ok {
		keyvals = append(keyvals, msg)
	}

	level := result["level"]
	if level == "" {
		level = "INFO"
	}

	logWithLevel(level, keyvals)
	return len(p), nil
}

func logWithLevel(lvl string, keyvals []interface{}) {
	switch lvl {
	case "DEBUG":
		klog.V(2).Info(keyvals...)
	case "INFO":
		klog.Info(keyvals...)
	case "WARN":
		klog.Warning(keyvals...)
	case "ERR", "ERROR":
		klog.Error(keyvals...)
	default:
		klog.Info(keyvals...)
	}
}
