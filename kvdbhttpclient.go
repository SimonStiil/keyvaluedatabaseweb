package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
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

func (c *Client) generatedBodyFromStatus(status int) string {
	return fmt.Sprintf("%v %v", status, http.StatusText(status))
}

func (c *Client) GetNamespaceList(logger *slog.Logger) ([]rest.NamespaceV2, error) {
	debugLogger := logger.With("function", "GetNamespaceList", "struct", "Client")
	debugLogger.Debug("Get Namespace List", "function", "GetNamespaceList", "struct", "Client")
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%v://%v:%v/v1/*", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port), nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK {
		debugLogger.Debug("Wrong status on request", "statuscode", resp.StatusCode, "response", resp)
		return nil, &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return nil, err
	}
	var list []rest.NamespaceV2
	err = json.NewDecoder(resp.Body).Decode(&list)
	if err != nil {
		bodyText, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, err
		}
		debugLogger.Debug("Json decoder error", "response", resp, "body", bodyText, "error", err)
	}
	return list, nil
}

func (c *Client) GetKeyList(logger *slog.Logger, namespace string) ([]rest.KVPairV2, error) {
	debugLogger := logger.With("function", "GetKeyList", "struct", "Client", "namespace", namespace)
	debugLogger.Debug("Get Key List")
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%v://%v:%v/v1/%v/*", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace), nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK {
		debugLogger.Debug("Wrong status on request", "statuscode", resp.StatusCode, "response", resp)
		return nil, &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return nil, err
	}
	var list []rest.KVPairV2
	err = json.NewDecoder(resp.Body).Decode(&list)
	if err != nil {
		bodyText, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, err
		}
		debugLogger.Debug("Json decoder error", "response", resp, "body", bodyText, "error", err)
	}
	return list, nil
}
func (c *Client) SetKey(logger *slog.Logger, namespace string, key string, value string) error {
	debugLogger := logger.With("function", "SetKey", "struct", "Client", "namespace", namespace)
	debugLogger.Debug("Set Key")
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	obj := rest.ObjectV1{Type: rest.TypeKey, Value: value}
	marshalled, err := json.Marshal(obj)
	if err != nil {
		logger.Error(fmt.Sprintf("Impossible to marshall pair: %s", err))
		return err
	}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%v://%v:%v/v1/%v/%v", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace, key), bytes.NewReader(marshalled))
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusCreated {
		debugLogger.Debug("Wrong status on request", "statuscode", resp.StatusCode, "response", resp)
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		debugLogger.Debug("ReadAll error", "response", resp, "error", err)
	}
	if strings.TrimSpace(string(bodyText)) == c.generatedBodyFromStatus(http.StatusCreated) {
		return nil
	}
	debugLogger.Debug("Content Error", "bodyText", bodyText)
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
}
func (c *Client) CreateNamespace(logger *slog.Logger, namespace string) error {
	debugLogger := logger.With("function", "CreateNamespace", "struct", "Client", "namespace", namespace)
	debugLogger.Debug("Create Namespace")
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	obj := rest.ObjectV1{Value: namespace}
	marshalled, err := json.Marshal(obj)
	if err != nil {
		logger.Error(fmt.Sprintf("impossible to marshall pair: %s", err))
		return err
	}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%v://%v:%v/v1", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port), bytes.NewReader(marshalled))
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		debugLogger.Debug("Wrong status on request", "statuscode", resp.StatusCode, "response", resp)
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		debugLogger.Debug("ReadAll error", "response", resp, "error", err)
	}
	if string(bodyText) == c.generatedBodyFromStatus(http.StatusCreated) {
		return nil
	}
	debugLogger.Debug("Content Error", "bodyText", bodyText)
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
}
func (c *Client) DeleteNamespace(logger *slog.Logger, namespace string) error {
	debugLogger := logger.With("function", "DeleteNamespace", "struct", "Client", "namespace", namespace)
	debugLogger.Debug("Delete Namespace")
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%v://%v:%v/v1/%v", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace), nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK {
		debugLogger.Debug("Wrong status on request", "statuscode", resp.StatusCode, "response", resp)
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		debugLogger.Debug("ReadAll error", "response", resp, "error", err)
	}
	if strings.TrimSpace(string(bodyText)) == c.generatedBodyFromStatus(http.StatusOK) {
		return nil
	}
	debugLogger.Debug("Content Error", "bodyText", bodyText)
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
}
func (c *Client) DeleteKey(logger *slog.Logger, namespace string, key string) error {
	debugLogger := logger.With("function", "DeleteKey", "struct", "Client", "namespace", namespace)
	debugLogger.Debug("Delete Key")
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%v://%v:%v/v1/%v/%v", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace, key), nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK {
		debugLogger.Debug("Wrong status on request", "statuscode", resp.StatusCode, "response", resp)
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		debugLogger.Debug("ReadAll error", "response", resp, "error", err)
	}
	if strings.TrimSpace(string(bodyText)) == c.generatedBodyFromStatus(http.StatusOK) {
		return nil
	}
	debugLogger.Debug("Content Error", "bodyText", bodyText)
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
}
func (c *Client) Roll(logger *slog.Logger, namespace string, key string) error {
	debugLogger := logger.With("function", "Roll", "struct", "Client", "namespace", namespace)
	debugLogger.Debug("Roll")
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	obj := rest.ObjectV1{Type: rest.TypeRoll}
	marshalled, err := json.Marshal(obj)
	if err != nil {
		logger.Error(fmt.Sprintf("Impossible to marshall pair: %s", err))
		return err
	}
	req, _ := http.NewRequest("UPDATE", fmt.Sprintf("%v://%v:%v/v1/%v/%v", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace, key), bytes.NewReader(marshalled))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusCreated {
		debugLogger.Debug("Wrong status on request", "statuscode", resp.StatusCode, "response", resp)
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	var pair rest.KVPairV2
	err = json.NewDecoder(resp.Body).Decode(&pair)
	if err != nil {
		bodyText, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return err
		}
		debugLogger.Debug("Json decoder error", "response", resp, "body", bodyText, "error", err)
	}
	if key == pair.Key {
		return nil
	}
	debugLogger.Debug("Content Error", "pair", pair)
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
}
func (c *Client) Generate(logger *slog.Logger, namespace string, key string) error {
	debugLogger := logger.With("function", "Generate", "struct", "Client", "namespace", namespace)
	debugLogger.Debug("Generate")
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	obj := rest.ObjectV1{Type: rest.TypeGenerate}
	marshalled, err := json.Marshal(obj)
	if err != nil {
		logger.Error(fmt.Sprintf("Impossible to marshall pair: %s", err))
		return err
	}
	req, _ := http.NewRequest("UPDATE", fmt.Sprintf("%v://%v:%v/v1/%v/%v", c.BackendConfig.Protocol, c.BackendConfig.Host, c.BackendConfig.Port, namespace, key), bytes.NewReader(marshalled))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusCreated {
		debugLogger.Debug("Wrong status on request", "statuscode", resp.StatusCode, "response", resp)
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if err != nil {
		return err
	}
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		debugLogger.Debug("Read Body Error", "response", resp, "error", readErr)
		return err
	}
	reader := io.NopCloser(bytes.NewReader(bodyBytes))
	var pair rest.KVPairV2
	err = json.NewDecoder(reader).Decode(&pair)
	if err != nil {
		debugLogger.Debug("Json decoder error", "response", resp, "body", string(bodyBytes), "error", err)
	}
	if key == "" || key == pair.Key {
		return nil
	}
	debugLogger.Debug("Content Error", "pair", pair)
	return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
}
func (c *Client) GetHealth(logger *slog.Logger) error {
	debugLogger := logger.With("function", "GetHealth", "struct", "Client")
	debugLogger.Debug("Get Health")
	transport := &http.Transport{TLSClientConfig: c.TLSCpnfig}
	client := &http.Client{Transport: transport}
	req, _ := http.NewRequest("GET", c.BackendConfig.Protocol+"://"+c.BackendConfig.Host+":"+c.BackendConfig.Port+"/system/health", nil)
	req.SetBasicAuth(c.BackendConfig.Username, c.Password)
	resp, err := client.Do(req)
	if !(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated) {
		debugLogger.Debug("Wrong status on request", "statuscode", resp.StatusCode, "response", resp)
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
