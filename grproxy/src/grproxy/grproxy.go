package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"time"
)

type endpoint struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

const URL_NGINX = "http://nginx:80"

var conn *zk.Conn
var previousServer string

func main() {
	var err error
	conn, _, err = zk.Connect([]string{"zookeeper"}, time.Second) //*10)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to Zookeeper")

	http.HandleFunc("/", handler)

	err = http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		panic(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {

	var prefix = regexp.MustCompile(`^\/library`)
	var nginx *url.URL
	var err error

	if prefix.MatchString(r.URL.Path) {
		URL, err := zookeeperSD()
		if err != nil {
			panic(err)
		}
		nginx, err = url.Parse(URL)
		if err != nil {
			panic(err)
		}
	} else {
		nginx, err = url.Parse(URL_NGINX)
		if err != nil {
			panic(err)
		}
	}
	proxy := httputil.NewSingleHostReverseProxy(nginx)
	//log.Println(r.URL)
	proxy.ServeHTTP(w, r)
}

func zookeeperSD() (string, error) {
	var currentServerURL = ""
	if !(zk.StateConnected == conn.State() || zk.StateHasSession == conn.State()) {
		return currentServerURL, errors.New("zookeeper disconnected")
	}
	children, _, _, err := conn.ChildrenW("/server")
	if err != nil {
		panic(err)
	}
	for _, child := range children {
		data, _, _ := conn.Get("/server/" + child)
		var e endpoint
		_ = json.Unmarshal(data, &e)

		if len(children) > 1 && e.Host != previousServer {
			currentServerURL = "http://" + e.Host + ":" + e.Port
			previousServer = e.Host
			break
		} else if len(children) == 1 {
			currentServerURL = "http://" + e.Host + ":" + e.Port
			previousServer = e.Host
		}
	}
	return currentServerURL, nil
}
