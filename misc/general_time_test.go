package misc

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestCurrentFuncStr(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(time.Until(time.Date(2024, 2, 6, 11, 22, 33, 0, time.Local)))

		if CurrentDateTimeFmt() != "2024-02-06_11-22" {
			t.Errorf("Wrong format %v", CurrentDateTimeFmt())
		}

		if CurrentDayStr() != "6" {
			t.Error("wrong day")
		}

		if CurrentDOYStr() != "37" {
			t.Errorf("wrong doy %v", CurrentDOYStr())
		}

		if CurrentMonthStr() != "2" {
			t.Error("wrong month")
		}

		if CurrentDOWStr() != "2" { //Tue
			t.Errorf("wrong dow %v", CurrentDOWStr())
		}

		if CurrentYearStr() != "2024" {
			t.Error("wrong year")
		}
	})
}

func TestBeginningOfThisDay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(time.Until(time.Date(2026, 2, 6, 11, 22, 33, 0, time.Local)))
		if BeginningOfThisDay() != time.Date(2026, 2, 6, 0, 0, 0, 0, time.Local) {
			t.Error("wrong date 1")
		}

		time.Sleep(time.Until(time.Date(2026, 2, 7, 18, 33, 44, 0, time.Local)))
		if BeginningOfThisDay() != time.Date(2026, 2, 7, 0, 0, 0, 0, time.Local) {
			t.Error("wrong date 2")
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
