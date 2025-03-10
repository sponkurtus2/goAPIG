package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

type EndpointWorkingOrNot struct {
	ID         int
	URL        string
	StatusCode int
}

type PageDataWorkingOrNot struct {
	EndpointsWorkingOrN []EndpointWorkingOrNot
}

type Endpoint struct {
	ID   string
	Name string
	URL  string
	Data string
}

type PageData struct {
	Endpoints []Endpoint
}

type ConfigFile struct {
	APIs []string `yaml:"apis"`
}

func readConfigFile() []string {
	bytes, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Error reading config.yaml -> %s", err)
	}

	var config ConfigFile
	if err := yaml.Unmarshal(bytes, &config); err != nil {
		log.Fatalf("Error when Unmarshal -> %s", err)
	}

	var apis []string
	for _, api := range config.APIs {
		apis = append(apis, api)
	}
	return apis
}

func checkIfWorking(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	var WorkingEP []EndpointWorkingOrNot
	var mu sync.Mutex

	var urls []string
	urls = readConfigFile()

	for i, url := range urls {
		wg.Add(1)

		go func(url string, index int) {
			defer wg.Done()

			response, err := http.Get(url)
			if err != nil {
				log.Printf("Error when obtaining response -> %s", err)
				return
			}
			defer response.Body.Close()

			statusCode := response.StatusCode

			endPointWorkingOrN := EndpointWorkingOrNot{
				ID:         i,
				URL:        url,
				StatusCode: statusCode,
			}
			mu.Lock()
			WorkingEP = append(WorkingEP, endPointWorkingOrN)
			mu.Unlock()
		}(url, i)
	}
	wg.Wait()
	tmpl := template.Must(template.ParseFiles("templates/working.html"))
	pageDataWorkingOrNot := PageDataWorkingOrNot{
		EndpointsWorkingOrN: WorkingEP,
	}
	tmpl.Execute(w, pageDataWorkingOrNot)
}

func checkData(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	var endpoints []Endpoint
	var mu sync.Mutex

	var urls []string
	urls = readConfigFile()

	for i, url := range urls {
		wg.Add(1)
		go func(url string, index int) {
			defer wg.Done()

			response, err := http.Get(url)
			if err != nil {
				return
			}
			defer response.Body.Close()

			responseData, err := io.ReadAll(response.Body)
			if err != nil {
				return
			}

			var prettyJSON []byte
			if json.Valid(responseData) {
				var parsedJSON interface{}
				if err := json.Unmarshal(responseData, &parsedJSON); err == nil {
					prettyJSON, _ = json.MarshalIndent(parsedJSON, "", "    ")
				}
			}

			endpoint := Endpoint{
				ID:   fmt.Sprintf("endpoint-%d", index),
				Name: getEndpointName(url),
				URL:  url,
				Data: string(prettyJSON),
			}

			mu.Lock()
			endpoints = append(endpoints, endpoint)
			mu.Unlock()
		}(url, i)
	}

	wg.Wait()

	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	pageData := PageData{
		Endpoints: endpoints,
	}

	tmpl.Execute(w, pageData)
}

func getEndpointName(url string) string {
	parts := strings.Split(url, "/api/")
	if len(parts) > 1 {
		return strings.Split(parts[1], "?")[0]
	}
	return url
}

func main() {
	http.HandleFunc("/", checkIfWorking)
	http.HandleFunc("/get", checkData)

	fmt.Println("Server starting on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
