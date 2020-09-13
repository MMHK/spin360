package main

import (
	"testing"
)

func TestGetMediaInfo(t *testing.T) {
	handler := NewFFprobe(FFPROBE_BIN)
	//info, err := handler.GetMediaInfo(getLocalPath(MEDIA_PATH))
	info, err := handler.GetMediaInfo(getLocalPath("data/test.mp4"))
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(info)

	duration, err := info.GetDuration()
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(duration)
}

