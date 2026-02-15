package notifier

import (
	"testing"

	"github.com/sirupsen/logrus"
)

//logrus.TraceLevel 6
//logrus.DebugLevel 5
//logrus.InfoLevel  4
//logrus.WarnLevel  3
//logrus.ErrorLevel 2
//logrus.FatalLevel 1
//logrus.PanicLevel 0

func TestMessageBuffer(t *testing.T) {
	buf := MessageBuffer{}
	buf.Init(logrus.InfoLevel)
	buf.FilterAndStore(logrus.InfoLevel, "job1", "line1")
	buf.FilterAndStore(logrus.InfoLevel, "job2", "line2")
	jobs, m := buf.RetriveAndClear(logrus.InfoLevel)
	if jobs != "job1, job2" {
		t.Errorf("got: %s", jobs)
	}
	if m != "line1\nline2" {
		t.Errorf("got: %s", m)
	}

	jobs, m = buf.RetriveAndClear(logrus.InfoLevel)
	if jobs != "" {
		t.Errorf("got: %s", jobs)
	}
	if m != "" {
		t.Errorf("got: %s", m)
	}

	buf.FilterAndStore(logrus.InfoLevel, "job1", "line1")
	buf.Clear()
	buf.FilterAndStore(logrus.InfoLevel, "job2", "line2")
	buf.FilterAndStore(logrus.DebugLevel, "job3", "line3")
	jobs, m = buf.RetriveAndClear(logrus.InfoLevel)
	if jobs != "job2" {
		t.Errorf("got: %s", jobs)
	}
	if m != "line2" {
		t.Errorf("got: %s", m)
	}

	//buffer has error => retrive all
	buf.FilterAndStore(logrus.TraceLevel, "1", "trace")
	buf.FilterAndStore(logrus.InfoLevel, "2", "info")
	buf.FilterAndStore(logrus.WarnLevel, "3", "warn")
	buf.FilterAndStore(logrus.ErrorLevel, "4", "error")
	buf.FilterAndStore(logrus.FatalLevel, "5", "fatal")
	jobs, m = buf.RetriveAndClear(logrus.ErrorLevel)
	if jobs != "2, 3, 4, 5" {
		t.Errorf("got: %s", jobs)
	}
	if m != "info\nwarn\nerror\nfatal" {
		t.Errorf("got: %s", m)
	}

	//buffer has no errors => retrive nothing
	buf.FilterAndStore(logrus.InfoLevel, "2", "info")
	buf.FilterAndStore(logrus.WarnLevel, "3", "warn")
	jobs, m = buf.RetriveAndClear(logrus.ErrorLevel)
	if jobs != "2, 3" {
		t.Errorf("got: %s", jobs)
	}
	if m != "" {
		t.Errorf("got: %s", m)
	}

	//buffer has fatal > error => retrive all
	buf.FilterAndStore(logrus.InfoLevel, "2", "info")
	buf.FilterAndStore(logrus.WarnLevel, "3", "warn")
	buf.FilterAndStore(logrus.FatalLevel, "5", "fatal")
	jobs, m = buf.RetriveAndClear(logrus.ErrorLevel)
	if jobs != "2, 3, 5" {
		t.Errorf("got: %s", jobs)
	}
	if m != "info\nwarn\nfatal" {
		t.Errorf("got: %s", m)
	}
}
