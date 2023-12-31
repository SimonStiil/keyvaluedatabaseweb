package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
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
	TLSCpnfig     *tls.Config
}

func InitClient(config ConfigBackend) *Client {
	files, err := os.ReadDir(config.CertDir)
	rootCAs := x509.NewCertPool()
	if err != nil {
		log.Printf("E %v", err)
	}
	for _, file := range files {
		fileName := config.CertDir + string(os.PathSeparator) + file.Name()
		certs, err := os.ReadFile(fileName)
		if err != nil {
			log.Printf("E Failed to append %q to RootCAs: %v", fileName, err)
		}
		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			log.Printf("I %v certs not appended", file.Name())
		}
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.insecure,
		RootCAs:            rootCAs}
	password := os.Getenv(BaseENVname + "_BACKEND_PASSWORD")
	httpClient := &Client{BackendConfig: config, Password: password, TLSCpnfig: tlsConfig}
	return httpClient
}

func (c *Client) GetList() []rest.KVPairV1 {
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
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
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
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
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
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
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
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
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
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
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
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
