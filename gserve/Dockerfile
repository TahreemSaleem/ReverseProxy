FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git

RUN mkdir /app

ADD src/gserve/ /app
ADD index.html /app
WORKDIR /app

RUN go get -d -v

RUN CGO_ENABLED=0 GOOS=linux go build -o gserve ./...

FROM scratch

COPY --from=builder /app .

EXPOSE $PORT


CMD ["./gserve"]