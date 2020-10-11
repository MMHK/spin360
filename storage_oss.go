package main

import (
	"crypto"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type OSSConfig struct {
	AccessKey  string `json:"access_key"`
	SecretKey  string `json:"secret_key"`
	Bucket     string `json:"bucket"`
	EndPoint   string `json:"endpoint"`
	PrefixPath string `json:"prefix"`
}

type OSSStorage struct {
	Conf *OSSConfig
	client *oss.Client
}

type OSSWebPostConfigStruct struct {
	Expiration string     `json:"expiration"`
	Conditions [][]string `json:"conditions"`
}

type OSSWebFormMultipart struct {
	AccessKeyId   string `json:"OSSAccessKeyId"`
	Host          string `json:"host"`
	Signature     string `json:"signature"`
	Policy        string `json:"policy"`
	FileName      string `json:"key"`
	Callback      string `json:"callback"`
	SuccessStatus int    `json:"success_action_status"`
}

type OSSCallbackParam struct {
	CallbackUrl      string `json:"callbackUrl"`
	CallbackBody     string `json:"callbackBody"`
	CallbackBodyType string `json:"callbackBodyType"`
}

func NewOSSStorage(conf *OSSConfig) (*OSSStorage, error) {
	client, err := oss.New(conf.EndPoint, conf.AccessKey, conf.SecretKey)
	if err != nil {
		return nil, err
	}
	
	return &OSSStorage{
		Conf: conf,
		client: client,
	}, nil
}

func (this *OSSStorage) GenerateFormMultipart(UploadFileName string, expire time.Duration, callback string) (*OSSWebFormMultipart, error) {

	callbackParam := OSSCallbackParam{
		CallbackUrl:      fmt.Sprintf("%s;https://aliyun-oss.mixmedia.com/", callback),
		CallbackBody:     `{"mimeType":${mimeType}, "size":${size}, "filename":${object}, "bucket":${bucket}}`,
		CallbackBodyType: "application/json",
	}

	jsonCallBackParam, err := json.Marshal(callbackParam)
	if err != nil {
		return nil, err
	}
	base64CallBackParam := base64.StdEncoding.EncodeToString(jsonCallBackParam)

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return nil, err
	}
	expireAt := time.Now().In(loc).Add(expire)

	uploadOptions := OSSWebPostConfigStruct{
		Expiration: expireAt.Format("2006-01-02T15:04:05Z"),
		Conditions: [][]string{
			[]string{
				"starts-with",
				"$key",
				this.Conf.PrefixPath,
			},
		},
	}

	jsonOpions, err := json.Marshal(uploadOptions)
	if err != nil {
		return nil, err
	}

	base64uploadOptions := base64.StdEncoding.EncodeToString(jsonOpions)

	signHash := hmac.New(func() hash.Hash { return sha1.New() }, []byte(this.Conf.SecretKey))
	_, err = io.WriteString(signHash, base64uploadOptions)
	if err != nil {
		return nil, err
	}
	base64signedUploadOptions := base64.StdEncoding.EncodeToString(signHash.Sum(nil))

	filename := fmt.Sprintf("%s/%s%s", this.Conf.PrefixPath, uuid.NewV4(), filepath.Ext(UploadFileName))

	return &OSSWebFormMultipart{
		AccessKeyId:   this.Conf.AccessKey,
		Host:          fmt.Sprintf("//%s.%s", this.Conf.Bucket, this.Conf.EndPoint),
		Signature:     base64signedUploadOptions,
		Policy:        base64uploadOptions,
		FileName:      filename,
		Callback:      base64CallBackParam,
		SuccessStatus: 200,
	}, nil
}

