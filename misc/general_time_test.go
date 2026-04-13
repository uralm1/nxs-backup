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

		if CurrentDOWStr() != "2" { //Tue
			t.Errorf("wrong dow %v", CurrentDOWStr())
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
