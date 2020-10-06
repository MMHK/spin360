package main

import (
	_ "image/jpeg"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func GetNona() (*NonaWrapper, error) {
	binPath, err := exec.LookPath("nona")
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return NewNonaWrapper(binPath), nil
}

func TestNonaWrapper_GenerateCubicConfigFile(t *testing.T) {
	Nona, err := GetNona()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	configFile := getLocalPath("./tests/cubic.pto")
	err = Nona.GenerateCubicConfigFile(configFile, `sample.jpeg`, 2160,1080, 360)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
}


func TestNonaWrapper_CreateCuteFace(t *testing.T) {
	Nona, err := GetNona()
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	srcImgPath := filepath.ToSlash(getLocalPath("./tests/sample.jpeg"))
	img, err := os.Open(srcImgPath)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	defer img.Close()

	w, h, err := Nona.GetImgSize(img)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	configFile := getLocalPath("./tests/cubic.pto")
	err = Nona.GenerateCubicConfigFile(configFile, srcImgPath, w, h, 360)
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

	err = Nona.GeneratingTiles(2546, getLocalPath("./tests"))
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

	err = Nona.GenerateConfigJSON(2546, getLocalPath("./tests"))
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
}

func Test_Sample(t *testing.T) {
	cubeSize := 800
	tileSize := 512

	levels := int(math.Ceil(math.Log2(float64(cubeSize) / float64(tileSize)))) + 1
	if int(math.Round(float64(cubeSize) / math.Pow(2, float64(levels - 2)))) == tileSize {
		levels -= 1
	}

	t.Log(levels)
}
