# Start from the latest golang base image
FROM golang:alpine
RUN apk add git

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN go get -d ./...

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cbp .

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl

RUN mkdir /app

WORKDIR /app/

COPY --from=0 /app .

EXPOSE 8888

# Command to run the executable
CMD ["./cbp"]
