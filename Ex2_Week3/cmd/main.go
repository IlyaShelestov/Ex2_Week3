package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
)

type Interaction struct {
	Request  string
	Response string
}

var (
	history         []Interaction
	lock            sync.Mutex
	allowedKeywords = []string{}
)

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	t.Execute(w, data)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderTemplate(w, "./web/index.html", map[string]interface{}{"History": history, "Filter": strings.Join(allowedKeywords, ", ")})
	} else if r.Method == "POST" {
		r.ParseForm()
		requestType := r.FormValue("type")
		if requestType == "request" {
			requestText := r.FormValue("request_text")
			if len(allowedKeywords) == 0 || isValidRequest(requestText) {
				responseContent := callAPI(requestText)
				updateHistory(requestText, responseContent)
				renderTemplate(w, "./web/index.html", map[string]interface{}{"Response": responseContent, "History": history, "Filter": strings.Join(allowedKeywords, ", ")})
			} else {
				renderTemplate(w, "./web/index.html", map[string]interface{}{"Response": "Your request was declined because your question does not match the specified filters", "History": history, "Filter": strings.Join(allowedKeywords, ", ")})
			}
		} else if requestType == "history_log" {
			renderTemplate(w, "./web/index.html", map[string]interface{}{"History": history, "Filter": strings.Join(allowedKeywords, ", ")})
		} else if requestType == "clear" {
			clearFilters()
			renderTemplate(w, "./web/index.html", map[string]interface{}{"History": history, "Filter": strings.Join(allowedKeywords, ", ")})
		} else if requestType == "filter" {
			newFilters := r.FormValue("filter")
			updateFilters(newFilters)
			renderTemplate(w, "./web/index.html", map[string]interface{}{"History": history, "Filter": strings.Join(allowedKeywords, ", ")})
		}
	}
}

func clearFilters() {
	lock.Lock()
	defer lock.Unlock()
	allowedKeywords = []string{}
}

func updateFilters(newFilters string) {
	lock.Lock()
	defer lock.Unlock()
	filters := strings.Split(newFilters, " ")
	keywordMap := make(map[string]bool)
	for _, keyword := range allowedKeywords {
		keywordMap[keyword] = true
	}
	for _, keyword := range filters {
		if _, exists := keywordMap[keyword]; !exists && keyword != "" {
			allowedKeywords = append(allowedKeywords, keyword)
			keywordMap[keyword] = true
		}
	}
}

func isValidRequest(input string) bool {
	for _, keyword := range allowedKeywords {
		if strings.Contains(strings.ToLower(input), keyword) {
			return true
		}
	}
	return false
}

func updateHistory(request, response string) {
	lock.Lock()
	defer lock.Unlock()
	if len(history) >= 5 {
		history = history[1:]
	}
	history = append(history, Interaction{Request: request, Response: response})
}

func callAPI(input string) string {
	client := resty.New()
	apiKey := "INSERT_YOUR_API_KEY"
	apiEndpoint := "https://api.openai.com/v1/chat/completions"

	response, err := client.R().
		SetAuthToken(apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "system",
					"content": "You are an assistant.",
				},
				map[string]interface{}{
					"role":    "user",
					"content": input,
				},
			},
			"max_tokens": 1024,
		}).
		Post(apiEndpoint)

	if err != nil {
		return "Error while sending the request: " + err.Error()
	}

	body := response.Body()

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "Error while decoding JSON response: " + err.Error()
	}

	content := data["choices"].([]interface{})[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)
	return content
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("Server starting on port 3030...")
	log.Fatal(http.ListenAndServe(":3030", nil))
}
