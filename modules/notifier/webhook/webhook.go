// this file was modified as of a derivative work of nxs-backup

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
	"github.com/uralm1/nxs-backup/modules/notifier"
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
	opts    Opts
	client  *http.Client
	message notifier.MessageBuffer
}

func (h *webhook) CanCombineMessages() bool {
	return true
}

func Init(opts Opts) (*webhook, error) {
	h := &webhook{
		opts: opts,
	}

	_, err := url.Parse(opts.WebhookURL)
	if err != nil {
		return h, err
	}

	d := &net.Dialer{
		Timeout: 5 * time.Second,
	}
	h.client = &http.Client{
		Transport: &http.Transport{
			DialContext: d.DialContext,
			//ResponseHeaderTimeout: 60 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.InsecureTLS,
			},
		},
	}

	//allocate space and clear buffer
	h.message.Init(h.opts.MessageLevel)

	return h, nil
}

func (h *webhook) ClearBuffer() {
	h.message.Clear()
}

func (h *webhook) TakeEvent(log *logrus.Logger, n logger.LogRecord) {
	h.message.FilterAndStore(n.Level, n.JobName, notifier.CreateBodyLine(n))
}

func (h *webhook) SendBuffer(log *logrus.Logger) {
	var jobs, b_msg string

	if jobs, b_msg = h.message.RetriveAndClear(h.opts.MessageLevel); b_msg == "" {
		//don't send anything if the buffer is empty or its level is not enough
		return
	}

	var m strings.Builder
	m.Grow(255)

	fmt.Fprintf(&m, "Nxs-backup [%s]\n", time.Now().Format("2006-01-02 15:04:05"))

	fmt.Fprintf(&m, "Server: %s\n", h.opts.ServerName)
	if h.opts.ProjectName != "" {
		fmt.Fprintf(&m, "Project: %s\n", h.opts.ProjectName)
	}
	if jobs != "" {
		fmt.Fprintf(&m, "Job: %s\n", jobs)
	}
	m.WriteString("\n")
	m.WriteString(b_msg)

	req, err := http.NewRequest(http.MethodPost, h.opts.WebhookURL, bytes.NewBuffer(h.getJsonData(log, m.String())))
	if err != nil {
		log.Errorf("Can't create webhook request: %v", err)
		return
	}
	req.Header.Add("Content-Type", "application/json")

	for k, v := range h.opts.ExtraHeaders {
		if k == "Content-Type" {
			continue
		}
		if k == "Host" {
			req.Host = v
		}
		req.Header.Add(k, v)
	}

	resp, err := h.client.Do(req)
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

func (h *webhook) getJsonData(log *logrus.Logger, message string) []byte {
	data := make(map[string]any)

	data[h.opts.PayloadMessageKey] = message
	for k, v := range h.opts.ExtraPayload {
		data[k] = v
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Can't marshal json for webhook request: %v", err)
		return nil
	}

	return jsonData
}
