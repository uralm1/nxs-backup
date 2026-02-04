package notifier

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/uralm1/nxs-backup/misc"
	"github.com/uralm1/nxs-backup/modules/logger"
)

type MessageBuffer struct {
	jobs         []string //strings: "job1, job2"
	lines        []string
	lowest_level logrus.Level //lowest level in all buffer, to determine send buffer or not
	lock         sync.Mutex
	pass_level   logrus.Level //highest message level we accept to store into buffer
}

// allocate space, clear buffer and set pass_level
// max_level - the level set in options
func (buf *MessageBuffer) Init(max_level logrus.Level) {
	buf.lock.Lock()
	defer buf.lock.Unlock()

	//reserve space for 10 lines and one job
	buf.lines = make([]string, 0, 10)
	buf.jobs = make([]string, 0, 1)

	buf.clear_nolock()

	//set pass_level
	if max_level < logrus.DebugLevel {
		buf.pass_level = logrus.InfoLevel //Info,Error,Warn etc set to Info
	} else {
		buf.pass_level = max_level //Debug,Trace set to Debug,Trace
	}
}

func (buf *MessageBuffer) Clear() {
	buf.lock.Lock()
	defer buf.lock.Unlock()
	buf.clear_nolock()
}

func (buf *MessageBuffer) clear_nolock() {
	buf.jobs = buf.jobs[:0]
	buf.lines = buf.lines[:0]
	buf.lowest_level = logrus.TraceLevel //set to maximum level here
}

func (buf *MessageBuffer) FilterAndStore(level logrus.Level, job, line string) {
	if level > buf.pass_level {
		return
	}

	buf.lock.Lock()
	defer buf.lock.Unlock()

	if level < buf.lowest_level {
		buf.lowest_level = level
	}

	if job != "" {
		if !misc.Contains(buf.jobs, job) {
			buf.jobs = append(buf.jobs, job)
		}
	}

	buf.lines = append(buf.lines, line)
}

// check the level and return comma separated jobs and buffered multiline message
// if buffer is empty or its level is not enough, return "" in message
func (buf *MessageBuffer) RetriveAndClear(max_level logrus.Level) (jobs, message string) {
	buf.lock.Lock()

	jobs = strings.Join(buf.jobs, ", ")

	if len(buf.lines) == 0 {
		message = ""
		buf.lock.Unlock()
		return
	}

	//drop buffer if its level is not enough
	if buf.lowest_level > max_level {
		message = ""
	} else {
		message = strings.Join(buf.lines, "\n")
	}

	buf.clear_nolock()
	buf.lock.Unlock()

	return
}

func CreateBodyLine(n logger.LogRecord) string {
	var sb strings.Builder
	sb.Grow(200)
	switch n.Level {
	case logrus.DebugLevel:
		sb.WriteString("DEBUG")
	case logrus.InfoLevel:
		sb.WriteString("INFO")
	case logrus.WarnLevel:
		sb.WriteString("WARNING")
	case logrus.ErrorLevel:
		sb.WriteString("ERROR")
	case logrus.PanicLevel:
		sb.WriteString("PANIC")
	case logrus.FatalLevel:
		sb.WriteString("FATAL")
	case logrus.TraceLevel:
		sb.WriteString("TRACE")
	}

	fmt.Fprintf(&sb, " [%s]", time.Now().Format("2006-01-02 15:04:05"))

	if n.JobName != "" {
		fmt.Fprintf(&sb, "[%s]", n.JobName)
	}
	if n.StorageName != "" {
		fmt.Fprintf(&sb, "(%s)", n.StorageName)
	}
	fmt.Fprintf(&sb, " %s", n.Message)
	return sb.String()
}
