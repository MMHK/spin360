package main

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
)

type NonaWrapper struct {
	Bin string
}

func NewNonaWrapper(binPath string) *NonaWrapper {
	return &NonaWrapper{
		Bin: binPath,
	}
}

type FileNode struct {
	Info os.FileInfo
	FullPath string
}

func (this *NonaWrapper) GetImgSize(reader io.Reader) (int, int, error) {
	im, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, err
	}

	return im.Width, im.Height, nil
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

//func (this *NonaWrapper) GenerateCubicConfigFile(imgFileName string,
//	width int, height int, haov int, cubeSize float64) (string, error) {
//	pitch := 0
//	prefix := fmt.Sprintf(`i a0 b0 c0 d0 e 0 f4 h %d w %d n %s r0 v %d`,
//		height, width, imgFileName, haov)
//
//	builder := stringset.NewBuilder()
//	builder.Add(fmt.Sprintf(`p E0 R0 f0 h %d w %d n"TIFF_m" u0 v90`, cubeSize, cubeSize), "\n")
//	builder.Add(`m g1 i0 m2 p0.00784314`, "\n")
//	builder.Add(fmt.Sprintf(`%s p %d y0`, prefix, pitch), "\n")
//}