FROM golang:1.13-alpine as builder

# Add Maintainer Info
LABEL maintainer="Sam Zhou <sam@mixmedia.com>"

# Set the Current Working Directory inside the container
WORKDIR /app/spin360

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go version \
 && export GO111MODULE=on \
 && export GOPROXY=https://goproxy.io \
 && go mod vendor \
 && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o spin360

######## Start a new stage from scratch #######
FROM phusion/baseimage:18.04-1.0.0-amd64

# UTF-8 Environment
ENV LC_ALL C.UTF-8

RUN curl https://github.com/Yelp/dumb-init/releases/download/v1.2.2/dumb-init_1.2.2_amd64 --output /usr/local/bin/dumb-init \
 && chmod +x /usr/local/bin/dumb-init \
 && apt-get update \
 && apt-get install -y --no-install-recommends hugin gettext-base ffmpeg \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/spin360/spin360 .
COPY --from=builder /app/spin360/webroot ./webroot
COPY --from=builder /app/spin360/config.json .

ENV HOST=0.0.0.0:3335 \
 SERVICE_NAME=spin360 \
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
 && /app/spin360 -c /app/temp.json
