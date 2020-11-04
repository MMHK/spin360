// config
package main

import (
	"encoding/json"
	"os"
)

const HOTSPOT_TYPE_EMBED = "embed"

const HOTSPOT_TYPE_LINK = "link"

const HOTSPOT_TYPE_TEXT = "text"

type HotSpotCoordinates struct {
	// 所在图片索引
	//
	// required: true
	PageIndex int `json:"index"`
	// X 坐标位置
	//
	// required: true
	X string `json:"x"`
	// Y 坐标位置
	//
	// required: true
	Y string `json:"y"`
}

type PageHotSpot struct {
	// 热点类型, 可能值 "embed", "link", "text"
	//
	// enum:
	//	- embed
	//	- link
	//	- text
	// required: true
	Type string `json:"type"`
	// Embed/Link 类型 URL
	//
	// required: true
	URL string `json:"url"`
	// Text 类型文字说明
	//
	// required: true
	Text string `json:"text"`
	// 热点坐标位置描述
	//
	// required: true
	Coordinates []*HotSpotCoordinates `json:"coordinate"`
}

type SpinPage struct {
	// 图片URL
	//
	// required: true
	ImageURL string `json:"img"`
}

// swagger:parameters configParams
type Spin360Params struct {
	//in: body
	Body *Spin360Config
}

//
// Spin360 player 配置
//
// swagger:model Spin360Config
type Spin360Config struct {
	// 页面URL 数组
	//
	// required: true
	Pages []*SpinPage `json:"page"`
	// 热点配置数组
	//
	// required: true
	HotSpot []*PageHotSpot `json:"hotspot"`
}

type VRHotSpot struct {
	//required: true
	Id string `json:"id"`
	//
	//
	//required: true
	Pitch int `json:"pitch"`
	//
	//
	// required: true
	Yaw int `json:"yaw"`
	// 热点类型, 可能值 "embed", "link", "text"
	//
	// enum:
	//	- embed
	//	- link
	//	- text
	// required: true
	Type string `json:"type"`
	// Embed/Link 类型 URL
	//
	// required: true
	URL string `json:"url"`
	// Text 类型文字说明
	//
	// required: true
	Text string `json:"text"`
}

//
// VR360 player 配置
//
// swagger:model VR360Config
type VR360Config struct {
	// 页面URL 数组
	//
	// required: true
	Source string `json:"src"`
	// 热点配置数组
	//
	// required: true
	HotSpots []*VRHotSpot `json:"hotspot"`
}

type FFMPEGConfig struct {
	FFmpeg  string `json:"ffmpeg"`
	FFProbe string `json:"ffprobe"`
}

type S3Config struct {
	AccessKey   string `json:"access_key"`
	SecretKey   string `json:"secret_key"`
	Bucket      string `json:"bucket"`
	Region      string `json:"region"`
	PrefixPath  string `json:"prefix"`
	VR360Prefix string `json:"vr360_prefix"`
}

type Config struct {
	Listen         string        `json:"listen"`
	FFMpegConf     *FFMPEGConfig `json:"ffmpeg"`
	S3             *S3Config     `json:"s3"`
	OSS            *OSSConfig    `json:"aliyun-oss"`
	WebRoot        string        `json:"web_root"`
	TempPath       string        `json:"temp"`
	MaxVideoHeight int           `json:"max_video_height"`
	sava_file      string
}

func NewConfig(filename string) (err error, c *Config) {
	c = &Config{}
	c.sava_file = filename
	err = c.load(filename)
	if err != nil {
		return err, nil
	}
	return nil, c
}

func (c *Config) load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Error(err)
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(c)
	if err != nil {
		log.Error(err)
	}
	return err
}

func (c *Config) Save() error {
	file, err := os.Create(c.sava_file)
	if err != nil {
		log.Error(err)
		return err
	}
	defer file.Close()
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		log.Error(err)
		return err
	}
	_, err = file.Write(data)
	if err != nil {
		log.Error(err)
	}
	return err
}
