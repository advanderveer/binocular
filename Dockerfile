FROM google/golang:1.3

#install httpry
RUN apt-get update
RUN echo 'deb http://http.debian.net/debian wheezy-backports main' >> /etc/apt/sources.list
RUN apt-get update
RUN apt-get install -y httpry

# install gostuff
RUN go get -d github.com/docker/docker/...
WORKDIR /gopath/src/app
ADD . /gopath/src/app/
RUN go get app

CMD []
EXPOSE 8000
ENTRYPOINT ["/gopath/bin/app"]