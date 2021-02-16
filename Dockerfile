FROM amd64/golang:1.14.9-stretch as builder

# Add Maintainer Info
LABEL maintainer="Sam Zhou <sam@mixmedia.com>"

# Set the Current Working Directory inside the container
WORKDIR /app/spin360

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go version \
 && export GO111MODULE=on \
 && go mod vendor \
 && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o spin360

######## Start a new stage from scratch #######
FROM bitnami/minideb:stretch

# UTF-8 Environment
ENV LC_ALL C.UTF-8

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates hugin gettext-base ffmpeg dumb-init \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/spin360/spin360 .
COPY --from=builder /app/spin360/webroot ./webroot
COPY --from=builder /app/spin360/config.json .

ENV HOST=0.0.0.0:3335 \
 SERVICE_NAME=spin360 \
 MAX_VIDEO_HEIGHT=720 \
 S3_APPKEY= \
 S3_SECRET= \
 S3_BUCKET=s3.test.mixmedia.com \
 OSS_BUCKET=oss-mixmedia-com \
 OSS_ENDPOINT=oss-cn-shenzhen.aliyuncs.com \
 OSS_APPKEY= \
 OSS_SECRET= \
 NONA_BIN=/usr/bin/nona \
 ROOT=/app/webroot \
 TEMP=/tmp \
 FFMPEG_BIN=/usr/bin/ffmpeg \
 FFPROBE_BIN=/usr/bin/ffprobe
 
EXPOSE 3335

ENTRYPOINT ["dumb-init", "--"]

CMD envsubst < /app/config.json > /app/temp.json \
 && /app/spin360 -c /app/temp.json
