package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func GetClient(xToken string) (*Client, error) {
	url := "https://api.monobank.ua/personal/client-info"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %w\n", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Token", xToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error making GET request: %w\n", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %w\n", err)
	}

	var apiClient Client
	err = json.Unmarshal(body, &apiClient)
	if err != nil {
		return nil, fmt.Errorf("Error parsing response body: %w\n", err)
	}

	return &apiClient, nil
}

func GetJar(name string, jars []Jar) *Jar {
	for _, jar := range jars {
		if jar.Title == name {
			return &jar
		}
	}
	return nil
}

func GetStatement(xToken string, accountId string, from time.Time, to time.Time) ([]Transaction, error) {
	urlFormat := "https://api.monobank.ua/personal/statement/%s/%d/%d"
	url := fmt.Sprintf(urlFormat, accountId, from.Unix(), to.Unix())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %w\n", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Token", xToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error making GET request: %w\n", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %w\n", err)
	}

	var transactions []Transaction
	err = json.Unmarshal(body, &transactions)
	if err != nil {
		return nil, fmt.Errorf("Error parsing response body: %w\n", err)
	}

	return transactions, nil
}

func GetStatementForLast(xToken string, d time.Duration, j Jar) ([]Transaction, error) {
	now := time.Now()
	dayBefore := now.Add(-d)
	statement, err := GetStatement(xToken, j.ID, dayBefore, now)
	if err != nil {
		return nil, err
	}
	return statement, nil
}
