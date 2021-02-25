package main

import (
	"testing"
	"time"
)

func TestGetMediaInfo(t *testing.T) {
	conf, err := loadConfig()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	handler := NewFFprobe(conf.FFMpegConf.FFProbe)
	//info, err := handler.GetMediaInfo(getLocalPath(MEDIA_PATH))
	info, err := handler.GetMediaInfo(getLocalPath("data/test3.mp4"))
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
	
	step := int(duration / 17 * 1000);
	stepSec := time.Millisecond * time.Duration(step)
	current := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	
	for i := 0; i < 18; i++ {
		t.Log(i)
		if i == 17 {
			t.Log(current.Add(stepSec * time.Duration(i) / time.Millisecond / 1000 * time.Second).Format("15:04:05.000"))
		} else {
			t.Log(current.Add(stepSec * time.Duration(i)).Format("15:04:05.000"))
		}
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

	videoPath := getLocalPath("data/test3.mp4")
	imageDir := getLocalPath("temp/snap")

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
		SplitSnap(videoPath, duration, 18, imageDir)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	//os.RemoveAll(imageDir)
}

