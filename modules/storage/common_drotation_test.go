package storage

import (
	"slices"
	"testing"
	"time"
)

func TestDRotationFilesFilter(t *testing.T) {
	mt := time.Date(2026, 2, 8, 0, 30, 0, 0, time.Local)
	rf := RotateFiles{
		{"file1.zzz", mt},
		{"file2.tar", mt},
		{"file3.tar.gz", mt},
		{".", mt},
		{"..", mt},
	}

	//zero count and use_count means delete all files except filtered
	r := DRotationFiles(rf, mt, 0, true, true)
	if !slices.Equal(r, []string{"file2.tar", "file3.tar.gz"}) {
		t.Errorf("got: %v", r)
	}
}

func TestDRotationFilesUseCount(t *testing.T) {
	zerodate := time.Date(1, 1, 1, 0, 0, 0, 0, time.Local)
	rf := RotateFiles{
		{"file1.tar", time.Date(2026, 2, 8, 4, 30, 0, 0, time.Local)},
		{"file2.tar", time.Date(2026, 2, 8, 2, 30, 0, 0, time.Local)},
		{"file3.tar.gz", time.Date(2026, 2, 8, 3, 30, 0, 0, time.Local)},
		{"file4.tar", time.Date(2026, 2, 8, 5, 30, 0, 0, time.Local)},
		{"file5.tar", time.Date(2026, 2, 8, 1, 30, 0, 0, time.Local)},
		{"..", time.Date(2026, 2, 8, 10, 30, 0, 0, time.Local)},
	}

	//3, not safe (3 deleted, 1 created = 3)
	r := DRotationFiles(rf, zerodate, 3, true, false)
	if !slices.Equal(r, []string{"file5.tar", "file2.tar", "file3.tar.gz"}) {
		t.Errorf("got: %v", r)
	}

	//1, not safe (all deleted, 1 created = 1)
	r = DRotationFiles(rf, zerodate, 1, true, false)
	if !slices.Equal(r, []string{"file5.tar", "file2.tar", "file3.tar.gz", "file1.tar", "file4.tar"}) {
		t.Errorf("got: %v", r)
	}

	//1, safe (1 created, 4 deleted)
	r = DRotationFiles(rf, zerodate, 1, true, true)
	if !slices.Equal(r, []string{"file5.tar", "file2.tar", "file3.tar.gz", "file1.tar"}) {
		t.Errorf("got: %v", r)
	}

	//5, not safe (1 deleted, 1 created = 5)
	r = DRotationFiles(rf, zerodate, 5, true, false)
	if !slices.Equal(r, []string{"file5.tar"}) {
		t.Errorf("got: %v", r)
	}

	//5, safe (1 created, 0 deleted)
	r = DRotationFiles(rf, zerodate, 5, true, true)
	if !slices.Equal(r, []string{}) {
		t.Errorf("got: %v", r)
	}

	//6, not safe (0 deleted, 1 created = 6)
	r = DRotationFiles(rf, zerodate, 6, true, false)
	if !slices.Equal(r, []string{}) {
		t.Errorf("got: %v", r)
	}

	//7, not safe (0 deleted, 1 created = 6)
	r = DRotationFiles(rf, zerodate, 7, true, false)
	if !slices.Equal(r, []string{}) {
		t.Errorf("got: %v", r)
	}
}

func TestDRotationFilesDate(t *testing.T) {
	rf := RotateFiles{
		{"file1.tar", time.Date(2026, 2, 2, 0, 30, 0, 0, time.Local)},
		{"file2.tar", time.Date(2026, 2, 5, 0, 30, 0, 0, time.Local)},
		{"file3.tar", time.Date(2026, 2, 3, 0, 30, 0, 0, time.Local)},
		{"file4.tar", time.Date(2026, 2, 4, 0, 30, 0, 0, time.Local)},
		{"file5.tar", time.Date(2026, 2, 1, 0, 30, 0, 0, time.Local)},
	}

	//2026-02-03, not safe (2 deleted, 1 created)
	r := DRotationFiles(rf, time.Date(2026, 2, 3, 0, 0, 0, 0, time.Local), 0, false, false)
	if !slices.Equal(r, []string{"file1.tar", "file5.tar"}) {
		t.Errorf("got: %v", r)
	}

	//2026-02-06, not safe (all deleted, 1 created)
	r = DRotationFiles(rf, time.Date(2026, 2, 6, 0, 0, 0, 0, time.Local), 0, false, false)
	if !slices.Equal(r, []string{"file1.tar", "file2.tar", "file3.tar", "file4.tar", "file5.tar"}) {
		t.Errorf("got: %v", r)
	}

	//2026-02-01, safe (1 created, 0 deleted)
	r = DRotationFiles(rf, time.Date(2026, 2, 1, 0, 0, 0, 0, time.Local), 0, false, true)
	if !slices.Equal(r, []string{}) {
		t.Errorf("got: %v", r)
	}

	//2026-02-02, not safe (1 deleted, 1 created)
	r = DRotationFiles(rf, time.Date(2026, 2, 2, 0, 0, 0, 0, time.Local), 0, false, false)
	if !slices.Equal(r, []string{"file5.tar"}) {
		t.Errorf("got: %v", r)
	}
}
