package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func main() {
	type DaySpec struct {
		Cases int
		Date  time.Time
	}

	fmt.Println("Starting the application...")

	response, err := http.Get("https://api.covid19api.com/total/country/united-kingdom/status/deaths")

	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)

		// get the byte response as a string
		var jsonData = string(data)

		// declare the array to deserialise to
		var days []DaySpec

		// deserialize the josn string into days
		json.Unmarshal([]byte(jsonData), &days)

		todaySpec := days[len(days)-1]

		fmt.Println(fmt.Sprintf("%d cases on %s", todaySpec.Cases, todaySpec.Date))
	}

	fmt.Println("Terminating the application...")
}
