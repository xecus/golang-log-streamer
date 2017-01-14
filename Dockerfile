FROM golang

RUN mkdir -p /go/src/app
WORKDIR /go/src/app
COPY . /go/src/app

RUN curl https://glide.sh/get | sh
RUN glide install -v

EXPOSE 8000
CMD go run main.go
