package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/samuel/go-zookeeper/zk"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var conn *zk.Conn

type endpoint struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

func main() {
	//connecting to zookeeper
	var err error
	conn, _, err = zk.Connect([]string{"zookeeper"}, time.Second) //*10)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to Zookeeper")
	entityData, _ := json.Marshal(endpoint{Host: os.Getenv("HOSTNAME"), Port: os.Getenv("PORT")})

	_, err = conn.Create("/server", []byte{}, 0, zk.WorldACL(zk.PermAll))
	_, err = conn.Create("/server/"+os.Getenv("HOSTNAME"), entityData, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", handler)
	fmt.Println("Starting server at port:" + os.Getenv("PORT"))
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":

		tpl := template.Must(template.New("index.html").Funcs(sprig.FuncMap()).ParseGlob("index.html"))

		err := tpl.Execute(w, retrieveHBase())
		if err != nil {
			panic(err)
		}

	case "POST":
		unencodedJSON, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "500 something went wrong", http.StatusInternalServerError)
			log.Fatalln(err)
		}
		encodedJSON, unencodedRows, err := parseJSON(unencodedJSON)
		if err != nil {
			http.Error(w, "400 Bad Request", http.StatusBadRequest)
			log.Fatalln(err)
		}
		if len(unencodedRows.Row) == 0 {
			http.Error(w, "200 OK", http.StatusOK)
			return
		}
		if zk.StateConnected == conn.State() || zk.StateHasSession == conn.State() {
			err = updateHBase(encodedJSON, unencodedRows)
			if err != nil {
				http.Error(w, "500 something went wrong", http.StatusInternalServerError)
				log.Fatalln(err)

			}
		}

	default:
		_, _ = fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

func parseJSON(unencodedJSON []byte) (encodedJSON []byte, unencodedRows RowsType, err error) {


	err = json.Unmarshal(unencodedJSON, &unencodedRows)
	// encode fields in Go objects
	encodedRows := unencodedRows.encode()
	// convert encoded Go objects to JSON
	encodedJSON, err = json.Marshal(encodedRows)

	return encodedJSON, unencodedRows, err
}
func retrieveHBase() RowsType {
	client := &http.Client{}

	req, err := http.NewRequest("PUT", "http://hbase:8080/se2:library/scanner/", bytes.NewBuffer([]byte(`<Scanner batch="10"> </Scanner>`)))

	req.Header.Set("Accept", "text/plain")
	req.Header.Set("Content-Type", "text/xml")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	Scanner, _ := resp.Location()

	req, err = http.NewRequest("GET", Scanner.String(), nil)
	req.Header.Set("Accept", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var encodedRows EncRowsType
	err = json.Unmarshal(bodyBytes, &encodedRows)
	decodedRows, _ := encodedRows.decode()

	//delete scanner
	_, err = http.NewRequest(http.MethodDelete, Scanner.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	return decodedRows

}
func updateHBase(encodedJSON []byte, unencodedRows RowsType) error {

	// initialize http client
	client := &http.Client{}

	key := unencodedRows.Row[0].Key
	req, err := http.NewRequest(http.MethodPut, "http://hbase:8080/se2:library/"+key, bytes.NewBuffer(encodedJSON))

	if err != nil {
		return err
	}
	defer req.Body.Close()

	// set the request header Content-Type for json
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)

	if resp.StatusCode == 200 {
		return nil
	}
	return err

}
