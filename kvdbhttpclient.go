package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/SimonStiil/keyvaluedatabase/rest"
)

type Client struct {
	BackendConfig ConfigBackend
	Password      string
}

func InitClient(config ConfigBackend) *Client {
	password := os.Getenv(BaseENVname + "_BACKEND_PASSWORD")
	httpClient := &Client{BackendConfig: config, Password: password}
	return httpClient
}

func (c *Client) GetList() []rest.KVPairV1 {
	client := &http.Client{}
	req, err := http.NewRequest("GET", c.BackendConfig.Protocol+"://"+c.BackendConfig.Host+":"+c.BackendConfig.Port+"/system/fullList", nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	var list []rest.KVPairV1
	err = json.NewDecoder(resp.Body).Decode(&list)
	if err != nil {
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		log.Printf("%+v %+v", resp, bodyText)
	}
	return list
}
func (c *Client) Set(pair rest.KVPairV1) bool {
	client := &http.Client{}
	marshalled, err := json.Marshal(pair)
	if err != nil {
		log.Printf("E impossible to marshall pair: %s", err)
		return false
	}
	req, err := http.NewRequest("POST", c.BackendConfig.Protocol+"://"+c.BackendConfig.Host+":"+c.BackendConfig.Port+"/", bytes.NewReader(marshalled))
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("E Client Set call failed: %s", err)
	}
	bodyText, err := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusCreated && strings.TrimSpace(string(bodyText)) == "OK" {
		return true
	}
	log.Printf("%+v %+v", resp.StatusCode, bodyText)
	return false
}
func (c *Client) Delete(key string) bool {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", c.BackendConfig.Protocol+"://"+c.BackendConfig.Host+":"+c.BackendConfig.Port+"/"+key, nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("E Client Delete call failed: %s", err)
	}
	bodyText, err := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusCreated && strings.TrimSpace(string(bodyText)) == "OK" {
		return true
	}
	log.Printf("%+v %+v", resp.StatusCode, bodyText)
	return false
}
func (c *Client) Roll(key string) bool {
	client := &http.Client{}
	method := "{\"type\": \"roll\"}"
	req, err := http.NewRequest("UPDATE", c.BackendConfig.Protocol+"://"+c.BackendConfig.Host+":"+c.BackendConfig.Port+"/"+key, bytes.NewReader([]byte(method)))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("E Client Update(roll) call failed: %s", err)
	}
	var pair rest.KVPairV1
	err = json.NewDecoder(resp.Body).Decode(&pair)
	if err != nil {
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("E Client Update(roll) call failed: %s", err)
		}
		log.Printf("E Client Update(roll) call failed: %s", bodyText)
		return false
	}
	if resp.StatusCode == http.StatusOK && key == pair.Key {
		return true
	} else {
		log.Printf("%+v %+v", resp.StatusCode, pair)
	}
	return false
}
func (c *Client) Generate(key string) bool {
	client := &http.Client{}
	method := "{\"type\": \"generate\"}"
	req, err := http.NewRequest("UPDATE", c.BackendConfig.Protocol+"://"+c.BackendConfig.Host+":"+c.BackendConfig.Port+"/"+key, bytes.NewReader([]byte(method)))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("E Client Update(generate) call failed: %s", err)
	}
	var pair rest.KVPairV1
	err = json.NewDecoder(resp.Body).Decode(&pair)
	if err != nil {
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("E Client Update(generate) call failed: %s", err)
		}
		log.Printf("E Client Update(generate) call failed: %s", bodyText)
		return false
	}
	if resp.StatusCode == http.StatusOK && key == pair.Key {
		return true
	} else {
		log.Printf("I %+v %+v", resp.StatusCode, pair)
	}
	return false
}
func (c *Client) GetHealth() bool {
	client := &http.Client{}
	req, err := http.NewRequest("GET", c.BackendConfig.Protocol+"://"+c.BackendConfig.Host+":"+c.BackendConfig.Port+"/system/health", nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	var health rest.HealthV1
	err = json.NewDecoder(resp.Body).Decode(&health)
	if err != nil {
		return false
	}
	return resp.StatusCode == 200 && health.Status == "UP"
}
