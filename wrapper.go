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

type CommandResult struct {
	Format *MediaInfo `json:"format"`
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

func (this *FFprobe) GetMediaInfo(mediaPath string) (*MediaInfo, error) {
	info := &CommandResult{}

	reader, err := NewBuilder(this.bin).SetParams("-v", "error", "-show_entries",
		"format=duration", "-pretty", "-of", "json", "-hide_banner", "-i",
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

	return info.Format, nil
}

type FFmpeg struct {
	bin     string
	builder *CommandBuilder
}

func NewFFmpeg(binPath string) *FFmpeg {
	return &FFmpeg{
		bin: binPath,
	}
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

	this.builder = NewBuilder(this.bin).SetParams("-i", filepath.ToSlash(mediaPath),
		"-r", fmt.Sprintf("%f", (splitSize / duration) ),
		"-vframes", fmt.Sprintf("%d", int(splitSize)),
		filepath.ToSlash(fmt.Sprintf("%s/snapshot-%%d.png", outPath)))

	build := this.builder
	done, err := build.Start()
	if err != nil {
		log.Error(err)
		return err
	}

	<- done
	defer close(done)

	return nil
}