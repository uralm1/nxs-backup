package logger

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type LogFormatter struct{}

func (f *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var (
		job, storage string
		s            []string
	)

	for k, v := range entry.Data {
		switch k {
		case "job":
			job = fmt.Sprintf("%s", v)
		case "storage":
			storage = fmt.Sprintf("%s", v)
		default:
			s = append(s, fmt.Sprintf("%s: %v", k, v))
		}
	}

	var out strings.Builder
	out.Grow(100)
	fmt.Fprintf(&out, "%s [%s]", strings.ToUpper(entry.Level.String()), entry.Time.Format("2006-01-02 15:04:05.000"))
	if job != "" {
		fmt.Fprintf(&out, "[%s]", job)
	}
	if storage != "" {
		fmt.Fprintf(&out, "(%s)", storage)
	}
	fmt.Fprintf(&out, " %s", entry.Message)
	if len(s) > 0 {
		fmt.Fprintf(&out, " (%s)\n", strings.Join(s, ", "))
	} else {
		out.WriteString("\n")
	}

	return []byte(out.String()), nil
}
