package ctx

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/uralm1/nxs-backup/modules/notifier/mailer"
	"github.com/uralm1/nxs-backup/modules/notifier/webhooker"

	"github.com/uralm1/nxs-backup/interfaces"
)

var messageLevels = map[string]logrus.Level{
	"ERR":     logrus.ErrorLevel,
	"ERROR":   logrus.ErrorLevel,
	"WARN":    logrus.WarnLevel,
	"WARNING": logrus.WarnLevel,
	"INF":     logrus.InfoLevel,
	"INFO":    logrus.InfoLevel,
}

func notifiersInit(c *Ctx, conf ConfOpts) error {
	var errs []error
	var ns []interfaces.Notifier

	if conf.Notifications.Mail.Enabled {
		var mailErrs []error
		mailList := conf.Notifications.Mail.Recipients
		for _, a := range mailList {
			_, err := mail.ParseAddress(a)
			if err != nil {
				mailErrs = append(mailErrs, fmt.Errorf("Email init fail. Failed to parse email \"%s\". %v ", a, err))
			}
		}
		if _, err := mail.ParseAddress(conf.Notifications.Mail.From); err != nil {
			mailErrs = append(mailErrs, fmt.Errorf("Email init fail. Failed to parse `mail_from` \"%s\". %v ", conf.Notifications.Mail.From, err))
		}

		ml, ok := messageLevels[strings.ToUpper(conf.Notifications.Mail.MessageLevel)]
		if ok {
			if len(mailErrs) > 0 {
				errs = append(errs, mailErrs...)
			} else {
				m, err := mailer.Init(mailer.Opts{
					From:         conf.Notifications.Mail.From,
					SmtpServer:   conf.Notifications.Mail.SmtpServer,
					SmtpPort:     conf.Notifications.Mail.SmtpPort,
					SmtpUser:     conf.Notifications.Mail.SmtpUser,
					SmtpPassword: conf.Notifications.Mail.SmtpPassword,
					Recipients:   conf.Notifications.Mail.Recipients,
					MessageLevel: ml,
					ProjectName:  conf.ProjectName,
					ServerName:   conf.ServerName,
				})
				if err != nil {
					errs = append(errs, err)
				} else {
					ns = append(ns, m)
				}
			}
		} else {
			errs = append(errs, fmt.Errorf("Email init fail. Unknown message level. Available levels: 'INFO', 'WARN', 'ERR' "))
		}
	}

	for _, wh := range conf.Notifications.Webhooks {
		if wh.Enabled {
			ml, ok := messageLevels[strings.ToUpper(wh.MessageLevel)]
			if ok {
				a, err := webhooker.Init(webhooker.Opts{
					WebhookURL:        wh.WebhookURL,
					InsecureTLS:       wh.InsecureTLS,
					ExtraHeaders:      wh.ExtraHeaders,
					PayloadMessageKey: wh.PayloadMessageKey,
					ExtraPayload:      wh.ExtraPayload,
					MessageLevel:      ml,
					ProjectName:       conf.ProjectName,
					ServerName:        conf.ServerName,
				})
				if err != nil {
					errs = append(errs, err)
				} else {
					ns = append(ns, a)
				}
			} else {
				errs = append(errs, fmt.Errorf("Webhook init fail. Unknown message level. Available levels: 'INFO', 'WARN', 'ERR' "))
			}
		}
	}

	c.Notifiers = ns

	return errors.Join(errs...)
}
