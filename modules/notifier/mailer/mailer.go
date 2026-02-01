// this file was modified as of a derivative work of nxs-backup

package mailer

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"

	"github.com/uralm1/nxs-backup/misc"
	"github.com/uralm1/nxs-backup/modules/logger"
)

type Opts struct {
	From         string
	SmtpServer   string
	SmtpPort     int
	SmtpUser     string
	SmtpPassword string
	SmtpTimeout  string
	Recipients   []string
	MessageLevel logrus.Level
	ProjectName  string
	ServerName   string
}

type mailer struct {
	opts    Opts
	message struct {
		jobs         []string //strings: "job1, job2"
		lines        []string
		lowest_level logrus.Level //lowest level in all buffer, to determine send buffer or not
		lock         sync.Mutex
	}
	pass_level logrus.Level //highest level we accept into buffer
}

func (m *mailer) CanCombineMessages() bool {
	return true
}

func Init(mailCfg Opts) (*mailer, error) {
	m := &mailer{opts: mailCfg}

	if mailCfg.SmtpServer != "" {
		d := gomail.NewDialer(mailCfg.SmtpServer, mailCfg.SmtpPort, mailCfg.SmtpUser, mailCfg.SmtpPassword)
		sc, err := d.Dial()
		if err != nil {
			return m, fmt.Errorf("Failed to dial SMTP server. Error: %v ", err)
		}
		defer func() { _ = sc.Close() }()
	}
	//reserve space for 10 lines and one job
	m.message.lines = make([]string, 0, 10)
	m.message.jobs = make([]string, 0, 1)

	if m.opts.MessageLevel < logrus.DebugLevel {
		m.pass_level = logrus.InfoLevel //Info,Error,Warn etc set to Info
	} else {
		m.pass_level = m.opts.MessageLevel //Debug,Trace set to Debug,Trace
	}
	m.message.lowest_level = logrus.TraceLevel //set to maximum level here

	return m, nil
}

func (m *mailer) ClearBuffer() {
	m.message.lock.Lock()
	defer m.message.lock.Unlock()
	m.clearBuffer_nolock()
}

func (m *mailer) clearBuffer_nolock() {
	m.message.jobs = m.message.jobs[:0]
	m.message.lines = m.message.lines[:0]
	m.message.lowest_level = logrus.TraceLevel
}

func (m *mailer) TakeEvent(log *logrus.Logger, n logger.LogRecord) {
	if n.Level > m.pass_level {
		return
	}

	m.message.lock.Lock()
	defer m.message.lock.Unlock()

	if n.Level < m.message.lowest_level {
		m.message.lowest_level = n.Level
	}

	if n.JobName != "" {
		if !misc.Contains(m.message.jobs, n.JobName) {
			m.message.jobs = append(m.message.jobs, n.JobName)
		}
	}

	m.message.lines = append(m.message.lines, m.createMailBodyLine(n))
}

func (m *mailer) SendBuffer(log *logrus.Logger) {
	m.message.lock.Lock()

	//don't send anything if the buffer is empty
	if len(m.message.lines) == 0 {
		m.message.lock.Unlock()
		return
	}
	//drop buffer if its level is not enough
	if m.message.lowest_level > m.opts.MessageLevel {
		m.clearBuffer_nolock()
		m.message.lock.Unlock()
		return
	}

	var subj strings.Builder
	subj.Grow(100)
	var body strings.Builder
	body.Grow(255)

	var pn string
	subj.WriteString("Nxs-backup")
	fmt.Fprintf(&body, "Server: %s\n", m.opts.ServerName)
	if m.opts.ProjectName != "" {
		pn = fmt.Sprintf(" (%s)", m.opts.ProjectName)
		fmt.Fprintf(&body, "Project: %s\n", m.opts.ProjectName)
	}

	if len(m.message.jobs) > 0 {
		fmt.Fprintf(&subj, " %s", strings.Join(m.message.jobs, ", "))
		fmt.Fprintf(&body, "Job: %s\n", strings.Join(m.message.jobs, ", "))
	}
	fmt.Fprintf(&subj, " on %s%s", m.opts.ServerName, pn)

	body.WriteString("\n")
	body.WriteString(strings.Join(m.message.lines, "\n"))

	m.clearBuffer_nolock()
	m.message.lock.Unlock()

	var sc gomail.SendCloser
	var err error

	defer func() { _ = sc.Close() }()

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.opts.From)
	msg.SetHeader("To", m.opts.Recipients...)
	msg.SetHeader("Subject", subj.String())
	msg.SetBody("text/plain", body.String())

	if m.opts.SmtpServer != "" {
		d := gomail.NewDialer(m.opts.SmtpServer, m.opts.SmtpPort, m.opts.SmtpUser, m.opts.SmtpPassword)
		sc, err = d.Dial()
		if err != nil {
			log.Errorf("Failed to dial SMTP server. Error: %v", err)
			return
		}
	} else {
		sc = localMail{}
	}

	if err = gomail.Send(sc, msg); err != nil {
		log.Errorf("Could not send email: %v", err)
	}
}

func (m *mailer) createMailBodyLine(n logger.LogRecord) string {
	var sb strings.Builder
	sb.Grow(200)
	switch n.Level {
	case logrus.DebugLevel:
		sb.WriteString("[DEBUG]")
	case logrus.InfoLevel:
		sb.WriteString("[INFO]")
	case logrus.WarnLevel:
		sb.WriteString("[WARNING]")
	case logrus.ErrorLevel:
		sb.WriteString("[ERROR]")
	}

	if n.JobName != "" {
		fmt.Fprintf(&sb, "[%s]", n.JobName)
	}
	if n.StorageName != "" {
		fmt.Fprintf(&sb, "(%s)", n.StorageName)
	}
	fmt.Fprintf(&sb, " %s", n.Message)
	return sb.String()
}

type localMail struct {
}

func (l localMail) Send(_ string, _ []string, msg io.WriterTo) error {
	buf := bytes.Buffer{}
	_, _ = msg.WriteTo(&buf)
	cmd := exec.Command("/usr/sbin/sendmail", "-t", "-oi")
	cmd.Stdin = &buf
	return cmd.Run()
}

func (l localMail) Close() error {
	return nil
}
