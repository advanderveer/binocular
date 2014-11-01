Build and run on a host
`docker build -t binocular . ; docker run --net=host -it -e DOCKER_HOST=tcp://`boot2docker ip`:2376 -e DOCKER_CERT_PATH=/cert -v /home/docker/.docker:/cert binocular`

Or just the container:
`docker run --net=host -it -e DOCKER_HOST=tcp://`boot2docker ip`:2376 -e DOCKER_CERT_PATH=/cert -v /home/docker/.docker:/cert binocular`