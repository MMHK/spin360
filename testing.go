package main

import (
	"path/filepath"
	"runtime"
)

const FFPROBE_BIN = "F:/grean/ffmpeg/bin/ffprobe.exe"

func loadConfig() (*Config, error) {
	err, conf := NewConfig(getLocalPath("./data/conf.json"))
	return conf, err
}


func getLocalPath(file string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), file)
}
