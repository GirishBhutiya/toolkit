package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Error("wrong length random string returned")
	}
}

var uploadTest = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/jpg", "image/png", "image/gif"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/jpg", "image/png", "image/gif"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg", "image/jpg", "image/gif"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTest {
		// setup a pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)

		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			// create the form data field file
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")

			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}
			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}

		}()

		// read from the pipe which receives data

		request := httptest.NewRequest("POST", "/", pr)

		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uplodedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)

		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uplodedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())

			}
			//cleanup
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uplodedFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but none received", e.name)
		}
		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// setup a pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		// create the form data field file
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")

		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}
		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}

	}()

	// read from the pipe which receives data

	request := httptest.NewRequest("POST", "/", pr)

	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uplodedFiles, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)

	if err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uplodedFiles.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())

	}
	//cleanup
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uplodedFiles.NewFileName))

}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTool Tools

	err := testTool.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	err = testTool.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	_ = os.Remove("./testdata/myDir")
}

var slugTest = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "now is the time", expected: "now-is-the-time", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "complex string", s: "Now is the time for all Good men! + fish & such &^123", expected: "now-is-the-time-for-all-good-men-fish-such-123", errorExpected: false},
	{name: "japanese string", s: "こんにちは世界", expected: "", errorExpected: true},
	{name: "japanese string and roman character", s: "hello world こんにちは世界", expected: "hello-world", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {

	var testTools Tools

	for _, e := range slugTest {
		slug, err := testTools.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error received when none expected: %s", e.name, err.Error())
		}
		if !e.errorExpected && slug != e.expected {
			t.Errorf("%s: wrong slug returned; expected %s but got %s", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTools Tools

	testTools.DownloadStaticFile(rr, req, "./testdata", "Dwij-cert.jpg", "dwij.jpg")

	res := rr.Result()

	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "58912" {
		t.Error("wrong content length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"dwij.jpg\"" {
		t.Error("wrong content disposition")
	}

	_, err := ioutil.ReadAll(res.Body)

	if err != nil {
		t.Error(err)
	}
}
