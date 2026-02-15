package misc

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestGetDateTimeNowBadUnit(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("GetDateTimeNow() didn't panic-ed on bad unit!")
		}
	}()

	_ = GetDateTimeNow("bad") //should panic
}

func TestGetDateTimeNow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// 2000-01-01 00:00:00 +0000 UTC
		//t.Log(time.Now().In(time.UTC))
		time.Sleep(time.Until(time.Date(2026, 2, 6, 11, 22, 33, 0, time.Local)))
		t.Log(time.Now())

		if GetDateTimeNow("") != "2026-02-06_11-22" {
			t.Error("Empty")
		}
		if GetDateTimeNow("dom") != "6" {
			t.Error("dom")
		}
		if GetDateTimeNow("dow") != "5" {
			t.Error("dow")
		}
		if GetDateTimeNow("doy") != "37" {
			t.Error("doy")
		}
		if GetDateTimeNow("moy") != "2" {
			t.Error("moy")
		}
		if GetDateTimeNow("year") != "2026" {
			t.Error("year")
		}
		if GetDateTimeNow("previous_year") != "2025" {
			t.Error("previous_year")
		}
	})
}

func TestGetDecadeDaySubdir(t *testing.T) {
	d := []int{1, 10, 11, 20, 21, 28}
	r := []string{"day_01", "day_01", "day_11", "day_11", "day_21", "day_21"}

	for idx, day := range d {
		synctest.Test(t, func(t *testing.T) {
			// 2000-01-01 00:00:00 +0000 UTC
			//t.Log(time.Now().In(time.UTC))
			time.Sleep(time.Until(time.Date(2026, 2, day, 11, 22, 33, 0, time.Local)))
			//t.Log(time.Now())
			if GetDecadeDaySubdir() != r[idx] {
				t.Errorf("day: %v, want %s", day, r[idx])
			}
		})
	}
}
