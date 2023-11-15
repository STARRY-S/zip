package zip_test

import (
	"io"
	"os"
	"testing"

	"github.com/STARRY-S/zip"
)

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func ls(t *testing.T, dir []zip.Directory) {
	for _, d := range dir {
		t.Logf("%s offset(%v) %s",
			d.Mode().String(), d.HeaderOffset(), d.Name)
	}
}

func append(u *zip.Updater, name string) {
	w, err := u.Append(name)
	handleErr(err)
	f, err := os.Open(name)
	handleErr(err)
	defer f.Close()
	_, err = io.Copy(w, f)
	handleErr(err)
}

func Test_Updater(t *testing.T) {
	// Create a new zip file.
	f, err := os.Create("test.zip")
	handleErr(err)
	zw := zip.NewWriter(f)
	w, err := zw.Create("LICENSE")
	handleErr(err)
	t.Log("create test.zip")
	t.Log("---------------------")

	// Write one file into zip archive.
	t1, err := os.Open("LICENSE")
	handleErr(err)
	_, err = io.Copy(w, t1)
	handleErr(err)
	t.Log("write 'LICENSE' file into test.zip")
	// Write comment
	err = zw.SetComment("Test create ZIP Archive")
	handleErr(err)
	t.Log("---------------------")
	// Finished create zip file.
	t1.Close()
	zw.Close()
	f.Close()

	// Reopen test.zip archive with read/write only mode for Updater.
	f, err = os.OpenFile("test.zip", os.O_RDWR, 0)
	handleErr(err)
	zu, err := zip.NewUpdater(f)
	handleErr(err)

	// Modift the zip comment
	err = zu.SetComment("Test update zip archive")
	handleErr(err)

	// Show current files in archive index.
	dir := zu.Directory()
	ls(t, dir)

	// Append one new file into existing zip archive.
	append(zu, "struct.go")
	t.Log("append struct.go into test.zip")
	t.Log("---------------------")

	// Close Updater and file
	zu.Close()
	f.Close()

	// Re-open test zip archive.
	f, err = os.OpenFile("test.zip", os.O_RDWR, 0)
	handleErr(err)
	zu, err = zip.NewUpdater(f)
	handleErr(err)

	dir = zu.Directory()
	ls(t, dir)

	// Append multiple new files into existing archive file.
	for _, n := range []string{
		"reader.go",
		"register.go",
		".gitignore",
	} {
		append(zu, n)
		t.Logf("append %q into test.zip", n)
	}
	t.Log("---------------------")
	dir = zu.Directory()
	ls(t, dir)

	// Updater supports to rewrite existing files into zip archive.
	var lastHeaderOffset int64
	for _, d := range dir {
		if lastHeaderOffset < d.HeaderOffset() {
			lastHeaderOffset = d.HeaderOffset()
		}
	}
	// Overwrite the last file (.gitignore) to 'writer.go'.
	// The size of writer.go is larger than .gitignore .
	w, err = zu.AppendAt("writer.go", lastHeaderOffset)
	handleErr(err)
	f, err = os.Open("writer.go")
	handleErr(err)
	_, err = io.Copy(w, f)
	handleErr(err)
	f.Close()

	t.Log("replaced .gitignore file to writer.go in test.zip file")
	t.Log("---------------------")
	dir = zu.Directory()
	ls(t, dir)

	zu.Close()

	// Finally, re-open the zip archive by Reader to validate.
	zr, err := zip.OpenReader("test.zip")
	handleErr(err)

	t.Logf("modified zip archive comment: %v", zr.Comment)
	t.Logf("modified zip archive directory:")
	buf := make([]byte, 10)
	for _, f := range zr.File {
		t.Logf("%s %s", f.Mode(), f.Name)
		rc, err := f.Open()
		handleErr(err)
		_, err = rc.Read(buf)
		handleErr(err)
		t.Logf("content (pre 30 Bytes): %v...n", string(buf))
	}
	zr.Close()

	// Clean-up test zip file.
	os.Remove("test.zip")
}
