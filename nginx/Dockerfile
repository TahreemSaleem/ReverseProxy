FROM alpine:latest

RUN apk upgrade

RUN apk add nginx

RUN mkdir -p /run/nginx

COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80

CMD ["nginx"]

