//Package Video Splitter for spin360
//
//	Schemes: http, https
//	Host: API_HOST
//	BasePath: /
//	Version: 1.0.1
//
//	Consumes:
//	 - multipart/form-data
//	 - application/json
//
//	Produces:
//	 - application/json
//
//	swagger:meta
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

const (
	STATUS_TASK_STARTED = "STARTED"
	STATUS_TASK_RUNNING = "RUNNING"
	STATUS_TASK_DONE    = "DONE"
	STATUS_TASK_FAILED  = "FAILED"
)

type Task struct {
	ID     string      `json:"id"`
	Result interface{} `json:"data"`
	Error  string      `json:"error"`
	Status string      `json:"status"`
}

type HTTPService struct {
	config   *Config
	tasks    map[string]*Task
	taskLock chan bool
}

// swagger:response ServiceResult
type ServiceResult struct {
	Status bool        `json:"status"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`
}

func NewHTTP(conf *Config) *HTTPService {
	return &HTTPService{
		config:   conf,
		tasks:    make(map[string]*Task),
		taskLock: make(chan bool, 1),
	}
}

func (this *HTTPService) getHTTPHandler() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/", this.RedirectSwagger)
	r.HandleFunc("/split", this.Split)
	r.HandleFunc("/config", this.SavePlayerConfig)
	r.HandleFunc("/s3", this.S3)
	r.HandleFunc("/task", this.GetTask)
	r.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/",
		http.FileServer(http.Dir(fmt.Sprintf("%s/swagger", this.config.WebRoot)))))
	r.NotFoundHandler = http.HandlerFunc(this.NotFoundHandle)

	return r
}

func (this *HTTPService) Start() error {
	log.Info("http service starting")
	log.Infof("Please open http://%s\n", this.config.Listen)
	return http.ListenAndServe(this.config.Listen, this.getHTTPHandler())
}

func (this *HTTPService) NotFoundHandle(writer http.ResponseWriter, request *http.Request) {
	http.Error(writer, "handle not found!", 404)
	this.ResponseError(errors.New("handle not found!"), writer, 404)
}

func (this *HTTPService) RedirectSwagger(writer http.ResponseWriter, request *http.Request) {
	http.Redirect(writer, request, "/swagger/index.html", 301)
}

func GetMimeType(src *multipart.FileHeader) (string, string, error) {
	e, stat := src.Header["Content-Type"]
	if stat && len(e) > 0 {
		return e[0], src.Filename, nil
	}

	return "", "", errors.New("Not Found MimeInfo")
}

//
// swagger:operation POST /split post
//
// 分割Video 并下载截图zip包
//
// ---
// consumes:
//   - multipart/form-data
// produces:
//   - application/json
// parameters:
// - name: video
//   type: file
//   in: formData
//   required: true
//   description: 视频文件
// - name: splitSize
//   type: integer
//   in: formData
//   required: true
//   description: 截图总数
// responses:
//   200:
//     description: OK
//   500:
//     description: Error
//
//
func (this *HTTPService) Split(writer http.ResponseWriter, request *http.Request) {

	request.ParseMultipartForm(32 << 20)
	uploadFile, _, err := request.FormFile("video")
	if err != nil {
		log.Error(err)
		this.ResponseError(err, writer, 500)
		return
	}
	defer uploadFile.Close()

	size := request.FormValue("splitSize")

	splitSize, err := strconv.Atoi(size)
	if err != nil {
		log.Error(err)
		this.ResponseError(err, writer, 500)
		return
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute*30))
	defer cancel()

	worker := NewWorker(this.config)
	out, err := worker.Split(ctx, uploadFile, splitSize)
	if err != nil {
		log.Error(err)
		this.ResponseError(err, writer, 500)
		return
	}

	this.streamFile(out, time.Now().Format("20060102150405.zip"), writer)
}

