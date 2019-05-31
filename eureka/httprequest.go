package eureka

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type HttpAction struct {
	Method            string `yaml:"method"`
	URL               string `yaml:"url"`
	Body              string `yaml:"body"`
	Template          string `yaml:"template"`
	Accept            string `yaml:"accept"`
	ContentType       string `yaml:"contentType"`
	Title             string `yaml:"title"`
	StoreCookie       string `yaml:"storeCookie"`
	HttpBasicUsername string
	HttpBasicPassword string
}

func DoHttpRequest(action HttpAction) bool {
	request := buildHttpRequest(action)

	var defaultTransport http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	response, err := defaultTransport.RoundTrip(request)

	if err != nil {
		log.Printf("Http request failed: %s", err)
		return false
	}

	if response.StatusCode < 200 || response.StatusCode > 300 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Printf("Http request failed: %d, invalid response body", response.StatusCode)
			return false
		}
		log.Printf("Http request failed: %d, %s", response.StatusCode, string(body))
		return false
	}

	defer response.Body.Close()
	return true
}

func buildHttpRequest(action HttpAction) *http.Request {
	var request *http.Request
	var err error

	if action.Body != "" {
		reader := strings.NewReader(action.Body)
		request, err = http.NewRequest(action.Method, action.URL, reader)
	} else if action.Template != "" {
		reader := strings.NewReader(action.Template)
		request, err = http.NewRequest(action.Method, action.URL, reader)
	} else {
		request, err = http.NewRequest(action.Method, action.URL, nil)
	}

	if err != nil {
		log.Fatal(err)
	}

	if action.HttpBasicUsername != "" {
		request.SetBasicAuth(action.HttpBasicUsername, action.HttpBasicPassword)
	}

	request.Header.Add("Accept", action.Accept)
	if action.ContentType != "" {
		request.Header.Add("Content-Type", action.ContentType)
	}

	return request
}
