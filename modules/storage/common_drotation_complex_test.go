package storage

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"
)

func TestDRotationFilesComplexDaily(t *testing.T) {
	rf := RotateFiles{
		{"file30.tar", time.Date(2026, 1, 30, 0, 30, 0, 0, time.Local)},
		{"file31.tar", time.Date(2026, 1, 31, 0, 30, 0, 0, time.Local)},
		{"file1.tar", time.Date(2026, 2, 1, 0, 30, 0, 0, time.Local)},
		{"file2.tar", time.Date(2026, 2, 2, 0, 30, 0, 0, time.Local)},
		{"file3.tar", time.Date(2026, 2, 3, 0, 30, 0, 0, time.Local)},
		{"file4.tar", time.Date(2026, 2, 4, 0, 30, 0, 0, time.Local)},
		{"file5.tar", time.Date(2026, 2, 5, 0, 30, 0, 0, time.Local)},
	}

	synctest.Test(t, func(t *testing.T) {
		// 2000-01-01 00:00:00 +0000 UTC
		//t.Log(time.Now().In(time.UTC))

		//today is 2026-02-05
		time.Sleep(time.Until(time.Date(2026, 2, 5, 10, 30, 0, 0, time.Local)))
		t.Log(time.Now())

		//Days:2, not safe (5 deleted, 2 remain)
		count, date := GetRetention("daily", Retention{2, 0, 0, false})
		r := DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{"file30.tar", "file31.tar", "file1.tar", "file2.tar", "file3.tar"}) {
			t.Errorf("got: %v", r)
		}

		//Days:6, not safe (1 deleted, 6 remain)
		count, date = GetRetention("daily", Retention{6, 0, 0, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{"file30.tar"}) {
			t.Errorf("got: %v", r)
		}

		//Days:7, not safe (0 deleted, 7 remain)
		count, date = GetRetention("daily", Retention{7, 0, 0, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{}) {
			t.Errorf("got: %v", r)
		}

		//Days:1, not safe (6 deleted, 1 remain)
		count, date = GetRetention("daily", Retention{1, 0, 0, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{"file30.tar", "file31.tar", "file1.tar", "file2.tar", "file3.tar", "file4.tar"}) {
			t.Errorf("got: %v", r)
		}

		//Days:0, not safe (do nothing, is this logically correct?)
		count, date = GetRetention("daily", Retention{0, 0, 0, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{}) {
			t.Errorf("got: %v", r)
		}
	})

	synctest.Test(t, func(t *testing.T) {
		//today is 2026-02-06
		time.Sleep(time.Until(time.Date(2026, 2, 6, 10, 30, 0, 0, time.Local)))
		t.Log(time.Now())

		//Days:2, not safe (6 deleted, 1 remain, 1 will be created)
		count, date := GetRetention("daily", Retention{2, 0, 0, false})
		r := DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{"file30.tar", "file31.tar", "file1.tar", "file2.tar", "file3.tar", "file4.tar"}) {
			t.Errorf("got: %v", r)
		}

		//Days:1, not safe (7 deleted, 0 remain, 1 will be created)
		count, date = GetRetention("daily", Retention{1, 0, 0, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{"file30.tar", "file31.tar", "file1.tar", "file2.tar", "file3.tar", "file4.tar", "file5.tar"}) {
			t.Errorf("got: %v", r)
		}
	})
}

func TestDRotationFilesComplexWeekly(t *testing.T) {
	rf := RotateFiles{
		{"file18.tar", time.Date(2026, 1, 18, 0, 30, 0, 0, time.Local)},
		{"file25.tar", time.Date(2026, 1, 25, 0, 30, 0, 0, time.Local)},
		{"file1_.tar", time.Date(2026, 2, 1, 0, 30, 0, 0, time.Local)}, //
		{"file1.tar", time.Date(2026, 2, 1, 0, 30, 0, 0, time.Local)},
		{"file8.tar", time.Date(2026, 2, 8, 0, 30, 0, 0, time.Local)},
		{"file9.tar", time.Date(2026, 2, 9, 0, 30, 0, 0, time.Local)}, //
		{"file15.tar", time.Date(2026, 2, 15, 0, 30, 0, 0, time.Local)},
		{"file20.tar", time.Date(2026, 2, 20, 0, 30, 0, 0, time.Local)}, //
		{"file22.tar", time.Date(2026, 2, 22, 0, 30, 0, 0, time.Local)},
	}

	synctest.Test(t, func(t *testing.T) {
		//today is 2026-02-23
		time.Sleep(time.Until(time.Date(2026, 2, 23, 10, 30, 0, 0, time.Local)))
		t.Log(time.Now())

		//Weeks:2, not safe (do nothing, 2026-02-23 is not Sun)
		count, date := GetRetention("weekly", Retention{0, 2, 0, false})
		r := DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{}) {
			t.Errorf("got: %v", r)
		}
	})

	synctest.Test(t, func(t *testing.T) {
		//today is 2026-03-01
		time.Sleep(time.Until(time.Date(2026, 3, 1, 10, 30, 0, 0, time.Local)))
		t.Log(time.Now())

		//Weeks:2, not safe (7 deleted, 1 remain + 1 not Sun, 1 will be created)
		count, date := GetRetention("weekly", Retention{0, 2, 0, false})
		r := DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{"file18.tar", "file25.tar", "file1_.tar", "file1.tar", "file8.tar", "file9.tar", "file15.tar"}) {
			t.Errorf("got: %v", r)
		}

		//Weeks:1, not safe (all deleted, 1 will be created)
		count, date = GetRetention("weekly", Retention{0, 1, 0, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{"file18.tar", "file25.tar", "file1_.tar", "file1.tar", "file8.tar", "file9.tar", "file15.tar", "file20.tar", "file22.tar"}) {
			t.Errorf("got: %v", r)
		}

		//Weeks:6, not safe (1 deleted, 1 will be created)
		count, date = GetRetention("weekly", Retention{0, 6, 0, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{"file18.tar"}) {
			t.Errorf("got: %v", r)
		}

		//Weeks:7, not safe (0 deleted, 1 will be created)
		count, date = GetRetention("weekly", Retention{0, 7, 0, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{}) {
			t.Errorf("got: %v", r)
		}

		//Weeks:0, not safe (do nothing, is this logically correct?)
		count, date = GetRetention("weekly", Retention{0, 0, 0, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{}) {
			t.Errorf("got: %v", r)
		}
	})
}

func TestDRotationFilesComplexMonthly(t *testing.T) {
	rf := RotateFiles{
		{"file9.tar", time.Date(2025, 9, 1, 0, 30, 0, 0, time.Local)},
		{"file10.tar", time.Date(2025, 10, 1, 0, 30, 0, 0, time.Local)},
		{"file11.tar", time.Date(2025, 11, 1, 0, 30, 0, 0, time.Local)},
		{"file12.tar", time.Date(2025, 12, 1, 0, 30, 0, 0, time.Local)},
		{"file1.tar", time.Date(2026, 1, 1, 0, 30, 0, 0, time.Local)},
		{"file2.tar", time.Date(2026, 2, 1, 0, 30, 0, 0, time.Local)},
	}

	synctest.Test(t, func(t *testing.T) {
		//today is 2026-02-23
		time.Sleep(time.Until(time.Date(2026, 2, 23, 10, 30, 0, 0, time.Local)))
		t.Log(time.Now())

		//Months:2, not safe (do nothing, 2026-02-23 is not 1-st of month)
		count, date := GetRetention("monthly", Retention{0, 0, 2, false})
		r := DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{}) {
			t.Errorf("got: %v", r)
		}
	})

	synctest.Test(t, func(t *testing.T) {
		//today is 2026-03-01
		time.Sleep(time.Until(time.Date(2026, 3, 1, 10, 30, 0, 0, time.Local)))
		t.Log(time.Now())

		//Months:2, not safe (5 deleted, 1 remain, 1 will be created)
		count, date := GetRetention("monthly", Retention{0, 0, 2, false})
		r := DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{"file9.tar", "file10.tar", "file11.tar", "file12.tar", "file1.tar"}) {
			t.Errorf("got: %v", r)
		}

		//Months:1, not safe (all deleted, 1 will be created)
		count, date = GetRetention("monthly", Retention{0, 0, 1, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{"file9.tar", "file10.tar", "file11.tar", "file12.tar", "file1.tar", "file2.tar"}) {
			t.Errorf("got: %v", r)
		}

		//Months:6, not safe (1 deleted, 1 will be created)
		count, date = GetRetention("monthly", Retention{0, 0, 6, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{"file9.tar"}) {
			t.Errorf("got: %v", r)
		}

		//Months:7, not safe (0 deleted, 1 will be created)
		count, date = GetRetention("monthly", Retention{0, 0, 7, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{}) {
			t.Errorf("got: %v", r)
		}

		//Months:0, not safe (do nothing, is this logically correct?)
		count, date = GetRetention("monthly", Retention{0, 0, 0, false})
		r = DRotationFiles(rf, date, count, false, false)
		if !slices.Equal(r, []string{}) {
			t.Errorf("got: %v", r)
		}
	})
}
