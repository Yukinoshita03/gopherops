FROM golang:1.26.8-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/identity ./cmd/identity

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
COPY --from=build /out/identity /usr/local/bin/identity

USER 65532:65532
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/identity"]
