FROM golang:1.15.6-alpine3.12
WORKDIR /build
COPY . /build
RUN cd cmd/discordgateway-alert-bot && go build -o main . && cp main /build/main
CMD ["./main"]