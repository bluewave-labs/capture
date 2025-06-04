# Use golang:1.23.1-alpine as the base image.
FROM golang:1.23.1-alpine AS builder
COPY . /app
# Change directory and build the binary. Build command is also used to download the dependencies.
RUN cd /app && go build -o capture ./cmd/capture

# Use chainguard/static:latest-glibc as the base image.
# This image is a minimal static image, which is suitable for running Go applications.
FROM chainguard/static:latest-glibc
COPY --from=builder /app/capture /usr/bin/

# Set the default GIN_MODE to release, so that the application runs in production mode. However, this can be overridden by setting the GIN_MODE environment variable.
# https://docs.docker.com/reference/dockerfile/#env
ENV GIN_MODE=release

# https://docs.docker.com/reference/dockerfile/#stopsignal
STOPSIGNAL SIGINT

# https://docs.docker.com/reference/dockerfile/#expose
EXPOSE 59232
CMD ["/usr/bin/capture"]
