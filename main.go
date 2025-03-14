package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

//go:embed templates/*
var content embed.FS

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

func getConfigFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error obtaining home directory -> %s", err)
	}

	configDir := filepath.Join(homeDir, ".goApig")
	configFilePath := filepath.Join(configDir, "config.yaml")
	return configFilePath
}

func createDefaultConfigFile() {
	configFilePath := getConfigFilePath()

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		configDir := filepath.Dir(configFilePath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			log.Fatalf("Error creating directory -> %s", err)
		}

		defaultConfig := ConfigFile{
			APIs: []string{
				"https://github.com/sponkurtus2/goAPIG",
				"https://jsonplaceholder.typicode.com/posts",
			},
		}

		if err := generateYAMLFile(configFilePath, defaultConfig); err != nil {
			log.Fatalf("Error creating config file -> %s", err)
		}

		log.Printf("Config file created at -> %s", configFilePath)
	}
}

func generateYAMLFile(filePath string, data interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling data to YAML: %v", err)
	}

	if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
		return fmt.Errorf("error writing YAML file: %v", err)
	}

	return nil
}

func readConfigFile() []string {
	configFilePath := getConfigFilePath()

	bytes, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Fatalf("Error reading config file -> %s", err)
	}

	var config ConfigFile
	if err := yaml.Unmarshal(bytes, &config); err != nil {
		log.Fatalf("Error unmarshaling YAML -> %s", err)
	}

	return config.APIs
}

func checkIfWorking(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	var WorkingEP []EndpointWorkingOrNot
	var mu sync.Mutex

	urls := readConfigFile()

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
				ID:         index,
				URL:        url,
				StatusCode: statusCode,
			}
			mu.Lock()
			WorkingEP = append(WorkingEP, endPointWorkingOrN)
			mu.Unlock()
		}(url, i)
	}
	wg.Wait()

	tmpl := template.Must(template.ParseFS(content, "templates/working.html"))
	pageDataWorkingOrNot := PageDataWorkingOrNot{
		EndpointsWorkingOrN: WorkingEP,
	}
	tmpl.Execute(w, pageDataWorkingOrNot)
}

func checkData(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	var endpoints []Endpoint
	var mu sync.Mutex

	urls := readConfigFile()

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

	tmpl := template.Must(template.ParseFS(content, "templates/index.html"))
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
	createDefaultConfigFile()

	http.HandleFunc("/", checkIfWorking)
	http.HandleFunc("/get", checkData)

	fmt.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Error starting server -> %s", err)
	}
}
