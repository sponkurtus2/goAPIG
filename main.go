package main

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type ConfigFile struct {
	APIs []string `yaml:"apis"`
}

func readConfigFile() []string {
	bytes, err := os.ReadFile("config_back.yaml")
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

func fetchApi(context context.Context, url string, respCode chan<- int) {

	req, err := http.NewRequestWithContext(context, "GET", url, nil)
	if err != nil {
		log.Fatalf("Error creating the request for url: %s, error: %s", url, err)
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error making the request for url: %s, error: %s", url, err)
	}
	defer resp.Body.Close()

	respCode <- resp.StatusCode
}

func checkIfWorking() {
	logger := zap.Must(zap.NewProduction())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	respCodes := make(chan int)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	var urls []string
	urls = readConfigFile()

	go func() {
		defer wg.Done()

		logger.Info("Making calls to the apis :)")

		for i := 0; i <= len(urls)-1; i++ {
			go fetchApi(ctx, urls[i], respCodes)
		}

		for i := 0; i <= len(urls)-1; i++ {
			if <-respCodes == 200 {
				logger.Info("Successfully fetched data ✅",
					zap.String("URL -> ", urls[i]))
			} else {
				logger.Info("Backend issue, Api not working 🚫",
					zap.String("URL -> ", urls[i]))
			}
		}

		logger.Info("Finished.")
	}()
	wg.Wait()
}

func main() {
	for {
		fmt.Print("\033[H\033[2J")
		checkIfWorking()
	}
}
