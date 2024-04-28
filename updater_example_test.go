package zip_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/STARRY-S/zip"
)

func Test_Updater_Example(t *testing.T) {
	// Prepare the temp zip archive.
	// f, err := os.CreateTemp("", "test-*.zip")
	f, err := os.Create("test.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(f.Name()); err != nil {
			t.Fatal(err)
		}
	}()
	zw := zip.NewWriter(f)
	// Write one file into the temp zip archive.
	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:   "1.txt",
		Method: zip.Store,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("1.txt, hello world, abcabcabc bala bala...")); err != nil {
		t.Fatal(err)
	}
	t.Logf("write 1.txt")
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	// Update the existing zip archive file.
	zu, err := zip.NewUpdater(f)
	if err != nil {
		t.Fatal(err)
	}
	// Updater supports modify the zip comment.
	if err := zu.SetComment("Test update zip archive"); err != nil {
		t.Fatal(err)
	}

	// Append new files into existing archive.
	// The APPEND_MODE_KEEP_ORIGINAL will append file in the end of the
	// zip archive and will not replace existing files.
	w, err = zu.AppendHeader(&zip.FileHeader{
		Name:   "2.txt",
		Method: zip.Store,
	}, zip.APPEND_MODE_KEEP_ORIGINAL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("hello world 2.txt")); err != nil {
		t.Fatal(err)
	}
	t.Logf("append 2.txt")
	// Append allows to replace existing files in the zip archive.
	w, err = zu.AppendHeader(&zip.FileHeader{
		Name:   "1.txt",
		Method: zip.Store,
	}, zip.APPEND_MODE_OVERWRITE)
	if err != nil {
		t.Fatal(err)
	}
	// The replaced file size can smaller than the original file.
	if _, err := w.Write([]byte("replaced data")); err != nil {
		t.Fatal(err)
	}
	t.Logf("replaced 1.txt")
	if err := zu.Close(); err != nil {
		t.Fatal(err)
	}

	// Open the zip file for validation.
	end, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(f, end)
	if err != nil {
		t.Fatal(err)
	}
	// validate file data
	files := map[string][]byte{
		"1.txt": []byte("replaced data"),
		"2.txt": []byte("hello world 2.txt"),
	}
	for _, f := range zr.File {
		v, ok := files[f.Name]
		if !ok {
			t.Fatalf("failed: %v not expected", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(v, b) {
			t.Fatalf("file %q filaed: expected %q, actual %q", f.Name, string(b), string(v))
		}
		t.Logf("read file %q data %q, pass", f.Name, string(v))
	}
}