func VerifyWebUploadCallback(req *http.Request) (bool, error) {
	publicKeyURLBase64 := req.Header.Get("x-oss-pub-key-url")
	if len(publicKeyURLBase64) <= 0 {
		return false, errors.New("no x-oss-pub-key-url field in Request header ")
	}
	publicKeyURL, err := base64.StdEncoding.DecodeString(publicKeyURLBase64)
	if err != nil {
		log.Error(err)
		return false, err
	}
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	responsePublicKeyURL, err := client.Get(string(publicKeyURL))
	if err != nil {
		log.Error(err)
		return false, err
	}
	bytePublicKey, err := ioutil.ReadAll(responsePublicKeyURL.Body)
	if err != nil {
		log.Error(err)
		return false, err
	}
	defer responsePublicKeyURL.Body.Close()

	strAuthorizationBase64 := req.Header.Get("authorization")
	if len(strAuthorizationBase64) <= 0 {
		return false, errors.New("Failed to get authorization field from request header.")
	}
	byteAuthorization, err := base64.StdEncoding.DecodeString(strAuthorizationBase64)
	if err != nil {
		log.Error(err)
		return false, err
	}

	bodyContent, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error(err)
		return false, err
	}
	defer req.Body.Close()

	strURLPathDecode, err := url.PathUnescape(req.URL.Path)
	if err != nil {
		log.Error(err)
		return false, err
	}

	strAuth := ""
	if len(req.URL.RawQuery) == 0 {
		strAuth = fmt.Sprintf("%s\n%s", strURLPathDecode, string(bodyContent))
	} else {
		strAuth = fmt.Sprintf("%s?%s\n%s", strURLPathDecode, req.URL.RawQuery, string(bodyContent))
	}

	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strAuth))
	byteMD5 := md5Ctx.Sum(nil)

	pubBlock, _ := pem.Decode(bytePublicKey)
	if pubBlock == nil {
		return false, errors.New("Failed to parse PEM block containing the public key")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		log.Error(err)
		return false, err
	}
	if pubInterface == nil {
		return false, errors.New("x509.ParsePKIXPublicKey(publicKey) failed")
	}

	pub, ok := pubInterface.(*rsa.PublicKey)
	if ok {
		err = rsa.VerifyPKCS1v15(pub, crypto.MD5, byteMD5, byteAuthorization)
		if err != nil {
			log.Error(err)
			return false, err
		}

		return true, nil
	}

	return false, errors.New("Signature Verification is Failed")
}

func (this *OSSStorage) Upload(localPath string, Key string) (path string, url string, err error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", "", err
	}
	
	defer file.Close()
	
	bucket, err := this.client.Bucket(this.Conf.Bucket)
	if err != nil {
		log.Error(err)
		return "", "", err
	}
	options := []oss.Option{
		oss.ObjectACL(oss.ACLPublicRead),
	}
	
	path = filepath.ToSlash(filepath.Join(this.Conf.PrefixPath, Key))
	
	if strings.EqualFold(strings.ToLower(filepath.Ext(path)), ".plist") {
		options = append(options, oss.ContentType("text/xml"), oss.ContentDisposition("inline"))
	}
	
	err = bucket.PutObject(path, file, options...)
	if err != nil {
		log.Error(err)
		return "", "", err
	}
	
	return path, filepath.ToSlash(
		filepath.Join(fmt.Sprintf("https://%s.%s", bucket.BucketName, this.Conf.EndPoint), path)), nil
}

func (this *OSSStorage) PutContent(content string, Key string, opt *UploadOptions) (path string, url string, err error) {
	bucket, err := this.client.Bucket(this.Conf.Bucket)
	if err != nil {
		log.Error(err)
		return "", "", err
	}
	options := []oss.Option{
		oss.ObjectACL(oss.ACLPublicRead),
	}
	if len(opt.ContentType) > 0 {
		options = append(options, oss.ContentType(opt.ContentType))
	}
	
	path = filepath.ToSlash(filepath.Join(this.Conf.PrefixPath, Key))
	
	err = bucket.PutObject(path, strings.NewReader(content), options...)
	if err != nil {
		log.Error(err)
		return "", "", err
	}
	
	return path, filepath.ToSlash(
		filepath.Join(fmt.Sprintf("https://%s.%s", bucket.BucketName, this.Conf.EndPoint), path)), nil
}

func (this *OSSStorage) Get(Key string) (io.Reader, error) {
	bucket, err := this.client.Bucket(this.Conf.Bucket)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	remoteKey := filepath.ToSlash(fmt.Sprintf("%s%s", this.Conf.PrefixPath, Key))

	reader, err := bucket.GetObject(remoteKey)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return reader, nil
}

func (this *OSSStorage) URL(Key string) string {
	return filepath.ToSlash(fmt.Sprintf("//%s.%s%s%s", this.Conf.Bucket, this.Conf.EndPoint, this.Conf.PrefixPath, Key))
}