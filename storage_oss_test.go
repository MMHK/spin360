package main

import (
	"bytes"
	"net/http"
	"testing"
	"time"
)

func getOSSStorage() (*OSSStorage, error) {
	conf, err := loadConfig()
	if err != nil {
		return nil, err
	}

	return NewOSSStorage(conf.OSS)
}

func Test_GenerateFormMultipart(t *testing.T) {
	oss, err := getOSSStorage()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	params, err := oss.GenerateFormMultipart("HelloWorld.png", 10*time.Minute, "http://127.0.0.1/")
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	t.Log(params)
}

func Test_VerifyWebUploadCallback(t *testing.T) {
	url := "http://demo2.mixmedia.com/temp/oss/index.php"
	body := bytes.NewBuffer([]byte(`{"mimeType":"image/png", "size":1081151, "filename","ipa/demo/4c7ccb26-8495-4f46-be10-df2b3296b7f6.png", "bucket":"oss-mixmedia-com"}`))
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	req.Header.Add("x-oss-pub-key-url", "aHR0cHM6Ly9nb3NzcHVibGljLmFsaWNkbi5jb20vY2FsbGJhY2tfcHViX2tleV92MS5wZW0=")
	req.Header.Add("authorization", "MEV0a4eDF4hqB2W1b7Xm9fj8N9NmGHmgKThpMJqFOf+qFoLJsy5AolGPYL/nQqvNhDQQ3MV74/VJT9p0iOg6ng==")

	result, err := VerifyWebUploadCallback(req)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	t.Log(result)
}

func TestOSSUpload(t *testing.T) {
	storage, err := getOSSStorage()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	
	path, url, err := storage.Upload(getLocalPath("./testdata/amore/amore.plist"), "amore.plist")
	
	if err != nil {
		t.Error(err)
		return
	}
	
	t.Log(path)
	t.Log(url)
}