FROM golang:1.15.6-alpine3.12
WORKDIR /build
COPY . /build
RUN rm -rf .git
RUN rm -rf .github
RUN rm README.md
RUN cd cmd/discordgateway-alert-bot && go build -o main . && cp main /build/main
CMD ["./main"]
