Send http requests 

install deps
```
apt-get update
apt-get install -y wget iptables
```

Run docker container
```
docker run -it --privileged ubuntu:bi
```


fire http request in the background
```
while true; do echo "requesting..."; wget http://www.google.com; echo "done"; sleep 1; done &> /dev/null &
```

httpry to silent wget to docker host
```
httpry -q -F | awk -v BI_HOST=`/sbin/ip route|awk '/default/ { print $3 }'` -v BI_PORT=3839 -W interactive '{print "http://"BI_HOST":"BI_PORT"?date="$1"&time="$2"&src="$3"&dst="$4"&dir="$5"&method="$6"&host="$7"&path="$8"&code="$10"&status="$11 }' | xargs -L 1 wget -qO- &> /dev/null
```

run binocular container on boot2docker binding to default port
`docker run -p 3839:3839 -it -e DOCKER_HOST=tcp://`boot2docker ip`:2376 -e DOCKER_CERT_PATH=/cert -v /home/docker/.docker:/cert binocular`