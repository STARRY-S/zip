package zip

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	// For testing zip64 large files, needs at least 10G disk spaces
	TEMP_DIR = "/var/tmp"
)

var overwriteTestsOriginal = []WriteTest{
	{
		Name:   "foo",
		Data:   []byte("short data"), // overwrite the short file to large file
		Method: Store,
		Mode:   0666,
	},
	{
		Name: "foo2",
		// overwrite large file to short file
		Data: []byte("long-data--------------------------------------------------" +
			"--------------------------------------------------------------------" +
			"--------------------------------------------------------------------" +
			"--------------------------------------------------------------------" +
			"--------------------------------------------------------------------"),
		Method: Store,
		Mode:   0666,
	},
	{
		Name:   "bar",
		Data:   nil, // large data set in the test
		Method: Deflate,
		Mode:   0644,
	},
	{
		Name:   "setuid",
		Data:   []byte("setuid file"),
		Method: Deflate,
		Mode:   0755 | fs.ModeSetuid,
	},
	{
		Name:   "setgid",
		Data:   []byte("setgid file"),
		Method: Deflate,
		Mode:   0755 | fs.ModeSetgid,
	},
	{
		Name:   "symlink",
		Data:   []byte("../link/target"),
		Method: Deflate,
		Mode:   0755 | fs.ModeSymlink,
	},
	{
		Name:   "device",
		Data:   []byte("device file"),
		Method: Deflate,
		Mode:   0755 | fs.ModeDevice,
	},
	{
		Name:   "chardevice",
		Data:   []byte("char device file"),
		Method: Deflate,
		Mode:   0755 | fs.ModeDevice | fs.ModeCharDevice,
	},
}

var overwriteTestsReplaced = []WriteTest{
	{
		Name:   "foo",
		Data:   []byte("replaced-long-data"),
		Method: Store,
		Mode:   0666,
	},
	{
		Name:   "foo2",
		Data:   []byte("replaced-short-data"),
		Method: Store,
		Mode:   0666,
	},
	{
		Name:   "bar",
		Data:   nil, // large data set in the test
		Method: Deflate,
		Mode:   0644,
	},
	{
		Name:   "setuid",
		Data:   []byte("setuid file"),
		Method: Deflate,
		Mode:   0755 | fs.ModeSetuid,
	},
	{
		Name:   "setgid",
		Data:   []byte("setgid file"),
		Method: Deflate,
		Mode:   0755 | fs.ModeSetgid,
	},
	{
		Name:   "symlink",
		Data:   []byte("../link/target"),
		Method: Deflate,
		Mode:   0755 | fs.ModeSymlink,
	},
	{
		Name:   "device",
		Data:   []byte("device file"),
		Method: Deflate,
		Mode:   0755 | fs.ModeDevice,
	},
	{
		Name:   "chardevice",
		Data:   []byte("char device file"),
		Method: Deflate,
		Mode:   0755 | fs.ModeDevice | fs.ModeCharDevice,
	},
}

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
		testAppend(t, u, &wt, APPEND_MODE_KEEP_ORIGINAL)
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

