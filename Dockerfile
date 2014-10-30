FROM google/golang:1.3

#we usegodep
RUN go get github.com/tools/godep
RUN mkdir -p /gopath/src/github.com/dockpit/binocular

WORKDIR /gopath/src/github.com/dockpit/binocular
ADD . /gopath/src/github.com/dockpit/binocular
RUN godep go build -o /gopath/bin/binocular

CMD []
EXPOSE 8000
ENTRYPOINT ["/gopath/bin/binocular"]