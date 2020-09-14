package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestWorker_Split(t *testing.T) {
	conf, err := loadConfig()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	videoFile, err := os.Open(getLocalPath("./data/test.mp4"))
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	defer videoFile.Close()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute * 30))


	worker := NewWorker(conf)
	_, err = worker.Split(ctx, videoFile, 64)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	cancel()

	<-time.After(time.Second * 3)
}

func TestWorker_S3(t *testing.T) {
	conf, err := loadConfig()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	
	videoFile, err := os.Open(getLocalPath("./data/test.mp4"))
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	defer videoFile.Close()
	
	worker := NewWorker(conf)
	list, err := worker.S3(videoFile, 32)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	
	t.Log(len(list))
	t.Log(list)
}

func TestWorker_SavePlayConfig(t *testing.T) {
	conf, err := loadConfig()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	worker := NewWorker(conf)

	url ,err := worker.SavePlayConfig(&Spin360Config{
		Pages: []*SpinPage{ &SpinPage{ ImageURL:"#" }, },
		HotSpot: []*PageHotSpot{
			&PageHotSpot{
				Type: HOTSPOT_TYPE_TEXT,
				Text: "Hello",
				Coordinates: []*HotSpotCoordinates{
					&HotSpotCoordinates{
						X: "10%",
						Y: "10%",
						PageIndex: 0,
					},
				},
			},
		},
	})
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	t.Log(url)
}

func TestWorker_GetConfig(t *testing.T) {
	conf, err := loadConfig()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	worker := NewWorker(conf)

	spin360Config, err := worker.GetConfig("225c2811-c13b-4e3b-81c5-3dbb53075bd2")
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	t.Logf("%+v", spin360Config)
}