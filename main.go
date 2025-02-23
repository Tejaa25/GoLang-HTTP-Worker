package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

// InputData represents the incoming JSON structure
type InputData struct {
	Ev    string `json:"ev"`
	Et    string `json:"et"`
	ID    string `json:"id"`
	UID   string `json:"uid"`
	MID   string `json:"mid"`
	T     string `json:"t"`
	P     string `json:"p"`
	L     string `json:"l"`
	SC    string `json:"sc"`
	Attrs map[string]string `json:"attributes"`
	Traits map[string]string `json:"traits"`
}

// TransformedData represents the final JSON structure
type TransformedData struct {
	Event          string                 `json:"event"`
	EventType      string                 `json:"event_type"`
	AppID          string                 `json:"app_id"`
	UserID         string                 `json:"user_id"`
	MessageID      string                 `json:"message_id"`
	PageTitle      string                 `json:"page_title"`
	PageURL        string                 `json:"page_url"`
	BrowserLang    string                 `json:"browser_language"`
	ScreenSize     string                 `json:"screen_size"`
	Attributes     map[string]interface{} `json:"attributes"`
	Traits         map[string]interface{} `json:"traits"`
}

var requestChannel = make(chan InputData, 10)
var wg sync.WaitGroup

func main() {
	// Start worker
	go worker()

	// Start HTTP server
	http.HandleFunc("/receive", receiveHandler)
	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func receiveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var input InputData
	if err := json.Unmarshal(body, &input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Send to channel for processing
	requestChannel <- input

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Success"))
}

func worker() {
	for data := range requestChannel {
		transformed := transformData(data)
		sendToWebhook(transformed)
	}
}

func transformData(data InputData) TransformedData {
	return TransformedData{
		Event:       data.Ev,
		EventType:   data.Et,
		AppID:       data.ID,
		UserID:      data.UID,
		MessageID:   data.MID,
		PageTitle:   data.T,
		PageURL:     data.P,
		BrowserLang: data.L,
		ScreenSize:  data.SC,
		Attributes:  parseAttributes(data.Attrs),
		Traits:      parseAttributes(data.Traits),
	}
}

func parseAttributes(attrs map[string]string) map[string]interface{} {
	parsed := make(map[string]interface{})
	for k, v := range attrs {
		parsed[k] = map[string]string{
			"value": v,
			"type":  "string",
		}
	}
	return parsed
}

func sendToWebhook(data TransformedData) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println("Error marshalling JSON:", err)
		return
	}

	req, err := http.NewRequest("POST", "https://webhook.site/", bytes.NewReader(jsonData))
	if err != nil {
		log.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Webhook response:", string(body))
}

