FROM golang:latest AS builder
RUN mkdir /app 
ADD . /app/ 
WORKDIR /app 
RUN go build -o prometheus-instrumentation .

FROM alpine:latest
RUN apk add libc6-compat
RUN mkdir /app
COPY --from=builder /app/prometheus-instrumentation /app/prometheus-instrumentation 
ENTRYPOINT ["/app/prometheus-instrumentation"]

#docker build -t prometheus-instrumentation . --platform linux/amd64
