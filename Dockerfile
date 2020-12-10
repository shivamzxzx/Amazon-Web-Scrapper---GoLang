FROM golang:1.15-alpine3.12
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN apk update
RUN apk add git
RUN go get github.com/gorilla/mux
RUN go get go.mongodb.org/mongo-driver/mongo
RUN go get github.com/gocolly/colly

