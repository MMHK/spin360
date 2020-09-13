package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func getHttpServer() (*HTTPService, error) {
	conf, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return NewHTTP(conf), nil
}

func getMultipart(parts map[string]string) (io.Reader, string, error) {
	testUploadFile := getLocalPath("./data/test.mp4")
	requestReader := new(bytes.Buffer)
	bodyWriter := multipart.NewWriter(requestReader)
	part, err := bodyWriter.CreateFormFile("video", filepath.Base(testUploadFile))
	if err != nil {
		return nil, "", err
	}
	defer bodyWriter.Close()
	
	testUpload, err := os.Open(testUploadFile)
	if err != nil {
		return nil, "", err
	}
	defer testUpload.Close()
	
	_, err = io.Copy(part, testUpload)
	if err != nil {
		return nil, "", err
	}
	
	for _name, val := range parts {
		w, err := bodyWriter.CreateFormField(_name)
		if err != nil {
			return nil, "", err
		}
		_, err = w.Write([]byte(val))
		if err != nil {
			return nil, "", err
		}
	}
	
	return requestReader, bodyWriter.FormDataContentType(), nil
}

func TestHTTPService_Split(t *testing.T) {
	httpServer, err := getHttpServer()
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	requestReader, mime, err := getMultipart(map[string]string{
		"splitSize": "36",
	})
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	req := httptest.NewRequest(http.MethodPost, "/split", requestReader)
	req.Header.Add("Content-Type", mime)
	writer := httptest.NewRecorder()

	httpServer.Split(writer, req)

	resp := writer.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Response code is %v", resp.StatusCode)
		t.Fail()
		return
	}

	saveFilePath := getLocalPath("./temp/temp.zip")
	saveFile, err := os.Create(saveFilePath)
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
	defer saveFile.Close()

	_, err = io.Copy(saveFile, resp.Body)
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	defer os.Remove(saveFilePath)
}
