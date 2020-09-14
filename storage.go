package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type S3Storage struct {
	Conf    *S3Config
	session *session.Session
}

type UploadOptions struct {
	ContentType string
}

type IStorage interface {
	Upload(localPath string, Key string) (string, string, error)
	PutContent(content string, Key string, opt *UploadOptions) (string, string, error)
	Get(Key string) (io.Reader, error)
}

func NewS3Storage(conf *S3Config) (IStorage, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(conf.Region),
		Credentials: credentials.NewStaticCredentials(conf.AccessKey, conf.SecretKey, ""),
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &S3Storage{
		Conf:    conf,
		session: sess,
	}, nil
}

func (this *S3Storage) Get(Key string) (io.Reader, error) {
	path := filepath.ToSlash(filepath.Join(this.Conf.PrefixPath, Key))

	svc := s3.New(this.session, aws.NewConfig())

	out, err := svc.GetObject(&s3.GetObjectInput {
		Bucket: aws.String(this.Conf.Bucket),
		Key: aws.String(path),
	})

	if err != nil {
		log.Error(err)
		return nil, err
	}

	return out.Body, nil
}

func (this *S3Storage) GetFileContentType(localPath string) (string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", err
	}

	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		log.Error(err)
		return "", err
	}

	contentType := http.DetectContentType(buffer)

	return contentType, nil
}

func (this *S3Storage) Upload(localPath string, Key string) (path string, url string, err error) {
	mimeType, err := this.GetFileContentType(localPath)
	if err != nil {
		log.Error(err)
		mimeType = "application/octet-stream"
	}

	file, err := os.Open(localPath)
	if err != nil {
		return "", "", err
	}

	defer file.Close()

	uploader := s3manager.NewUploader(this.session)
	path = filepath.ToSlash(filepath.Join(this.Conf.PrefixPath, Key))

	info, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(this.Conf.Bucket),
		Key:    aws.String(path),
		Body:   file,
		ACL:    aws.String("public-read"),
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return path, "", err
	}

	return path, info.Location, err
}

func (this *S3Storage) PutContent(content string, Key string, opt *UploadOptions) (path string, url string, err error) {
	uploader := s3manager.NewUploader(this.session)

	contentType := "application/octet-stream"
	if len(opt.ContentType) > 0 {
		contentType = opt.ContentType
	}

	path = filepath.ToSlash(filepath.Join(this.Conf.PrefixPath, Key))

	info, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(this.Conf.Bucket),
		Key:         aws.String(path),
		Body:        strings.NewReader(content),
		ACL:         aws.String("public-read"),
		ContentType: aws.String(contentType),
	})

	return path, info.Location, err
}
