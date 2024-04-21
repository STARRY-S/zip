package zip

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestUpdater(t *testing.T) {
	// init a empty zip file
	f, err := os.CreateTemp("", "test-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("create test zip file %q", f.Name())
	defer func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
		if err := os.Remove(f.Name()); err != nil {
			t.Error(err)
		}
		t.Logf("delete %q", f.Name())
	}()
	w := NewWriter(f)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	u, err := NewUpdater(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, wt := range writeTests {
		testAppend(t, u, &wt)
	}
	if err := u.Close(); err != nil {
		t.Fatal(err)
	}

	// read it back
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(f, size)
	if err != nil {
		t.Fatal(err)
	}
	for i, wt := range writeTests {
		testReadFile(t, r.File[i], &wt)
	}
}

// TestUpdateComment is test for EOCD comment read/write.
func TestUpdateComment(t *testing.T) {
	var tests = []struct {
		comment string
		ok      bool
	}{
		{"hi, hello", true},
		{"hi, こんにちわ", true},
		{strings.Repeat("a", uint16max), true},
		{strings.Repeat("a", uint16max+1), false},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			// init a empty zip file
			f, err := os.CreateTemp("", "test-*.zip")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := f.Close(); err != nil {
					t.Error(err)
				}
				if err := os.Remove(f.Name()); err != nil {
					t.Error(err)
				}
			}()
			w := NewWriter(f)
			if err := w.SetComment(test.comment); err != nil {
				if test.ok {
					t.Fatalf("SetComment: unexpected error %v", err)
				}
				return
			} else {
				if !test.ok {
					t.Fatalf("SetComment: unexpected success, want error")
				}
			}

			if err := w.Close(); test.ok == (err != nil) {
				t.Fatal(err)
			}

			if w.closed != test.ok {
				t.Fatalf("Writer.closed: got %v, want %v", w.closed, test.ok)
			}

			// skip read test in failure cases
			if !test.ok {
				return
			}

			// read it back
			size, err := f.Seek(0, io.SeekEnd)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = f.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			r, err := NewReader(f, size)
			if err != nil {
				t.Fatal(err)
			}
			if r.Comment != test.comment {
				t.Fatalf("Reader.Comment: got %v, want %v", r.Comment, test.comment)
			}
		})
	}
}

func testAppend(t *testing.T, u *Updater, ut *WriteTest) {
	header := &FileHeader{
		Name:   ut.Name,
		Method: ut.Method,
	}
	if ut.Mode != 0 {
		header.SetMode(ut.Mode)
	}
	f, err := u.AppendHeaderAt(header, -1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Write(ut.Data)
	if err != nil {
		t.Fatal(err)
	}
}
