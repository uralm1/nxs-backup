package mailer

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"

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
		job     string
		storage string
		lines   []string
	}
}

func (m *mailer) SupportPostponedNotification() bool {
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
	//reserve space for 10 lines
	m.message.lines = make([]string, 0, 10)

	return m, nil
}

func (m *mailer) ClearBuffer() {
	m.message.job = ""
	m.message.storage = ""
	m.message.lines = m.message.lines[:0]
}

func (m *mailer) TakeEvent(log *logrus.Logger, n logger.LogRecord) {
	if n.Level > m.opts.MessageLevel {
		return
	}
	m.message.job = n.JobName
	m.message.storage = n.StorageName
	m.message.lines = append(m.message.lines, m.createMailBodyLine(n))
}

func (m *mailer) SendBuffer(log *logrus.Logger) {
	var (
		sc  gomail.SendCloser
		err error
	)
	defer func() { _ = sc.Close() }()

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.opts.From)
	msg.SetHeader("To", m.opts.Recipients...)

	var subj strings.Builder
	subj.Grow(100)
	var body strings.Builder
	body.Grow(255)

	var pn string
	subj.WriteString("Nxs-backup")
	fmt.Fprintf(&body, "Server %q\n", m.opts.ServerName)
	if m.opts.ProjectName != "" {
		pn = fmt.Sprintf(" (%q)", m.opts.ProjectName)
		fmt.Fprintf(&body, "Project %q\n", m.opts.ProjectName)
	}

	if m.message.job != "" {
		fmt.Fprintf(&subj, " %s", m.message.job)
		fmt.Fprintf(&body, "Job: %s\n", m.message.job)
	}
	fmt.Fprintf(&subj, " on %q%s", m.opts.ServerName, pn)
	msg.SetHeader("Subject", subj.String())

	if m.message.storage != "" {
		fmt.Fprintf(&body, "Storage: %s\n", m.message.storage)
	}
	body.WriteString("\n")
	body.WriteString(strings.Join(m.message.lines, "\n"))
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
	fmt.Fprintf(&sb, ": %s", n.Message)
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
