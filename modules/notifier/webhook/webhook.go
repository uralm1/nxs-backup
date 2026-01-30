package webhook

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/uralm1/nxs-backup/modules/logger"
)

// Opts contains webhook options
type Opts struct {
	WebhookURL        string
	InsecureTLS       bool
	PayloadMessageKey string
	ExtraPayload      map[string]any
	ExtraHeaders      map[string]string
	MessageLevel      logrus.Level
	ProjectName       string
	ServerName        string
}

type webhook struct {
	opts      Opts
	client    *http.Client
	a_message logger.LogRecord
}

func (wh *webhook) SupportPostponedNotification() bool {
	return false
}

func Init(opts Opts) (*webhook, error) {

	wh := &webhook{
		opts: opts,
	}

	_, err := url.Parse(opts.WebhookURL)
	if err != nil {
		return wh, err
	}

	d := &net.Dialer{
		Timeout: 5 * time.Second,
	}
	wh.client = &http.Client{
		Transport: &http.Transport{
			DialContext: d.DialContext,
			//ResponseHeaderTimeout: 60 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.InsecureTLS,
			},
		},
	}

	return wh, nil
}

func (wh *webhook) ClearBuffer() {
}

func (wh *webhook) TakeEvent(log *logrus.Logger, n logger.LogRecord) {
	if n.Level > wh.opts.MessageLevel {
		return
	}
	wh.a_message = n
}

func (wh *webhook) SendBuffer(log *logrus.Logger) {
	req, err := http.NewRequest(http.MethodPost, wh.opts.WebhookURL, bytes.NewBuffer(wh.getJsonData(log, wh.a_message)))
	if err != nil {
		log.Errorf("Can't create webhook request: %v", err)
		return
	}
	req.Header.Add("Content-Type", "application/json")

	for k, v := range wh.opts.ExtraHeaders {
		if k == "Content-Type" {
			continue
		}
		if k == "Host" {
			req.Host = v
		}
		req.Header.Add(k, v)
	}

	resp, err := wh.client.Do(req)
	if err != nil {
		log.Errorf("Request error: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	log.Tracef("HTTP response code: %d, body: %v", resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		log.Errorf("Unexpected HTTP response code: %d, body: %v", resp.StatusCode, string(body))
	}
}

// createMessage generates notification message from event log record
func (wh *webhook) createMessage(n logger.LogRecord) string {
	var sb strings.Builder
	sb.Grow(255)
	switch n.Level {
	case logrus.DebugLevel:
		sb.WriteString("[DEBUG]")
	case logrus.InfoLevel:
		sb.WriteString("[INFO]")
	case logrus.WarnLevel:
		sb.WriteString("[WARNING]")
	case logrus.ErrorLevel:
		sb.WriteString("[ERROR]")
	case logrus.PanicLevel:
	case logrus.FatalLevel:
	case logrus.TraceLevel:
	}

	if sb.Len() > 0 {
		sb.WriteString("\n\n")
	}

	if wh.opts.ProjectName != "" {
		fmt.Fprintf(&sb, "Project: %s\n", wh.opts.ProjectName)
	}
	if wh.opts.ServerName != "" {
		fmt.Fprintf(&sb, "Server: %s\n\n", wh.opts.ServerName)
	}

	if n.JobName != "" {
		fmt.Fprintf(&sb, "Job: %s\n", n.JobName)
	}
	if n.StorageName != "" {
		fmt.Fprintf(&sb, "Storage: %s\n", n.StorageName)
	}
	fmt.Fprintf(&sb, "\nMessage: %s\n", n.Message)

	return sb.String()
}

func (wh *webhook) getJsonData(log *logrus.Logger, n logger.LogRecord) []byte {
	data := make(map[string]any)

	data[wh.opts.PayloadMessageKey] = wh.createMessage(n)
	for k, v := range wh.opts.ExtraPayload {
		data[k] = v
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Can't marshal json for webhook request: %v", err)
		return nil
	}

	return jsonData
}
