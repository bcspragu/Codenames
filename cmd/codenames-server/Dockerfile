FROM alpine
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
WORKDIR /app
ADD ./server /app
ADD ./config.example /app/config
CMD ["./server"]
