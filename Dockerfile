FROM golang:1.13-alpine as builder

# Add Maintainer Info
LABEL maintainer="Sam Zhou <sam@mixmedia.com>"

# Set the Current Working Directory inside the container
WORKDIR /app/video-splitter

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go version \
 && export GO111MODULE=on \
 && export GOPROXY=https://goproxy.io \
 && go mod vendor \
 && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o video-splitter

######## Start a new stage from scratch #######
FROM alpine:latest  

RUN wget -O /usr/local/bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.2/dumb-init_1.2.2_amd64 \
 && chmod +x /usr/local/bin/dumb-init \
 && sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
 && apk add --update libintl \
 && apk add ffmpeg \
 && apk add --virtual build_deps gettext  \
 && cp /usr/bin/envsubst /usr/local/bin/envsubst \
 && apk del build_deps

WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/video-splitter/video-splitter .
COPY --from=builder /app/video-splitter/webroot ./webroot
COPY --from=builder /app/video-splitter/config.json .

ENV HOST=0.0.0.0:3335 \
 SERVICE_NAME=video-splitter \
 S3_APPKEY= \
 S3_SECRET= \
 S3_BUCKET=s3.test.mixmedia.com \
 S3_REGION=ap-southeast-1 \
 ROOT=/app/webroot \
 TEMP=/tmp \
 FFMPEG_BIN=/usr/bin/ffmpeg \
 FFPROBE_BIN=/usr/bin/ffprobe
 
EXPOSE 3335

ENTRYPOINT ["dumb-init", "--"]

CMD envsubst < /app/config.json > /app/temp.json \
 && /app/video-splitter -c /app/temp.json