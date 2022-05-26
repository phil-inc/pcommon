package google_sheets

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/phil-inc/plog/logging"
)

var logger = logging.GetContextLogger("network")

var httpClient = &http.Client{
	Timeout: time.Second * 60,
}

// HTTPGet - makes a get request to the given URL and HTTP headers.
// it returns response data byte or error
func HTTPGet(url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	//add headers!
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	log.Printf("GET request to url: %s\n", url)
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	if res.StatusCode != 200 && res.StatusCode != 201 {
		if res.StatusCode >= 500 {
			logger.ErrorPrintf("[EXTERNAL][FATAL][GET] %d response code with external service at URL: %s. Status: %s", res.StatusCode, url, res.Status)
		}
		return nil, fmt.Errorf("Http response NOT_OK. Status: %s, Code:%d", res.Status, res.StatusCode)
	}

	return ioutil.ReadAll(res.Body)
}
