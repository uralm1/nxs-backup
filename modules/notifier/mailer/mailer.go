// this file was modified as of a derivative work of nxs-backup

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
	"github.com/uralm1/nxs-backup/modules/notifier"
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
	message notifier.MessageBuffer
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

	//allocate space and clear buffer
	m.message.Init(m.opts.MessageLevel)

	return m, nil
}

func (m *mailer) ClearBuffer() {
	m.message.Clear()
}

func (m *mailer) TakeEvent(log *logrus.Logger, n logger.LogRecord) {
	m.message.FilterAndStore(n.Level, n.JobName, notifier.CreateBodyLine(n))
}

func (m *mailer) SendBuffer(log *logrus.Logger) {
	var jobs, b_msg string

	if jobs, b_msg = m.message.RetriveAndClear(m.opts.MessageLevel); b_msg == "" {
		//don't send anything if the buffer is empty or its level is not enough
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

	if jobs != "" {
		fmt.Fprintf(&subj, " %s", jobs)
		fmt.Fprintf(&body, "Job: %s\n", jobs)
	}
	fmt.Fprintf(&subj, " on %s%s", m.opts.ServerName, pn)

	body.WriteString("\n")
	body.WriteString(b_msg)

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
