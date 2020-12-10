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
	//port := "9090"

	//connecting to zookeeper
	var err error
	conn, _, err = zk.Connect([]string{"zookeeper"}, time.Millisecond) //*10)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to Zookeeper")
	entityData, _ := json.Marshal(endpoint{Host: os.Getenv("HOSTNAME"), Port: os.Getenv("PORT")})
	_, err = conn.Create("/", entityData, zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", handler)
	//http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("http/css"))))

	fmt.Printf("Starting server at port 9090\n")
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
	//if err := http.ListenAndServe(":9090", nil); err != nil {
	//	log.Fatal(err)
	//}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":

		tpl := template.Must(template.New("index.html").Funcs(sprig.FuncMap()).ParseGlob("index.html"))

		err := tpl.Execute(w, retrieveHBase())
		if err != nil {
			panic(err)
		}
	// add response
	case "POST":
		unencodedJSON, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "500 something went wrong", http.StatusInternalServerError)
			log.Fatalln(err)
		}

		if zk.StateConnected == conn.State() || zk.StateHasSession == conn.State() {
			encodedJSON, unencodedRows, err := parseJSON(unencodedJSON)
			if err != nil {
				http.Error(w, "400 Bad Request", http.StatusBadRequest)
				log.Fatalln(err)
			}

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

	// unencoded JSON bytes from landing page
	// note: quotation marks need to be escaped with backslashes within Go strings: " -> \"
	//unencodedJSON := []byte("{\"Row\":[{\"key\":\"My first document\",\"Cell\":[{\"column\":\"document:Chapter 1\",\"$\":\"value:Once upon a time...\"},{\"column\":\"metadata:Author\",\"$\":\"value:The incredible me!\"}]}]}")
	// convert JSON to Go objects

	err = json.Unmarshal(unencodedJSON, &unencodedRows)
	// encode fields in Go objects
	encodedRows := unencodedRows.encode()
	// convert encoded Go objects to JSON
	encodedJSON, err = json.Marshal(encodedRows)

	//println("unencoded:", string(unencodedJSON))
	//println("encoded:", string(encodedJSON))

	return encodedJSON, unencodedRows, err
}
func retrieveHBase()RowsType{
	client := &http.Client{}

	//body := strings.NewReader(`<Scanner batch="10"/>`)
	req, err := http.NewRequest("PUT", "http://hbase:8080/se2:library/scanner/", bytes.NewBuffer([]byte(`<Scanner batch="10"> </Scanner>`)))

	req.Header.Set("Accept", "text/plain")
	req.Header.Set("Content-Type", "text/xml")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	Scanner, _ := resp.Location()
	fmt.Println(Scanner)
	req, err = http.NewRequest("GET",Scanner.String(), nil)
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
	decodedRows ,_ := encodedRows.decode()


	//delete scanner
	_, err = http.NewRequest(http.MethodDelete,Scanner.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	return decodedRows

}
func updateHBase(encodedJSON []byte, unencodedRows RowsType) error {

	/*
		output:

		unencoded: {"Row":[{"key":"Myfirstdocument","Cell":[{"column":"document:Chapter 1","$":"value:Once upon a time..."},{"column":"metadata:Author","$":"value:The incredible me!"}]}]}
		encoded: {"Row":[{"key":"TXkgZmlyc3QgZG9jdW1lbnQ=","Cell":[{"column":"ZG9jdW1lbnQ6Q2hhcHRlciAx","$":"dmFsdWU6T25jZSB1cG9uIGEgdGltZS4uLg=="},{"column":"bWV0YWRhdGE6QXV0aG9y","$":"dmFsdWU6VGhlIGluY3JlZGlibGUgbWUh"}]}]}
	*/

	// initialize http client
	client := &http.Client{}

	//resp, err := http.Post("","application/json", bytes.NewBuffer(requestBody))
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