func TestUpdaterOverwrite(t *testing.T) {
	// init zip file
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
	for _, wt := range overwriteTestsOriginal {
		// write files when create zip
		testCreate(t, w, &wt)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	u, err := NewUpdater(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, wt := range overwriteTestsReplaced {
		// replace (overwrite) files when create zip
		testAppend(t, u, &wt, APPEND_MODE_OVERWRITE)
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
	for i, wt := range overwriteTestsReplaced {
		testReadFile(t, r.File[i], &wt)
	}
}

func testAppend(t *testing.T, u *Updater, ut *WriteTest, mode AppendMode) {
	header := &FileHeader{
		Name:   ut.Name,
		Method: ut.Method,
	}
	if ut.Mode != 0 {
		header.SetMode(ut.Mode)
	}
	f, err := u.AppendHeader(header, mode)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Write(ut.Data)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdaterOverwriteZip64(t *testing.T) {
	if os.Getenv("SKIP_ZIP64") != "" {
		t.Skip("Skip test for zip64 large file")
	}
	var tmpFiles = []string{}
	var sha256sums = []string{}

	// Create 5G zip archive
	for i := 0; i < 5; i++ {
		f, err := os.CreateTemp(TEMP_DIR, fmt.Sprintf("test-%d-*.iso", i))
		if err != nil {
			t.Error(err)
			return
		}
		t.Logf("create temp file %v", f.Name())
		// Delete file after test
		defer os.Remove(f.Name())

		h := sha256.New()
		buff := make([]byte, 1<<20) //1M
		for i := range buff {
			buff[i] = byte(rand.Int32())
		}
		for i := 0; i < 1<<10; i++ { // 1024 * 1M = 1G
			n, err := f.Write(buff)
			if err != nil {
				t.Error(err)
				return
			}
			_, err = h.Write(buff[:n])
			if err != nil {
				t.Error(err)
				return
			}
		}

		tmpFiles = append(tmpFiles, f.Name())
		s := fmt.Sprintf("%x", h.Sum(nil))
		t.Logf("sha256sum of %q is %q", f.Name(), s)
		sha256sums = append(sha256sums, s)
		f.Close()
	}

	f, err := os.CreateTemp(TEMP_DIR, "test-append-*.zip")
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("create zip file %v", f.Name())
	defer f.Close()
	// Delete file after test
	defer os.Remove(f.Name())

	// Create zip file and write some large files
	w := NewWriter(f)
	for _, n := range tmpFiles {
		f, err := os.Open(n)
		if err != nil {
			t.Error(err)
			return
		}
		w, err := w.CreateHeader(&FileHeader{
			Name:   f.Name(),
			Method: Store,
		})
		if err != nil {
			t.Error(err)
			return
		}
		t.Logf("writing %v to zip", f.Name())
		_, err = io.Copy(w, f)
		if err != nil {
			t.Error(err)
			return
		}
		f.Close()
	}
	if err := w.Close(); err != nil {
		t.Error(err)
		return
	}

	u, err := NewUpdater(f)
	if err != nil {
		t.Error(err)
		return
	}
	for _, n := range tmpFiles {
		t.Logf("appending %v to zip", n)
		f, err := os.Open(n)
		if err != nil {
			t.Error(err)
			return
		}
		start := time.Now()
		w, err := u.AppendHeader(&FileHeader{
			Name:   f.Name(),
			Method: Store,
		}, APPEND_MODE_OVERWRITE)
		if err != nil {
			t.Error(err)
			return
		}
		end := time.Now()
		t.Logf("rewind data of file %q spent: %v", n, end.Sub(start))
		start = time.Now()
		if _, err := io.Copy(w, f); err != nil {
			t.Error(err)
			return
		}
		end = time.Now()
		t.Logf("write data of file %q spent: %v", n, end.Sub(start))
		f.Close()
	}
	u.Close()

	// The replaced files in zip archive should same with the original one
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		t.Error(err)
		return
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Error(err)
		return
	}
	r, err := NewReader(f, size)
	if err != nil {
		t.Error(err)
		return
	}
	for i, file := range r.File {
		rc, err := file.Open()
		if err != nil {
			t.Error(err)
			return
		}
		var buffer = make([]byte, bufferSize)
		h := sha256.New()
		for {
			n, err := rc.Read(buffer)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				t.Error(err)
				return
			}
			h.Write(buffer[:n])
		}
		s := fmt.Sprintf("%x", h.Sum(nil))
		if sha256sums[i] != s {
			t.Errorf("sha256sum mismatch for file %q: expected %q != actual %q",
				file.Name, sha256sums[i], s)
			return
		}
		t.Logf("file %q sha256sum %q pass", file.Name, s)
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
