package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	elastic "gopkg.in/olivere/elastic.v7"
)

const defaultIndexName string = "covid-uk-deaths"
const localClusterURL string = "http://localhost:9200"

// DaySpec : here you tell us what Salutation is
type DaySpec struct {
	Cases int
	Date  time.Time
}

var wg sync.WaitGroup

func main() {
	if isLambdaContext() {
		lambda.Start(covidHandler)
	} else {
		fmt.Println("Starting the application from local context...")
		covidHandler()
	}
}

func isLambdaContext() bool {
	_, hasEnvVars := os.LookupEnv("ESINDEXNAME")
	return hasEnvVars
}

func covidHandler() (string, error) {
	fmt.Println("Querying covid19api...")

	apiResponse, connErr := http.Get("https://api.covid19api.com/total/country/united-kingdom/status/deaths")

	if connErr != nil {
		fmt.Printf("The HTTP request failed with error %s\n", connErr)
		return "", errors.New("failed to read covid API, exiting")
	}

	ukMortalityData, _ := ioutil.ReadAll(apiResponse.Body)

	// get the byte response as a string
	var jsonData = string(ukMortalityData)

	// declare the array to deserialise to
	var days []DaySpec

	// deserialize the json string into days
	json.Unmarshal([]byte(jsonData), &days)

	ctx := context.Background()
	esclient, err := getESClient()

	if err != nil {
		fmt.Println("Error initializing : ", err)
		panic("Client fail.")
	} else {
		fmt.Println("ES client initialized.")
	}

	wg.Add(len(days))

	for _, daySpec := range days {
		go postDaySpec(ctx, daySpec, esclient)
	}

	wg.Wait()

	todaySpec := days[len(days)-1]
	returnOutput := fmt.Sprintf("%d total UK deaths to date", todaySpec.Cases)
	return returnOutput, nil
}

// postDaySpec : Contrived to test concurrency - would normally just be done as batch post to ES
func postDaySpec(ctx context.Context, daySpec DaySpec, esclient *elastic.Client) {
	fmt.Println(fmt.Sprintf("%d cases on %s", daySpec.Cases, daySpec.Date))

	dataJSON, err := json.Marshal(daySpec)
	js := string(dataJSON)

	_, indexErr := esclient.Index().
		Id(daySpec.Date.Format("2006-01-02")).
		Index(getEnvVarOrDefault("ESINDEXNAME", defaultIndexName)).
		BodyJson(js).
		Do(ctx)

	if indexErr != nil {
		fmt.Println("Error indexing: ", err)
	}

	wg.Done()
}

func getESClient() (*elastic.Client, error) {
	client, err := elastic.NewClient(
		elastic.SetURL(getEnvVarOrDefault("ESCLUSTERURL", localClusterURL)),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false))

	return client, err
}

func getEnvVarOrDefault(envVarKey string, defaultVal string) string {
	envVarValue := os.Getenv(envVarKey)
	if envVarValue == "" {
		envVarValue = defaultVal
	}
	return envVarValue
}
