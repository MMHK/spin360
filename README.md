# Spin360

snapshot from a video upload

具体实现的功能如下:

- 通过上传视频，将视频截图为N等份

外部依赖：
- [ffmepg](https://github.com/FFmpeg/FFmpeg), 视频截图依赖。

## 编译
----
- 安装Golang环境, Go >= 1.13
- checkout 源码
- 在源码目录 执行` go mod vendor `签出所有的依赖库
- ` go build -o spin360 . ` 编译成二进制可执行文件
- 执行文件 ` spin360 -c ./config.json`

## 配置文件
----
该项目使用json文件进行配置，具体例子如下

```JS
{
  "listen": "127.0.0.1:3335", //服务端口endpoint
  "web_root": "./webroot", //web 根目录
  "temp": "...", //临时目录
  "ffmpeg": {
    "ffmpeg": "...", //ffmepg 执行路径
    "ffprobe": ".." //ffprobe 执行路径
  },
  "s3": {
    "access_key": "", //s3 access key
    "secret_key": "", //s3 access secret
    "bucket": "", //s3 bucket
    "region": "", //s3 region
    "prefix": "/spin360"  //s3 save prefix path
  }
}
```

- `listen` 启动http service时绑定的地址
- `temp` 临时文件的保存路径，一般临时包括：上传图片的原图、待上传到Zurich的文件、待转换的HTML文件，
  这些文件一般会在使用后马上删除，不过也不排除程序问题没有删除的文件。
- `web_root` http service使用的webroot
- `s3` S3 相关信息
   - `access_key` S3 访问key
   - `secret_key` S3 访问秘钥
   - `bucket` S3 存储桶
   - `region`  S3 存储区域
   - `prefix` S3保存路径前缀


## 生成 `swagger` 文档

- 安装 [swagger-go](https://github.com/go-swagger/go-swagger)
- 在项目目录执行
```bash
swagger generate spec -o ./webroot/swagger/swagger.json
```

## Docker

此项目已经打包成docker 镜像

- 签出docker 镜像
```
docker pull mmhk/spin360
```
- 环境变量，具体请参考 `config.json` 的说明。
  - HOST，service绑定的服务地址及端口，默认为 `127.0.0.1:3335`
  - ROOT, swagger-ui 存放的本地目录，可以设置空来屏蔽 swagger-ui 的显示， 默认为 `/usr/local/mmhk/pgp-sftp-proxy/web_root`
  - FFMPEG_BIN, SSH远程访问host
  - FFPROBE_BIN, SSH远程登录账户
  - S3_APPKEY, SSH远程登录密码
  - S3_SECRET, SSH远程登录密匙，当sftp 使用密匙登录的时候使用，是一个本地文件路径。（注意是容器中的路径，应该使用 `-v`参数映射进容器）
  - S3_BUCKET, sftp 远程开发目录文件夹, 默认值：`/Interface_Development_Files/`
  - S3_REGION, sftp 远程产品目录文件夹, 默认值：`/Interface_Production_Files/`
- 运行
```
docker run --name spin360 -p 3335:3335 mmhk/spin360:latest
```
