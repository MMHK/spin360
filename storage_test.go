package main

import "testing"

var storage IStorage

func TestNewStorage(t *testing.T) {
	conf, err := loadConfig()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	
	s3, err := NewS3Storage(conf.S3)

	if err != nil {
		t.Error(err)
		return
	}

	storage = s3
}

func TestUpload(t *testing.T) {
	path, url, err := storage.Upload(getLocalPath("./data/amore/amore.plist"), "amore.plist")

	if err != nil {
		t.Error(err)
		return
	}

	t.Log(path)
	t.Log(url)
}
