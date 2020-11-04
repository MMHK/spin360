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

	t.Logf("%v", info)

	duration, err := info.GetFormat().GetDuration()
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("resolution: %d x %d", info.GetStream().Width, info.GetStream().Height)

	t.Logf("duration: %f", duration)
}

func TestFFmpeg_SplitSnap(t *testing.T) {
	conf, err := loadConfig()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	videoPath := getLocalPath("data/test2.mp4")
	imageDir := getLocalPath("temp/snap");

	handler := NewFFprobe(conf.FFMpegConf.FFProbe)

	info, err := handler.GetMediaInfo(videoPath)
	if err != nil {
		t.Error(err)
		return
	}

	duration, err := info.GetFormat().GetDuration()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	outputHeight := info.GetStream().Height
	if outputHeight > 720 {
		outputHeight = 720
	}

	ffmpeg := NewFFmpeg(conf.FFMpegConf.FFmpeg)
	err = ffmpeg.SetOutputHeight(outputHeight).
		SplitSnap(videoPath, duration, 16, imageDir)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	//os.RemoveAll(imageDir)
}

