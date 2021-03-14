FROM golang:1.16.2-alpine3.13
WORKDIR /build
COPY . /build
RUN rm -rf .git
RUN rm -rf .github
RUN rm README.md
RUN cd cmd/discordgateway-alert-bot && go build -o main . && cp main /build/main
CMD ["./main"]
