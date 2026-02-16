package storage

import (
	"slices"
	"testing"
	"time"
)

func TestDRotationObjectsFilter(t *testing.T) {
	mt := time.Date(2026, 2, 8, 0, 30, 0, 0, time.Local)
	rf := RotationObjects{
		{"file1.zzz", mt},
		{"file2.tar", mt},
		{"file3.tar.gz", mt},
		{".", mt},
		{"..", mt},
	}

	//zero count and use_count means delete all files except filtered
	r := DGetRotatedObjects(rf, 0, mt, true, true)
	if !slices.Equal(r, []string{"file2.tar", "file3.tar.gz"}) {
		t.Errorf("got: %v", r)
	}
}

func TestDRotationObjectsUseCount(t *testing.T) {
	zerodate := time.Date(1, 1, 1, 0, 0, 0, 0, time.Local)
	rf := NewRotationObjects(6)
	rf.AddObject("file1.tar", time.Date(2026, 2, 8, 4, 30, 0, 0, time.Local))
	rf.AddObject("file2.tar", time.Date(2026, 2, 8, 2, 30, 0, 0, time.Local))
	rf.AddObject("file3.tar.gz", time.Date(2026, 2, 8, 3, 30, 0, 0, time.Local))
	rf.AddObject("file4.tar", time.Date(2026, 2, 8, 5, 30, 0, 0, time.Local))
	rf.AddObject("file5.tar", time.Date(2026, 2, 8, 1, 30, 0, 0, time.Local))
	rf.AddObject("..", time.Date(2026, 2, 8, 10, 30, 0, 0, time.Local))

	//3, not safe (3 deleted, 1 created = 3)
	r := DGetRotatedObjects(rf, 3, zerodate, true, false)
	if !slices.Equal(r, []string{"file5.tar", "file2.tar", "file3.tar.gz"}) {
		t.Errorf("got: %v", r)
	}

	//1, not safe (all deleted, 1 created = 1)
	r = DGetRotatedObjects(rf, 1, zerodate, true, false)
	if !slices.Equal(r, []string{"file5.tar", "file2.tar", "file3.tar.gz", "file1.tar", "file4.tar"}) {
		t.Errorf("got: %v", r)
	}

	//1, safe (1 created, 4 deleted)
	r = DGetRotatedObjects(rf, 1, zerodate, true, true)
	if !slices.Equal(r, []string{"file5.tar", "file2.tar", "file3.tar.gz", "file1.tar"}) {
		t.Errorf("got: %v", r)
	}

	//5, not safe (1 deleted, 1 created = 5)
	r = DGetRotatedObjects(rf, 5, zerodate, true, false)
	if !slices.Equal(r, []string{"file5.tar"}) {
		t.Errorf("got: %v", r)
	}

	//5, safe (1 created, 0 deleted)
	r = DGetRotatedObjects(rf, 5, zerodate, true, true)
	if !slices.Equal(r, []string{}) {
		t.Errorf("got: %v", r)
	}

	//6, not safe (0 deleted, 1 created = 6)
	r = DGetRotatedObjects(rf, 6, zerodate, true, false)
	if !slices.Equal(r, []string{}) {
		t.Errorf("got: %v", r)
	}

	//7, not safe (0 deleted, 1 created = 6)
	r = DGetRotatedObjects(rf, 7, zerodate, true, false)
	if !slices.Equal(r, []string{}) {
		t.Errorf("got: %v", r)
	}
}

func TestDRotationObjectsDate(t *testing.T) {
	rf := RotationObjects{
		{"file1.tar", time.Date(2026, 2, 2, 0, 30, 0, 0, time.Local)},
		{"file2.tar", time.Date(2026, 2, 5, 0, 30, 0, 0, time.Local)},
		{"file3.tar", time.Date(2026, 2, 3, 0, 30, 0, 0, time.Local)},
		{"file4.tar", time.Date(2026, 2, 4, 0, 30, 0, 0, time.Local)},
		{"file5.tar", time.Date(2026, 2, 1, 0, 30, 0, 0, time.Local)},
	}

	//2026-02-03, not safe (2 deleted, 1 created)
	r := DGetRotatedObjects(rf, 0, time.Date(2026, 2, 3, 0, 0, 0, 0, time.Local), false, false)
	if !slices.Equal(r, []string{"file1.tar", "file5.tar"}) {
		t.Errorf("got: %v", r)
	}

	//2026-02-06, not safe (all deleted, 1 created)
	r = DGetRotatedObjects(rf, 0, time.Date(2026, 2, 6, 0, 0, 0, 0, time.Local), false, false)
	if !slices.Equal(r, []string{"file1.tar", "file2.tar", "file3.tar", "file4.tar", "file5.tar"}) {
		t.Errorf("got: %v", r)
	}

	//2026-02-01, safe (1 created, 0 deleted)
	r = DGetRotatedObjects(rf, 0, time.Date(2026, 2, 1, 0, 0, 0, 0, time.Local), false, true)
	if !slices.Equal(r, []string{}) {
		t.Errorf("got: %v", r)
	}

	//2026-02-02, not safe (1 deleted, 1 created)
	r = DGetRotatedObjects(rf, 0, time.Date(2026, 2, 2, 0, 0, 0, 0, time.Local), false, false)
	if !slices.Equal(r, []string{"file5.tar"}) {
		t.Errorf("got: %v", r)
	}
}

func TestDRotationObjectsDateDifferentLocations(t *testing.T) {
	rf := RotationObjects{
		{"file1.tar", time.Date(2026, 2, 3, 10, 29, 0, 0, time.Local)},
		{"file1_utc.tar", time.Date(2026, 2, 3, 10, 29, 0, 0, time.Local).In(time.UTC)},
		{"file2.tar", time.Date(2026, 2, 3, 10, 31, 0, 0, time.Local)},
		{"file2_utc.tar", time.Date(2026, 2, 3, 10, 31, 0, 0, time.Local).In(time.UTC)},
	}

	if rf[1].modtime.Location() != time.UTC || rf[3].modtime.Location() != time.UTC {
		t.Error("should be in utc tz")
	}

	//2026-02-03 10:30, not safe (2 deleted)
	r := DGetRotatedObjects(rf, 0, time.Date(2026, 2, 3, 10, 30, 0, 0, time.Local), false, false)
	if !slices.Equal(r, []string{"file1.tar", "file1_utc.tar"}) {
		t.Errorf("got: %v", r)
	}
}
