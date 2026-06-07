package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

var apiEndpoint = "http://localhost:8080/api/v1"

func apiGet(path string) (map[string]interface{}, error) {
	resp, err := http.Get(apiEndpoint + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func apiPost(path string, data interface{}) (map[string]interface{}, error) {
	jsonBody, _ := json.Marshal(data)
	resp, err := http.Post(apiEndpoint+path, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func apiDelete(path string) error {
	req, _ := http.NewRequest("DELETE", apiEndpoint+path, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func apiPatch(path string, data interface{}) (map[string]interface{}, error) {
	jsonBody, _ := json.Marshal(data)
	req, _ := http.NewRequest("PATCH", apiEndpoint+path, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func setAPIEndpoint(endpoint string) {
	apiEndpoint = strings.TrimSuffix(endpoint, "/")
	if !strings.HasSuffix(apiEndpoint, "/api/v1") {
		apiEndpoint = apiEndpoint + "/api/v1"
	}
}

func checkAPIConnection() error {
	_, err := apiGet("/instances")
	return err
}
