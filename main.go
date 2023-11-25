package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	client      = &http.Client{}
	bearerToken string
)

func fetchData(url string) ([]map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil) // fetch data our storage
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+bearerToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []map[string]interface{} // the json data fetched are being represented as a slice and we return the slice in a map
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func extractField(data map[string]interface{}, fieldPath string) interface{} {
	parts := strings.Split(fieldPath, ".") // the goes through the array one by one seprated by .
	fieldValue := interface{}(
		data,
	) // we are returning the fiels data so it have to come in as an interface

	for _, part := range parts {
		fieldValueMap, ok := fieldValue.(map[string]interface{})
		if !ok {
			return nil
		}
		fieldValue = fieldValueMap[part]
	}
	return fieldValue
}

func incinerateItem(wg *sync.WaitGroup, url string, id string) error {
	defer wg.Done()

	payload := fmt.Sprintf(`{
		"itemToIncinerateId": "%s"
	}`, id) // the string our incinerator takes in and that is the id of the item to be incinerated

	// the payload is just the json, the PUT statement takes in as an input
	req, err := http.NewRequest("PUT", url, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+bearerToken)

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

func main() {
	flag.StringVar(&bearerToken, "bearer-Token", "", "Bearer token for Authorization")

	flag.Parse()

	if bearerToken == "" {
		fmt.Println("You can not gain Authorization without your bearerToken")
		os.Exit(1)
	}

	startTime := time.Now()

	apiData, err := fetchData("https://api.theremnants.app/bank/storage?search=")
	if err != nil {
		fmt.Println("Error fetching data:", err)
		return
	}

	var wg sync.WaitGroup

	for _, item := range apiData {
		id := extractField(item, "item.id")
		degradation := extractField(item, "item.degradation")

		if degradation != nil {
			if value, ok := degradation.(float64); ok && value == 0 {
				// i added these switch statement because there are times the storage will be empty and id will be nil, we cant convert it to string
				// the id is needed as string cause that is the json taken in by the PUT statement of our api
				switch id := id.(type) {
				case string:
					wg.Add(1)
					go incinerateItem(&wg, "https://api.theremnants.app/item/incinerate", id)
				}
			}
		}
	}

	elapsedTime := time.Since(startTime)
	wg.Wait()
	fmt.Println("All incineration operations completed successfully. \n", elapsedTime)
}
