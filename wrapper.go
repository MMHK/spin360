package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type CommandBuilder struct {
	Bin    string
	cmd    *exec.Cmd
	Params []string
}


func NewBuilder(binPath string) *CommandBuilder {
	return &CommandBuilder{
		Bin:    binPath,
		Params: []string{},
	}
}

func (this *CommandBuilder) SetParams(args ...string) *CommandBuilder {
	this.Params = append(this.Params, args...)
	return this
}

func (this *CommandBuilder) Run() (io.Reader, error) {
	var outPipe bytes.Buffer
	var errorPipe bytes.Buffer

	this.cmd = exec.Command(this.Bin, this.Params...)
	this.cmd.Stdout = &outPipe
	this.cmd.Stderr = &errorPipe

	log.Debug(this.cmd)

	err := this.cmd.Run()
	if err != nil {
		log.Error(err)
		return nil, errors.New(errorPipe.String())
	}

	return bytes.NewReader(outPipe.Bytes()), nil
}

func (this *CommandBuilder) Start() (chan io.Reader, error) {
	var outPipe bytes.Buffer
	var errorPipe bytes.Buffer

	this.cmd = exec.Command(this.Bin, this.Params...)
	this.cmd.Stdout = &outPipe
	this.cmd.Stderr = &errorPipe

	log.Debug(this.cmd)

	err := this.cmd.Start()
	if err != nil {
		log.Error(err)
		log.Error(errorPipe.String())
		return nil, err
	}

	done := make(chan io.Reader, 0)
	go func() {
		err := this.cmd.Wait()
		if err != nil {
			log.Error(err)
			done <- strings.NewReader(err.Error())
			return
		}
		done <- bytes.NewReader(outPipe.Bytes())
	}()

	return done, nil
}

func (this *CommandBuilder) Process() *os.Process {
	return this.cmd.Process
}

func (this *CommandBuilder) Stop() error {
	if this != nil && this.cmd != nil && this.cmd.Process != nil {
		return this.cmd.Process.Kill()
	}
	return nil
}

type FFprobe struct {
	bin string
}

type StreamInfo struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (this *StreamInfo) GetResolution() (int, int) {
	return this.Width, this.Height
}

type CommandResult struct {
	Format *MediaInfo `json:"format"`
	Stream []*StreamInfo `json:"streams"`
}

func (this *CommandResult) GetFormat() *MediaInfo {
	if this.Format == nil {
		return &MediaInfo{
			Duration: "",
		}
	}

	return this.Format
}

func (this *CommandResult) GetStream() *StreamInfo {
	if this.Stream == nil || len(this.Stream) == 0 {
		return &StreamInfo{
			Width: 0,
			Height: 0,
		}
	}

	return this.Stream[0]
}

type MediaInfo struct {
	Duration string `json:"duration"`
}

func (this *MediaInfo) GetDuration() (float64, error) {
	duration, err := time.Parse("15:04:05", this.Duration)
	if err != nil {
		return 0, err
	}
	return duration.Sub(time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)).Seconds(), nil
}

func NewFFprobe(binPath string) *FFprobe {
	return &FFprobe{
		bin: binPath,
	}
}

func (this *FFprobe) GetMediaInfo(mediaPath string) (*CommandResult, error) {
	info := &CommandResult{}

	reader, err := NewBuilder(this.bin).SetParams("-v", "error", "-select_streams", "v:0", "-show_entries",
		"format=duration", "-show_entries", "stream=height,width", "-pretty", "-of", "json", "-hide_banner", "-i",
		mediaPath).Run()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	decoder := json.NewDecoder(reader)
	err = decoder.Decode(info)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	
	log.Debugf("resolution: %d x %d", info.GetStream().Width, info.GetStream().Height)

	return info, nil
}

type FFmpeg struct {
	bin     string
	builder *CommandBuilder
	outHeight int
}

func NewFFmpeg(binPath string) *FFmpeg {
	return &FFmpeg{
		bin: binPath,
		outHeight: 720,
	}
}

func (this *FFmpeg) SetOutputHeight(height int) *FFmpeg {
	this.outHeight = height
	return this
}

func (this *FFmpeg) SplitSnap(mediaPath string, duration float64, splitSize float64, outPath string) (error) {
	if this.builder != nil {
		err := this.builder.Stop()
		if err != nil {
			log.Error(err)
		}
	}

	if _, err := os.Stat(outPath); err != nil && os.IsNotExist(err) {
		os.MkdirAll(outPath, os.ModePerm)
	}
	
	counter := int(splitSize)
	starQueue := make(chan bool, 2)
	doneQueue := make(chan bool, 0)
	jobCount := 1
	
	defer close(starQueue)
	defer close(doneQueue)
	
	step := int(duration / splitSize * 1000);
	stepSec := time.Millisecond * time.Duration(step)
	current := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	
	for i := 0; i < counter; i++ {
		go func(index int) {
			starQueue <- true
			defer func() {
				<-starQueue
				doneQueue <- true
			}()
			
			position := current.Add(stepSec * time.Duration(index)).Format("15:04:05.000")
			
			this.builder = NewBuilder(this.bin).SetParams(
				"-ss", position,
				"-y",
				"-i", filepath.ToSlash(mediaPath),
				"-filter:v", fmt.Sprintf("scale=-1:%d",
					this.outHeight),
				"-vframes", "1",
				filepath.ToSlash(fmt.Sprintf("%s/snapshot-%d.png", outPath, index + 1)))
			
			build := this.builder
			done, err := build.Start()
			if err != nil {
				log.Error(err)
			}
			
			<- done
			defer close(done)
			
		}(i)
	}
	
	for jobCount < counter {
		<-doneQueue
		jobCount++
	}


	return nil
}