package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"github.com/satori/go.uuid"
)

type Worker struct {
	Conf *Config
}

func NewWorker(conf *Config) *Worker {
	return &Worker{
		Conf: conf,
	}
}

func (this *Worker) TempDir(callback func(tempDir string) error) error {
	id := uuid.NewV4()
	tid := fmt.Sprintf("%s", id)

	tempDirPath := filepath.Join(this.Conf.TempPath, tid)
	if _, err := os.Stat(tempDirPath); os.IsNotExist(err) {
		err = os.MkdirAll(tempDirPath, os.ModePerm)
		if err != nil {
			return err
		}
	}

	defer os.RemoveAll(tempDirPath)

	return callback(tempDirPath)
}

func (this *Worker) Split(ctx context.Context, src io.Reader, size int) (io.Reader, error)  {
	var zipPath string

	err := this.TempDir(func(tempDir string) error {
		videoPath := filepath.Join(tempDir, "video.tmp")
		video, err := os.Create(videoPath)
		if err != nil {
			log.Error(err)
			return err
		}
		defer video.Close()

		_, err = io.Copy(video, src)
		if err != nil {
			log.Error(err)
			return err
		}

		ffprobe := NewFFprobe(this.Conf.FFMpegConf.FFProbe)
		info, err := ffprobe.GetMediaInfo(videoPath)
		if err != nil {
			log.Error(err)
			return err
		}
		duration, err := info.GetDuration()
		if err != nil {
			log.Error(err)
			return err
		}
		imageDir := filepath.Join(tempDir, "snap")
		ffmpeg := NewFFmpeg(this.Conf.FFMpegConf.FFmpeg)
		err = ffmpeg.SplitSnap(videoPath, duration, float64(size), imageDir)
		if err != nil {
			log.Error(err)
			return err
		}

		zipPath = tempDir + ".zip"
		err = ZipFolder(imageDir, zipPath)
		if err != nil {
			log.Error(err)
			return err
		}

		return nil
	})

	if err != nil {
		log.Error(err)
		return nil, err
	}

	zipFile, err := os.Open(zipPath)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	go func() {
		select {
			case <-ctx.Done():
				zipFile.Close()
				os.Remove(zipPath)
				return
		}
	}()

	return zipFile, nil
}

func (this *Worker) S3(src io.Reader, size int) ([]string, error) {
	s3List := make([]string, 0)
	err := this.TempDir(func(tempDir string) error {
		videoPath := filepath.Join(tempDir, "video.tmp")
		video, err := os.Create(videoPath)
		if err != nil {
			log.Error(err)
			return err
		}
		defer video.Close()
		
		_, err = io.Copy(video, src)
		if err != nil {
			log.Error(err)
			return err
		}
		
		ffprobe := NewFFprobe(this.Conf.FFMpegConf.FFProbe)
		info, err := ffprobe.GetMediaInfo(videoPath)
		if err != nil {
			log.Error(err)
			return err
		}
		duration, err := info.GetDuration()
		if err != nil {
			log.Error(err)
			return err
		}
		imageDir := filepath.Join(tempDir, "snap")
		ffmpeg := NewFFmpeg(this.Conf.FFMpegConf.FFmpeg)
		err = ffmpeg.SplitSnap(videoPath, duration, float64(size), imageDir)
		if err != nil {
			log.Error(err)
			return err
		}
		
		s3, err := NewS3Storage(this.Conf.S3)
		if err != nil {
			log.Error(err)
			return err
		}
		
		queue := make(chan bool, 0)
		defer close(queue)
		jobCount := 0
		
		err = filepath.Walk(imageDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			remotePath := filepath.ToSlash(filepath.Join(filepath.Base(tempDir), filepath.Base(path)))
			
			jobCount++
			go func(path string, remotePath string, s3 IStorage) {
				defer func() {
					queue <- true
				}()
				log.Infof("%s => s3:%s", path, remotePath)
				
				_, url, err := s3.Upload(path, remotePath)
				if err != nil {
					log.Error(err)
					return
				}
				s3List = append(s3List, url)
			}(path, remotePath, s3)
			
			return nil
		})
		
		for jobCount > 0 {
			<-queue
			jobCount--
		}
		
		if err != nil {
			log.Error(err)
			return err
		}
		
		return nil
	})
	
	if err != nil {
		log.Error(err)
		return nil, err
	}
	
	return s3List, nil
}

func (this *Worker) UpdatePlayConfig(hash string, conf *Spin360Config) (string, error) {
	remoteKey := fmt.Sprintf("%s.json", hash)

	s3, err := NewS3Storage(this.Conf.S3)
	if err != nil {
		log.Error(err)
		return "", err
	}

	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)
	err = encoder.Encode(conf)
	if err != nil {
		log.Error(err)
		return "", err
	}

	_, url, err := s3.PutContent(buf.String(), remoteKey, &UploadOptions{
		ContentType: "application/json",
	})
	if err != nil {
		log.Error(err)
		return "", err
	}

	return url, nil
}

func (this *Worker) GetConfig(hash string) (*Spin360Config, error) {
	remoteKey := fmt.Sprintf("%s.json", hash)

	s3, err := NewS3Storage(this.Conf.S3)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	reader, err := s3.Get(remoteKey)
	if err != nil {
		return nil, err
	}

	conf := new(Spin360Config)

	decoder := json.NewDecoder(reader)
	err = decoder.Decode(conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (this *Worker) SavePlayConfig(conf *Spin360Config) (string, error) {
	return this.UpdatePlayConfig(uuid.NewV4().String(), conf)
}

func ZipFolder(srcDirPath string, distFileName string) (err error) {

	zipfile, e := os.Create(distFileName)
	if e != nil {
		log.Error("create file error:", e)
		return e
	}
	defer zipfile.Close()

	zipWriter := zip.NewWriter(zipfile)
	defer zipWriter.Close()

	srcDirPath = filepath.FromSlash(srcDirPath)

	err = filepath.Walk(srcDirPath, func(localPath string, info os.FileInfo, err error) (_e error) {
		if err != nil {
			log.Error("Walk file error:", err)
			return
		}
		if info.Mode().IsDir() {
			return
		}
		file, err := os.Open(localPath)
		if err != nil {
			log.Error("open file error:", err)
			return err
		}
		defer file.Close()

		fileHeader := new(zip.FileHeader)
		fileHeader.Name = strings.TrimLeft(filepath.ToSlash(
			strings.Replace(localPath, srcDirPath, "", 1)), "/")
		fileHeader.Method = zip.Store
		fileHeader.Modified = info.ModTime().UTC()

		writer, err := zipWriter.CreateHeader(fileHeader)
		if err != nil {
			log.Error(err)
			return err
		}
		if _, err = io.Copy(writer, file); err != nil {
			log.Error(err)
			return err
		}

		return nil
	})

	return
}