package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
)

type KVPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ServerHealth struct {
	Status   string `json:"status"`
	Requests int    `json:"requests"`
}

type Client struct {
	BackendConfig ConfigBackend
	Password      string
}

func InitClient(config ConfigBackend) *Client {
	password := os.Getenv(BaseENVname + "_BACKEND_PASSWORD")
	httpClient := &Client{BackendConfig: config, Password: password}
	return httpClient
}

func (c *Client) GetList() []KVPair {
	client := &http.Client{}
	req, err := http.NewRequest("GET", c.BackendConfig.Protocol+"://"+c.BackendConfig.Host+":"+c.BackendConfig.Port+"/system/fullList", nil)
	log.Printf("%+v %+v", c.BackendConfig.Username, c.Password)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	var list []KVPair
	err = json.NewDecoder(resp.Body).Decode(&list)
	if err != nil {
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		log.Printf("%+v %+v", resp, bodyText)
	} else {
		log.Printf("%+v", list)
	}
	return list
}
func (c *Client) GetHealth() bool {
	client := &http.Client{}
	req, err := http.NewRequest("GET", c.BackendConfig.Protocol+"://"+c.BackendConfig.Host+":"+c.BackendConfig.Port+"/system/health", nil)
	log.Printf("%+v %+v", c.BackendConfig.Username, c.Password)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	var health ServerHealth
	err = json.NewDecoder(resp.Body).Decode(&health)
	if err != nil {
		return false
	}
	return resp.StatusCode == 200 && health.Status == "UP"
}
