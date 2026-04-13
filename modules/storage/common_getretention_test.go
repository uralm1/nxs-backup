package storage

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestGetRetentionBadPeriod(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("GetRetention() didn't panic-ed on bad period!")
		}
	}()

	_, _ = GetRetention("bad", Retention{}) //should panic
}

func TestGetRetentionDaily(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := Retention{
			Days:     7,
			Weeks:    0,
			Months:   0,
			UseCount: false,
		}
		// 2000-01-01 00:00:00 +0000 UTC
		//t.Log(time.Now().In(time.UTC))
		time.Sleep(time.Until(time.Date(2026, 2, 6, 0, 30, 0, 0, time.Local)))
		t.Log(time.Now())

		count, date := GetRetention("daily", r)
		if date != time.Date(2026, 1, 31, 0, 0, 0, 0, time.Local) {
			t.Errorf("wrong date: %v, want 31", date)
		}
		if count != 7 {
			t.Errorf("wrong count: %v, want 7", count)
		}

		//after midday
		time.Sleep(time.Until(time.Date(2026, 2, 6, 15, 55, 0, 0, time.Local)))
		t.Log(time.Now())

		count, date = GetRetention("daily", r)
		if date != time.Date(2026, 1, 31, 0, 0, 0, 0, time.Local) {
			t.Errorf("wrong date: %v, want 31", date)
		}
		if count != 7 {
			t.Errorf("wrong count: %v, want 7", count)
		}

		//zero
		r.Days = 0
		count, date = GetRetention("daily", r)
		if !date.IsZero() {
			t.Errorf("wrong date: %v, want zero", date)
		}
		if count != 0 {
			t.Errorf("wrong count: %v, want 0", count)
		}
	})
}

func TestGetRetentionWeekly(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := Retention{
			Days:     0,
			Weeks:    2,
			Months:   0,
			UseCount: false,
		}
		// 2000-01-01 00:00:00 +0000 UTC
		//t.Log(time.Now().In(time.UTC))
		time.Sleep(time.Until(time.Date(2026, 2, 8, 0, 30, 0, 0, time.Local)))
		t.Log(time.Now())

		count, date := GetRetention("weekly", r)
		if date != time.Date(2026, 1, 26, 0, 0, 0, 0, time.Local) {
			t.Errorf("wrong date: %v, want 26", date)
		}
		if count != 2 {
			t.Errorf("wrong count: %v, want 2", count)
		}

		//not start of the week
		time.Sleep(time.Until(time.Date(2026, 2, 9, 0, 30, 0, 0, time.Local)))
		count, date = GetRetention("weekly", r)
		if !date.IsZero() {
			t.Errorf("wrong date: %v, want zero", date)
		}
		if count != 0 {
			t.Errorf("wrong count: %v, want 0", count)
		}

		//zero
		r.Weeks = 0
		count, date = GetRetention("weekly", r)
		if !date.IsZero() {
			t.Errorf("wrong date: %v, want zero", date)
		}
		if count != 0 {
			t.Errorf("wrong count: %v, want 0", count)
		}
	})
}

func TestGetRetentionMonthly(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := Retention{
			Days:     0,
			Weeks:    0,
			Months:   3,
			UseCount: false,
		}
		// 2000-01-01 00:00:00 +0000 UTC
		//t.Log(time.Now().In(time.UTC))
		time.Sleep(time.Until(time.Date(2026, 2, 1, 0, 30, 0, 0, time.Local)))
		t.Log(time.Now())

		count, date := GetRetention("monthly", r)
		if date != time.Date(2025, 11, 2, 0, 0, 0, 0, time.Local) {
			t.Errorf("wrong date: %v", date)
		}
		if count != 3 {
			t.Errorf("wrong count: %v, want 3", count)
		}

		//not start of the month
		time.Sleep(time.Until(time.Date(2026, 2, 2, 0, 30, 0, 0, time.Local)))
		count, date = GetRetention("monthly", r)
		if !date.IsZero() {
			t.Errorf("wrong date: %v, want zero", date)
		}
		if count != 0 {
			t.Errorf("wrong count: %v, want 0", count)
		}

		//zero
		r.Months = 0
		count, date = GetRetention("monthly", r)
		if !date.IsZero() {
			t.Errorf("wrong date: %v, want zero", date)
		}
		if count != 0 {
			t.Errorf("wrong count: %v, want 0", count)
		}
	})
}
