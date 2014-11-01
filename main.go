package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/fsouza/go-dockerclient"
	"github.com/nu7hatch/gouuid"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

type Docker struct {
	client *docker.Client
}

type Log struct {
	Time   time.Time
	From   string //src container id
	To     string //dst container id
	Src    string
	Dst    string
	Way    string
	Method string
	Host   string
	Path   string
	Code   int
	Status string
}

type Node struct {
	PortMap map[int64]docker.APIContainers
	IpMap   map[string]docker.APIContainers
}

var iface = flag.String("i", "docker0", "the docker network interface.")

var tlogs = []byte("logs")

func main() {

	//get cwd
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// open bolt db
	db, err := bolt.Open(filepath.Join(cwd, "binocular.db"), 0644, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//parse flags
	flag.Parse()

	//get config from env
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		log.Fatal(fmt.Errorf("Could not retrieve DOCKER_HOST, not provided as option and not in env"))
	}

	cpath := os.Getenv("DOCKER_CERT_PATH")
	if cpath == "" {
		log.Fatal(fmt.Errorf("Could not retrieve DOCKER_CERT_PATH, not provided as option and not in env"))
	}

	//change to http connection
	addr, err := url.Parse(host)
	if err != nil {
		log.Fatal(err)
	}

	addr.Scheme = "https"

	//setup tls docker client
	client, err := docker.NewTLSClient(addr.String(), filepath.Join(cpath, "cert.pem"), filepath.Join(cpath, "key.pem"), filepath.Join(cpath, "ca.pem"))
	if err != nil {
		log.Fatal(err)
	}

	// get all running containers
	lopts := docker.ListContainersOptions{}
	cs, err := client.ListContainers(lopts)
	if err != nil {
		log.Fatal(err)
	}

	//host information
	node := &Node{
		PortMap: map[int64]docker.APIContainers{},
		IpMap:   map[string]docker.APIContainers{},
	}

	//create httpry command
	var parts []string
	for _, c := range cs {

		//fetch public ports
		for _, p := range c.Ports {
			parts = append(parts, fmt.Sprintf("port %d", p.PublicPort))

			//map public ports to container
			node.PortMap[p.PublicPort] = c
		}

		details, err := client.InspectContainer(c.ID)
		if err != nil {
			log.Fatal(err)
		}

		//map ip to container
		node.IpMap[details.NetworkSettings.IPAddress] = c
	}

	//create expression and httpry command
	exp := strings.Join(parts, " or ")
	log.Printf("Starting httpry on '%s' with exp '%s'...", *iface, exp)
	cmd := exec.Command("httpry", "-q", "-F", "-i", *iface, exp)
	cmd.Stderr = os.Stderr

	//pipe stdout for analyses
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	//Handle httpry and save to db
	err = cmd.Start()
	if err == nil {

		//scan httpry output
		scanner := bufio.NewScanner(stdout)
		go func() {
			for scanner.Scan() {

				//split line by whitespace
				fields := strings.Fields(scanner.Text())

				//parse using layou Mon Jan 2 15:04:05 MST 2006
				t, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s %s", fields[0], fields[1]))
				if err != nil {
					t = time.Time{}
				}

				c, err := strconv.Atoi(fields[9])
				if err != nil {
					c = -1
				}

				//create log
				l := &Log{
					Time:   t,
					Src:    fields[2],
					Dst:    fields[3],
					Way:    fields[4],
					Method: fields[5],
					Host:   fields[6],
					Path:   fields[7],
					Code:   c,
					Status: fields[10],
				}

				//since requests cannot be associated to responses only log requests
				if l.Way == "<" {
					continue
				}

				var to docker.APIContainers
				var from docker.APIContainers
				var ok bool

				//get To by hostname port
				dstport := int64(80)
				dstparts := strings.SplitN(l.Host, ":", 2)
				if len(dstparts) > 1 {
					i, err := strconv.Atoi(dstparts[1])
					if err != nil {
						log.Println("Error:", err)
					}

					dstport = int64(i)
				}

				if to, ok = node.PortMap[dstport]; !ok {
					log.Printf("Error: Failed to find container from portmap, host: '%s', port: '%d'", l.Host, dstport)
				} else {
					l.To = to.ID
				}

				// From by src ip
				if from, ok = node.IpMap[l.Src]; !ok {
					log.Printf("Error: Failed to find container from ipmap, src: '%s', map: '%d'", l.Src, node.IpMap)
				} else {
					l.From = from.ID
				}

				//serialize
				json, err := json.Marshal(l)
				if err != nil {
					log.Println("Error:", err)
				}

				// write to db
				err = db.Update(func(tx *bolt.Tx) error {
					b, err := tx.CreateBucketIfNotExists(tlogs)
					if err != nil {
						return err
					}

					//generate uid
					uid, err := uuid.NewV4()
					if err != nil {
						return err
					}

					//use current nano as a key @todo come up with something more ressilient
					err = b.Put([]byte(uid.String()), json)
					if err != nil {
						return err
					}

					return nil
				})

				fmt.Printf("%s - %s %s -> %s\n", l.From, l.Method, l.Path, l.To)
			}
		}()

	}

	//serve logs
	goji.Get("/logs", func(c web.C, w http.ResponseWriter, r *http.Request) {

		logs := []*Log{}
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(tlogs)
			if b == nil {
				return err
			}

			b.ForEach(func(k, v []byte) error {
				l := &Log{}
				err := json.Unmarshal(v, l)
				if err != nil {
					return err
				}

				logs = append(logs, l)
				return nil
			})

			return nil
		})

		enc := json.NewEncoder(w)
		err = enc.Encode(logs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	//otherwise server static file
	goji.Use(func(c *web.C, h http.Handler) http.Handler {

		dir := filepath.Join(cwd, "client")
		fs := http.FileServer(http.Dir(dir))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			fname := filepath.Join(dir, r.URL.Path)

			_, err := os.Stat(fname)
			if err != nil {
				h.ServeHTTP(w, r)
				return
			}

			fs.ServeHTTP(w, r)
		})
	})

	goji.Serve()

}
