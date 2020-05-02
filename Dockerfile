# Start from the latest golang base image
FROM golang:1.14.2-alpine3.11
RUN apk add git
ENV GOBIN /go/bin

COPY . /go/src/guacamole-library/
WORKDIR /go/src/guacamole-library/
# # Node exporter install and run
# RUN go get github.com/prometheus/node_exporter
# RUN cd $(GOPATH-HOME/go}/src/github.com/prometheus/node_exporter
# RUN make
# RUN ./node_exporter
# EXPOSE 9100

RUN go get
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cbp .

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
RUN mkdir /app
WORKDIR /app/
COPY --from=0 /go/src/guacamole-library/ .
EXPOSE 8888


CMD ["./cbp"]

