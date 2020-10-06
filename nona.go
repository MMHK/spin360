package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	_ "golang.org/x/image/tiff"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/disintegration/imaging"
)

type NonaWrapper struct {
	Bin    string
	UseGPU bool
}

func NewNonaWrapper(binPath string) *NonaWrapper {
	return &NonaWrapper{
		Bin:    binPath,
		UseGPU: false,
	}
}

type FileNode struct {
	Info     os.FileInfo
	FullPath string
}

func (this *NonaWrapper) GetImgSize(reader io.Reader) (int, int, error) {
	im, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, err
	}

	return im.Width, im.Height, nil
}

type PannellumConfig struct {
	Type string            `json:"type"`
	Config *MultiResConfig `json:"multiRes"`
}

type MultiResConfig struct {
	BasePath string `json:"basePath"`
	Path string `json:"path"`
	FallbackPath string `json:"fallbackPath"`
	Extension string `json:"extension"`
	TileResolution int `json:"tileResolution"`
	MaxLevel int `json:"maxLevel"`
	CubeResolution int `json:"cubeResolution"`
}

//func (this *NonaWrapper) Generate(imageFilePath string) ([]*FileNode, error)  {
//	imgFile, err := os.Open(imageFilePath)
//	if err != nil {
//		log.Error(err)
//		return nil, err
//	}
//	defer imgFile.Close()
//
//	width, height, err := this.GetImgSize(imgFile)
//	if err != nil {
//		log.Error(err)
//		return nil, err
//	}
//
//	haov := 360
//	vaov := 180
//
//	if width / height != 2 {
//		return nil, errors.New(`the image width is not twice the image height.`)
//	}
//
//	cubeSize := math.Round(8 * float64(360 / haov) * (float64(width) / math.Pi) / 8)
//
//}

func (this *NonaWrapper) CreateCuteFace(TempPath string) error {
	log.Info(`Generating cube faces...`)
	configFilePath := filepath.Join(TempPath, `cubic.pto`)
	args := []string{`-d`, `-o`, filepath.ToSlash(filepath.Join(TempPath, `face`)), configFilePath}
	if this.UseGPU {
		args[0] = `-g`
	}
	var outPipe bytes.Buffer
	var errorPipe bytes.Buffer

	cmd := exec.Command(this.Bin, args...)
	cmd.Stdout = &outPipe
	cmd.Stderr = &errorPipe
	err := cmd.Run()
	if err != nil {
		log.Error(err)
		return errors.New(errorPipe.String())
	}

	log.Info(outPipe.String())

	return err
}

func (this *NonaWrapper) GenerateCubicConfigFile(distFileName string, imgFileName string,
	width int, height int, haov int) error {
	log.Info(`Generating Cubic Config File...`)

	cubeSize := int(8 * (float64(360/haov) * float64(width) / math.Pi / 8))

	pitch := 0
	prefix := fmt.Sprintf(`i a0 b0 c0 d0 e0 f4 h%d w%d n"%s" r0 v%d`,
		height, width, imgFileName, haov)

	buff := bytes.NewBuffer([]byte{})
	_, err := fmt.Fprintln(buff, fmt.Sprintf(`p E0 R0 f0 h%d w%d n"TIFF_m" u0 v90`, cubeSize, cubeSize))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(buff, `m g1 i0 m2 p0.00784314`)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y0`, prefix, pitch))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y180`, prefix, pitch))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y0`, prefix, pitch-90))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y0`, prefix, pitch+90))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y90`, prefix, pitch))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y-90`, prefix, pitch))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(buff, `v`)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(buff, `*`)
	if err != nil {
		return err
	}

	conf, err := os.Create(distFileName)
	if err != nil {
		return err
	}
	defer conf.Close()

	_, err = io.Copy(conf, buff)
	if err != nil {
		return err
	}

	return nil
}

