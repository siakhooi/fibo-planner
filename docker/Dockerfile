FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY app ./app
ENV CGO_ENABLED=0
RUN go build -C app -trimpath -ldflags="-s -w" -o /server

FROM scratch
COPY --from=builder --chown=65532:65532 /server /server
USER 65532:65532
EXPOSE 8080
CMD ["/server"]