//
// swagger:operation POST /s3 post
//
// 视频截图，返回task（任务）ID
//
// ---
// consumes:
//   - multipart/form-data
// produces:
//   - application/json
// parameters:
// - name: video
//   type: file
//   in: formData
//   required: true
//   description: 视频文件
// - name: splitSize
//   type: integer
//   in: formData
//   required: true
//   description: 截图总数
// responses:
//   200:
//     description: OK
//   500:
//     description: Error
//
//
func (this *HTTPService) S3(writer http.ResponseWriter, request *http.Request) {
	request.ParseMultipartForm(32 << 20)
	uploadFile, _, err := request.FormFile("video")
	if err != nil {
		log.Error(err)
		this.ResponseError(err, writer, 500)
		return
	}

	size := request.FormValue("splitSize")

	splitSize, err := strconv.Atoi(size)
	if err != nil {
		log.Error(err)
		this.ResponseError(err, writer, 500)
		return
	}

	task := this.createTask()

	go func() {
		defer uploadFile.Close()

		task.Status = STATUS_TASK_RUNNING
		this.UpdateTaskStatus(task.ID, task)

		worker := NewWorker(this.config)
		list, err := worker.S3(uploadFile, splitSize)
		if err != nil {
			log.Error(err)
			task.Status = STATUS_TASK_FAILED
			this.UpdateTaskStatus(task.ID, task)
			return
		}

		task.Status = STATUS_TASK_DONE
		task.Result = list
		this.UpdateTaskStatus(task.ID, task)
	}()

	this.ResponseJSON(&task, writer)
}

//
// swagger:operation POST /config configParams
//
// 保存播放器配置，返回配置文件的URL
//
// ---
// consumes:
//   - application/json
//   - multipart/form-data
// produces:
//   - application/json
// parameters:
// - name: Body
//   in: body
//   description: 配置
// - name: hash
//   type: string
//   in: query
//   description: 配置hash, 如提供即更新已有配置
// responses:
//   200:
//     description: OK
//   500:
//     description: Error
//
//
func (this *HTTPService) SavePlayerConfig(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	config := new(Spin360Config)
	err := decoder.Decode(config)
	if err != nil {
		this.ResponseError(err, writer, 500)
		return
	}

	hash := ""
	params := request.URL.Query()
	vals, ok := params["hash"]
	if ok && len(vals) > 0 {
		hash = vals[0]
	}

	url := ""
	worker := NewWorker(this.config)
	if len(hash) > 0 {
		url, err = worker.UpdatePlayConfig(hash, config)
	} else {
		url, err = worker.SavePlayConfig(config)
	}
	if err != nil {
		this.ResponseError(err, writer, 500)
		return
	}

	this.ResponseJSON(&ServiceResult{
		Status: true,
		Data:   url,
	}, writer)
}

// swagger:operation GET /task task
//
// 获取task （任务）状态
//
// ---
// consumes:
//   - multipart/form-data
// produces:
//   - application/json
// parameters:
// - name: id
//   type: string
//   in: query
//   required: true
//   description: Task（任务）ID
// responses:
//   200:
//     description: OK
//   500:
//     description: Error
//
//
func (this *HTTPService) GetTask(writer http.ResponseWriter, request *http.Request) {
	TaskID := request.FormValue("id")

	task, ok := this.tasks[TaskID]
	if !ok {
		this.ResponseError(errors.New("task not found"), writer, 500)
		return
	}

	this.ResponseJSON(&task, writer)
}

func (this *HTTPService) createTask() (*Task) {
	tid := fmt.Sprintf("%s", uuid.NewV4())

	this.taskLock <- true
	defer func() {
		<-this.taskLock
	}()

	task := &Task{
		ID:     tid,
		Status: STATUS_TASK_STARTED,
	}

	this.tasks[tid] = task

	return task
}

func (this *HTTPService) UpdateTaskStatus(uuid string, task *Task) {
	this.taskLock <- true
	defer func() {
		<-this.taskLock
	}()

	this.tasks[uuid] = task
}

func (this *HTTPService) RemoveTask(uuid string) {
	this.taskLock <- true
	defer func() {
		<-this.taskLock
	}()

	delete(this.tasks, uuid)
}

func (this *HTTPService) streamFile(out io.Reader, filename string, writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", "application/zip")
	writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	writer.WriteHeader(http.StatusOK)

	io.Copy(writer, out)
}

func (this *HTTPService) ResponseError(err error, writer http.ResponseWriter, StatusCode int) {
	server_error := &ServiceResult{Error: err.Error(), Status: false}
	json_str, _ := json.Marshal(server_error)
	writer.Header().Add("Content-Type", "application/json")

	http.Error(writer, string(json_str), StatusCode)
}

func (this *HTTPService) ResponseJSON(src interface{}, writer http.ResponseWriter) {
	serverResult := &ServiceResult{Data: src, Status: true}
	bin, _ := json.Marshal(serverResult)
	reader := bytes.NewReader(bin)

	writer.Header().Add("Content-Type", "application/json")

	io.Copy(writer, reader)
}
