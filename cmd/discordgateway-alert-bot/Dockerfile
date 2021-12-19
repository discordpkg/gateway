FROM golang:1.16.2-alpine3.13
WORKDIR /build
COPY . /build
RUN cd cmd/discordgateway-alert-bot && go build -o main . && cp main /build/main
CMD ["./main"]
