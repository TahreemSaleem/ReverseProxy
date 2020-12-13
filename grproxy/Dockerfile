FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git

RUN mkdir /app

ADD src/grproxy/ /app

WORKDIR /app

RUN go get -d -v

RUN CGO_ENABLED=0 GOOS=linux go build -o grproxy ./...

FROM scratch

COPY --from=builder /app .

EXPOSE $PORT


CMD ["./grproxy"]