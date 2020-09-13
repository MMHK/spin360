package main

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	conf_file := getLocalPath("./conf.json")
	err, conf := NewConfig(conf_file)
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
	err = conf.Save()
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
	t.Log(conf)
}
