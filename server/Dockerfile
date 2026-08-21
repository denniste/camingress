# Go 中间层构建 (多阶段)
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o camingress .

FROM alpine:3.19
RUN apk add --no-cache ffmpeg
WORKDIR /app
COPY --from=builder /build/camingress .
EXPOSE 8080
ENTRYPOINT ["./camingress"]
