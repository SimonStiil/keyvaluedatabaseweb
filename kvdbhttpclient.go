package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
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

type HTTPStatusError struct {
	StatusCode int
	Status     string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("%v %v", e.StatusCode, e.Status)
}

func (c *Client) GetNamespaceList() ([]rest.NamespaceV2, error) {
	logger.Debug("Get Namespace List", "function", "GetNamespaceList", "struct", "Client")
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", fmt.Sprintf("%v://%v:%v/v1/*", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port), nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK || resp.StatusCode != http.StatusCreated {
		return nil, &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return nil, err
	}
	var list []rest.NamespaceV2
	err = json.NewDecoder(resp.Body).Decode(&list)
	if err != nil {
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Printf("%+v %+v", resp, bodyText)
	}
	return list, nil
}

func (c *Client) GetKeyList(namespace string) ([]rest.KVPairV2, error) {
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", fmt.Sprintf("%v://%v:%v/v1/%v/*", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace), nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK || resp.StatusCode != http.StatusCreated {
		return nil, &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return nil, err
	}
	var list []rest.KVPairV2
	err = json.NewDecoder(resp.Body).Decode(&list)
	if err != nil {
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Printf("%+v %+v", resp, bodyText)
	}
	return list, nil
}
func (c *Client) SetKey(namespace string, pair rest.KVPairV2) error {
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	marshalled, err := json.Marshal(pair)
	if err != nil {
		log.Printf("E impossible to marshall pair: %s", err)
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%v://%v:%v/v1", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace), bytes.NewReader(marshalled))
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK || resp.StatusCode != http.StatusCreated {
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	bodyText, err := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusCreated && strings.TrimSpace(string(bodyText)) == "OK" {
		return nil
	}
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	log.Printf("%+v %+v", resp.StatusCode, bodyText)
}
func (c *Client) CreateNamespace(namespace string) error { // TODO
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	namespaceObj := rest.NamespaceV2{Name: namespace}
	marshalled, err := json.Marshal(namespaceObj)
	if err != nil {
		log.Printf("E impossible to marshall pair: %s", err)
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%v://%v:%v/v1", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace), bytes.NewReader(marshalled))
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK || resp.StatusCode != http.StatusCreated {
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	bodyText, err := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusCreated && strings.TrimSpace(string(bodyText)) == "OK" {
		return nil
	}
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	log.Printf("%+v %+v", resp.StatusCode, bodyText)
}
func (c *Client) DeleteNamespace(namespace string) error {
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%v://%v:%v/v1/%v", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace), nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK || resp.StatusCode != http.StatusCreated {
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	bodyText, err := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusCreated && strings.TrimSpace(string(bodyText)) == "OK" {
		return nil
	}
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	log.Printf("%+v %+v", resp.StatusCode, bodyText)
}
func (c *Client) DeleteKey(namespace string, key string) error {
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%v://%v:%v/v1/%v/%v", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace, key), nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK || resp.StatusCode != http.StatusCreated {
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	bodyText, err := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusCreated && strings.TrimSpace(string(bodyText)) == "OK" {
		return nil
	}
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	log.Printf("%+v %+v", resp.StatusCode, bodyText)
}
func (c *Client) Roll(namespace string, key string) error {
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	method := "{\"type\": \"roll\"}"
	req, err := http.NewRequest("UPDATE", fmt.Sprintf("%v://%v:%v/v1/%v/%v", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace, key), bytes.NewReader([]byte(method)))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK || resp.StatusCode != http.StatusCreated {
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	var pair rest.KVPairV2
	err = json.NewDecoder(resp.Body).Decode(&pair)
	if err != nil {
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("E Client Update(roll) call failed: %s", err)
		}
		log.Printf("E Client Update(roll) call failed: %s", bodyText)
		return err
	}
	if resp.StatusCode == http.StatusOK && key == pair.Key {
		return nil
	}
	log.Printf("%+v %+v", resp.StatusCode, pair)
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
}
func (c *Client) Generate(namespace string, key string) error {
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	method := "{\"type\": \"generate\"}"
	req, err := http.NewRequest("UPDATE", fmt.Sprintf("%v://%v:%v/v1/%v/%v", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace, key), bytes.NewReader([]byte(method)))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK || resp.StatusCode != http.StatusCreated {
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	var pair rest.KVPairV2
	err = json.NewDecoder(resp.Body).Decode(&pair)
	if err != nil {
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("E Client Update(generate) call failed: %s", err)
		}
		log.Printf("E Client Update(generate) call failed: %s", bodyText)
		return err
	}
	if resp.StatusCode == http.StatusOK && key == pair.Key {
		return nil
	}
	log.Printf("I %+v %+v", resp.StatusCode, pair)
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
}
func (c *Client) GetHealth() error {
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", c.BackendConfig.Protocol+"://"+c.BackendConfig.Host+":"+c.BackendConfig.Port+"/system/health", nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK || resp.StatusCode != http.StatusCreated {
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	var health rest.HealthV1
	err = json.NewDecoder(resp.Body).Decode(&health)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK && health.Status == "UP" {
		return nil
	}
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: ".Status not matching UP: " + health.Status}
}