func (this *NonaWrapper) GeneratingTiles(cubeSize int, tempDir string) error {
	log.Info(`Generating tiles...`)

	faces := []string{"face0000.tif", "face0001.tif", "face0002.tif", "face0003.tif", "face0004.tif", "face0005.tif"}
	faceLetters := []string{"f", "b", "u", "d", "l", "r"}
	extension := ".jpg"
	tileSize := 512
	if tileSize > cubeSize {
		tileSize = cubeSize
	}
	tiles := int(math.Ceil(float64(cubeSize) / float64(tileSize)))
	levels := int(math.Ceil(math.Log2(float64(cubeSize) / float64(tileSize)))) + 1
	if int(math.Round(float64(cubeSize) / math.Pow(2, float64(levels - 2)))) == tileSize {
		levels -= 1
	}

	for f := 0; f < 6; f++ {
		size := cubeSize
		facePath := filepath.Join(tempDir, faces[f])
		if _, err := os.Stat(facePath); err == nil {
			img, err := imaging.Open(facePath)
			if err != nil {
				log.Error(err)
				return err
			}

			for level := levels; level > 0; level-- {
				if level < levels {
					img = imaging.Resize(img, size, size, imaging.Lanczos)
				}

				for i := 0; i < tiles; i++ {
					for j := 0; j < tiles; j++ {
						left := j * tileSize
						upper := i * tileSize
						right := int(math.Min(float64(j) * float64(tileSize) + float64(tileSize), float64(size)))
						lower := int(math.Min(float64(i) *float64(tileSize) + float64(tileSize), float64(size)))

						tile := imaging.Crop(img, image.Rect(left, upper, right, lower))
						tilePath := filepath.Join(tempDir, fmt.Sprintf(`%d/%s%d_%d%s`, level, faceLetters[f], i, j, extension))
						tileDir := filepath.Dir(tilePath)
						if _, err := os.Stat(tileDir); err != nil && os.IsNotExist(err) {
							os.MkdirAll(tileDir, os.ModePerm)
						}
						err := imaging.Save(tile, tilePath, imaging.JPEGQuality(95))
						if err != nil {
							log.Error(err)
							return err
						}
					}
				}
 			}

			size = int(size / 2)
		}
	}

	return nil
}

func (this *NonaWrapper) GenerateFallback(tempDir string) error {
	log.Info(`Generating fallback...`)

	faces := []string{"face0000.tif", "face0001.tif", "face0002.tif", "face0003.tif", "face0004.tif", "face0005.tif"}
	faceLetters := []string{"f", "b", "u", "d", "l", "r"}
	extension := ".jpg"
	fallbackSize := 1024

	fallbackDir := filepath.Join(tempDir, "fallback")
	if _, err := os.Stat(fallbackDir); err != nil && os.IsNotExist(err) {
		os.MkdirAll(fallbackDir, os.ModePerm)
	}
	for f := 0; f < 6; f++ {
		facePath := filepath.Join(tempDir, faces[f])
		img, err := imaging.Open(facePath)
		if err != nil {
			log.Error(err)
			return err
		}
		img = imaging.Resize(img, fallbackSize, fallbackSize, imaging.Lanczos)

		imgPath := filepath.Join(fallbackDir, fmt.Sprintf(`%s%s`, faceLetters[f], extension))
		err = imaging.Save(img, imgPath, imaging.JPEGQuality(95))
		if err != nil {
			log.Error(err)
			return err
		}
	}

	return nil
}

func (this *NonaWrapper) GenerateConfigJSON(cubeSize int, tempDir string) error {
	jsonPath := filepath.Join(tempDir, "config.json")
	jsonFile, err := os.Create(jsonPath)
	if err != nil {
		log.Error(err)
		return err
	}
	defer jsonFile.Close()

	conf := &PannellumConfig{
		Type: "multires",
		Config: &MultiResConfig{
			BasePath:       "",
			Path:           "/%l/%s%y_%x",
			FallbackPath:   "/fallback/%s",
			Extension:      "jpg",
			TileResolution: 512,
			MaxLevel:       4,
			CubeResolution: cubeSize,
		},
	}

	encoder := json.NewEncoder(jsonFile)
	err = encoder.Encode(conf)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
