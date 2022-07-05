# CGO_ENABLED=0 GO111MODULE=on go build -a -o ./bin/app main.go
FROM wutongpaas/alpine:3.15
RUN mkdir /app \
    && apk add --update apache2-utils \
    && rm -rf /var/cache/apk/*
ENV TZ=Asia/Shanghai
WORKDIR /
COPY ./bin/app ./manager

CMD ["./manager"]