FROM golang:1.26 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/migrate ./cmd/migrate

FROM alpine:3.20
# ffmpeg and ffprobe are used by the video processing job.
RUN apk add --no-cache ffmpeg ca-certificates
COPY --from=build /bin/api /bin/api
# Applies database migrations; deploys run it before starting the new API.
COPY --from=build /bin/migrate /bin/migrate
EXPOSE 8080
# CMD rather than ENTRYPOINT, so a deploy can run /bin/migrate in its place.
CMD ["/bin/api"]