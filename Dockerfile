FROM golang:alpine as builder

ENV GO111MODULE=on
RUN apk --no-cache add ca-certificates git


WORKDIR /build

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . /build/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main .

FROM alpine
WORKDIR /app

RUN mkdir .gofish
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/main /app/
COPY --from=builder /build/config /app/config
CMD ["./main"]