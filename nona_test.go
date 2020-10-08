package main

import (
	"encoding/json"
	_ "image/jpeg"
	"os"
	"testing"
)

func GetNona() (*NonaWrapper, error) {
	return NewNonaWrapper(getLocalPath("./tests/sample.jpg")), nil
}

func TestNonaWrapper_GenerateCubicConfigFile(t *testing.T) {
	Nona, err := GetNona()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	configFile := getLocalPath("./tests/cubic.pto")
	cuteSize, err := Nona.GenerateCubicConfigFile(configFile)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	t.Log(cuteSize)
}

func TestNonaWrapper_CreateCuteFace(t *testing.T) {
	Nona, err := GetNona()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	configFile := getLocalPath("./tests/cubic.pto")
	_, err = Nona.GenerateCubicConfigFile(configFile)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	err = Nona.CreateCuteFace(getLocalPath("./tests/"))
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
}

func TestNonaWrapper_GeneratingTiles(t *testing.T) {
	Nona, err := GetNona()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	configFile := getLocalPath("./tests/cubic.pto")
	cubeSize, err := Nona.GenerateCubicConfigFile(configFile)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	
	err = Nona.CreateCuteFace(getLocalPath("./tests/"))
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	_,_,err = Nona.GeneratingTiles(cubeSize, getLocalPath("./tests"))
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
}

func TestNonaWrapper_GeneratingFallback(t *testing.T) {
	Nona, err := GetNona()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	err = Nona.GenerateFallback(getLocalPath("./tests"))
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
}

func TestNonaWrapper_GenerateConfigJSON(t *testing.T) {
	Nona, err := GetNona()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	configFile := getLocalPath("./tests/cubic.pto")
	cubeSize, err := Nona.GenerateCubicConfigFile(configFile)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	config, err := Nona.GenerateConfigJSON(cubeSize, 4, 512)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	t.Logf("%+v", config)
}

func TestNonaWrapper_Generate(t *testing.T) {
	Nona, err := GetNona()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	conf, err := Nona.Generate(getLocalPath("./tests"))
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	configPath := getLocalPath("./tests/config.json")
	configFile, err := os.Create(configPath)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	defer configFile.Close()

	encoder := json.NewEncoder(configFile)
	err = encoder.Encode(conf)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
}