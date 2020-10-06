package main

import (
	"bytes"
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

var (
	faces = []string{"face0000.tif", "face0001.tif", "face0002.tif", "face0003.tif", "face0004.tif", "face0005.tif"}
	faceLetters = []string{"f", "b", "u", "d", "l", "r"}
	extension = ".jpg"
)

const (
	TYPE_PANNELLUM_HOTSPOT_EMBED = "embed"
	TYPE_PANNELLUM_HOTSPOT_TEXT  = "text"
	TYPE_PANNELLUM_HOTSPOT_LINK  = "link"
)

type NonaWrapper struct {
	Bin    string
	UseGPU bool
	HaoV   int
	SrcImgPath string
}

func NewNonaWrapper(ImagePath string) *NonaWrapper {
	self := &NonaWrapper{
		UseGPU: false,
		HaoV: 360,
		SrcImgPath: ImagePath,
	}

	return self.GetBinPath()
}

func (this *NonaWrapper) SetBinPath(bin string) *NonaWrapper {
	this.Bin = bin

	return this
}

func (this *NonaWrapper) GetBinPath() *NonaWrapper {
	if len(this.Bin) <= 0 {
		binPath, err := exec.LookPath("nona")
		if err != nil {
			log.Error(err)
			return this
		}

		this.Bin = binPath
	}

	return this
}

func (this *NonaWrapper) GetImgSize(reader io.Reader) (int, int, error) {
	im, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, err
	}

	return im.Width, im.Height, nil
}

type PannellumConfig struct {
	Type    string              `json:"type"`
	Config  *MultiResConfig     `json:"multiRes"`
	HotSpot []*PannellumHotSpot `json:"hotSpots"`
}

type PannellumHotSpot struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Link  string `json:"link"`
	Pitch int    `json:"pitch"`
	Yaw   int    `json:"yaw"`
	Id    string `json:"id"`
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

func (this *NonaWrapper) Generate(distDir string) (*PannellumConfig, error) {
	cuteConfig := filepath.Join(distDir, `cubic.pto`)
	cubeSize, err := this.GenerateCubicConfigFile(cuteConfig)
	if err != nil {
		return nil, err
	}
	err = this.CreateCuteFace(distDir)
	if err != nil {
		return nil, err
	}

	err = this.GeneratingTiles(cubeSize, distDir)
	if err != nil {
		return nil, err
	}

	err = this.GenerateFallback(distDir)
	if err != nil {
		return nil, err
	}

	this.ClearCuteFaceFiles(distDir)

	return this.GenerateConfigJSON(cubeSize)
}

func (this *NonaWrapper) GenerateFromReader(distDir string, reader io.Reader) (*PannellumConfig, error) {
	sampleImagePath := filepath.ToSlash(filepath.Join(distDir, `sample.tmp`))
	sampleImage, err := os.Create(sampleImagePath)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	_, err = io.Copy(sampleImage, reader)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	sampleImage.Close()

	this.SrcImgPath = sampleImagePath

	defer func() {
		os.Remove(sampleImagePath)
	}()

	return this.Generate(distDir)
}

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

func (this *NonaWrapper) ClearCuteFaceFiles(TempDir string) {
	log.Info(`Clearing face files...`)

	files := append(faces, `cubic.pto`)
	for _, file := range files {
		fullPath := filepath.Join(TempDir, file)
		err := os.Remove(fullPath)
		if err != nil {
			log.Error(err)
		}
	}
}

func (this *NonaWrapper) GenerateCubicConfigFile(distFileName string) (int, error) {
	log.Info(`Generating Cubic Config File...`)

	img, err := os.Open(this.SrcImgPath)
	if err != nil {
		log.Error(err)
		return 0, err
	}
	defer img.Close()

	width, height, err := this.GetImgSize(img)
	if err != nil {
		log.Error(err)
		return 0, err
	}

	cubeSize := int(8 * (float64(360/this.HaoV) * float64(width) / math.Pi / 8))

	pitch := 0
	prefix := fmt.Sprintf(`i a0 b0 c0 d0 e0 f4 h%d w%d n"%s" r0 v%d`,
		height, width, filepath.ToSlash(this.SrcImgPath), this.HaoV)

	buff := bytes.NewBuffer([]byte{})
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`p E0 R0 f0 h%d w%d n"TIFF_m" u0 v90`, cubeSize, cubeSize))
	if err != nil {
		return 0, err
	}
	_, err = fmt.Fprintln(buff, `m g1 i0 m2 p0.00784314`)
	if err != nil {
		return 0, err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y0`, prefix, pitch))
	if err != nil {
		return 0, err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y180`, prefix, pitch))
	if err != nil {
		return 0, err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y0`, prefix, pitch-90))
	if err != nil {
		return 0, err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y0`, prefix, pitch+90))
	if err != nil {
		return 0, err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y90`, prefix, pitch))
	if err != nil {
		return 0, err
	}
	_, err = fmt.Fprintln(buff, fmt.Sprintf(`%s p%d y-90`, prefix, pitch))
	if err != nil {
		return 0, err
	}
	_, err = fmt.Fprintln(buff, `v`)
	if err != nil {
		return 0, err
	}
	_, err = fmt.Fprintln(buff, `*`)
	if err != nil {
		return 0, err
	}

	conf, err := os.Create(distFileName)
	if err != nil {
		return 0, err
	}
	defer conf.Close()

	_, err = io.Copy(conf, buff)
	if err != nil {
		return 0, err
	}

	return cubeSize, nil
}

func (this *NonaWrapper) GeneratingTiles(cubeSize int, tempDir string) error {
	log.Info(`Generating tiles...`)

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

func (this *NonaWrapper) GenerateConfigJSON(cubeSize int) (*PannellumConfig, error) {
	log.Info(`Generating config ...`)

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

	return conf, nil
}
