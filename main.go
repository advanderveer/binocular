package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/fsouza/go-dockerclient"
)

type Docker struct {
	client *docker.Client
}

func NewDocker(addr string, cert string) (*Docker, error) {
	host, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	//change to http connection
	host.Scheme = "https"

	c, err := docker.NewClient(host.String())
	if err != nil {
		return nil, err
	}

	//we use our own transform and client to support boot2docker tls requirements
	//@see https://github.com/boot2docker/boot2docker/issues/576
	//@see http://stackoverflow.com/questions/21562269/golang-how-to-specify-certificate-in-tls-config-for-http-client
	cas := x509.NewCertPool()
	pemData, err := ioutil.ReadFile(filepath.Join(cert, "ca.pem"))
	if err != nil {
		return nil, err
	}

	//add to pool and configrue tls
	cas.AppendCertsFromPEM(pemData)

	//load pair
	pair, err := tls.LoadX509KeyPair(filepath.Join(cert, "cert.pem"), filepath.Join(cert, "key.pem"))
	if err != nil {
		return nil, err
	}

	//create new tls config with the created ca and pair
	conf := &tls.Config{
		RootCAs:      cas,
		Certificates: []tls.Certificate{pair},
	}

	//create our own transport
	tr := &http.Transport{
		TLSClientConfig: conf,
	}

	//set docker client with new transport
	c.HTTPClient = &http.Client{Transport: tr}

	return &Docker{c}, nil
}

var bind = flag.String("bind", ":3839", "the port on which the http server will bind")

func main() {

	//parse flags
	flag.Parse()

	//probably 172.17.42.1
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		log.Fatal(fmt.Errorf("Could not retrieve DOCKER_HOST, not provided as option and not in env"))
	}

	cert := os.Getenv("DOCKER_CERT_PATH")
	if cert == "" {
		log.Fatal(fmt.Errorf("Could not retrieve DOCKER_CERT_PATH, not provided as option and not in env"))
	}

	dock, err := NewDocker(host, cert)
	if err != nil {
		log.Fatal(err)
	}

	// get all running containers
	lopts := docker.ListContainersOptions{}
	cs, err := dock.client.ListContainers(lopts)
	if err != nil {
		log.Fatal(err)
	}

	//inject monitoring into each container
	for _, c := range cs {

		//setup
		copts := docker.CreateExecOptions{
			Container: c.ID,
			Cmd:       []string{"touch", "/tmp/x"},
		}

		execObj, err := dock.client.CreateExec(copts)
		if err != nil {
			log.Fatal(err)
		}

		//start
		sopts := docker.StartExecOptions{
			Detach: true,
		}

		err = dock.client.StartExec(execObj.Id, sopts)
		if err != nil {
			log.Fatal(err)
		}

		//progress?
		fmt.Println(c.ID, execObj.Id)
	}

	//start a web server that logs incoming requests to a file
	log.Printf("Listening for incoming on %s...", *bind)
	err = http.ListenAndServe(*bind, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World from binocular")

		//also print to console
		log.Println(fmt.Sprint(r.URL))
		log.Println(fmt.Sprint(r.Header))

	}))

	if err != nil {
		log.Fatal(err)
	}

}
