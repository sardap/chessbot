FROM golang:latest

WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o main .

RUN rm -rf assets
RUN mkdir assets
COPY ./assets/*.png ./assets/

ENTRYPOINT [ "/app/main" ]