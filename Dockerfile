FROM golang:alpine as builder
WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/app .
FROM scratch
COPY --from=builder /go/bin/app /app
EXPOSE 80
CMD ["/app"]
