FROM golang:alpine as builder
WORKDIR /go/src/app
COPY . .
RUN go build -o app .
FROM scratch
COPY --from=builder app /
CMD ["/app"]
