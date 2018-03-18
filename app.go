package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type SkiResort struct {
	Name  string
	Areas []ForecastArea
}

type ForecastArea struct {
	TimeUpdated string
	Name        string
	Url         string
	Periods     [4]ForecastPeriod
}

type ForecastPeriod struct {
	Name        string
	Weather     string
	Temperature uint
	Snow        string
}

type NOAAJsonResponse struct {
	Properties NOAAProperties
}

type NOAAProperties struct {
	Periods [13]NOAAPeriod
	Updated string
}

type NOAAPeriod struct {
	Name             string
	StartTime        string
	EndTime          string
	Temperature      uint
	ShortForecast    string
	DetailedForecast string
}

type Inventory struct {
	Material string
	Count    uint
}

func makeTemplate(inName string, outName string, forecast []SkiResort) {
	t, err := template.ParseFiles("templates/" + inName)
	check(err)

	f, err := os.Create("pages/" + outName)
	check(err)

	err = t.Execute(f, forecast)
	check(err)
	f.Close()
}

func TrimPrefix(s, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		s = s[len(prefix):]
	}
	return s
}

func TrimSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}

func UpdateForecast(area ForecastArea) ForecastArea {
	client := &http.Client{}
	req, err := http.NewRequest("GET", area.Url, nil)
	check(err)
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	res, err := client.Do(req)
	check(err)
	body, err := ioutil.ReadAll(res.Body)
	check(err)

	var response NOAAJsonResponse

	err = json.Unmarshal(body, &response)
	check(err)

	t, err := time.Parse(
		time.RFC3339,
		response.Properties.Updated)
	check(err)
	pst, err := time.LoadLocation("America/Los_Angeles")
	check(err)
	area.TimeUpdated = t.In(pst).Format("3:04pm, Jan _2")

	for i := 0; i <= 3; i++ {
		area.Periods[i] = ForecastPeriod{response.Properties.Periods[i].Name, response.Properties.Periods[i].ShortForecast, response.Properties.Periods[i].Temperature, ExtractSnow(response.Properties.Periods[i].DetailedForecast)}
		area.Periods[i].Temperature = response.Properties.Periods[i].Temperature
		area.Periods[i].Weather = response.Properties.Periods[i].ShortForecast
		area.Periods[i].Name = response.Properties.Periods[i].Name
		area.Periods[i].Snow = ExtractSnow(response.Properties.Periods[i].DetailedForecast)
	}
	return area
}

func ExtractSnow(data string) string {
	var findSnowAccumulation = regexp.MustCompile(`(New snow accumulation of )(.*)(possible.)`)
	snowAccumulation := string(findSnowAccumulation.Find([]byte(data)))
	snowAccumulation = TrimPrefix(snowAccumulation, "New snow accumulation of ")
	snowAccumulation = TrimSuffix(snowAccumulation, " possible.")
	return snowAccumulation
}

func UpdateAllForecasts(resorts []SkiResort) {
	for i := 0; i <= len(resorts)-1; i++ {
		for j := 0; j <= len(resorts[i].Areas)-1; j++ {
			resorts[i].Areas[j] = UpdateForecast(resorts[i].Areas[j])
		}
	}
}

func LoadResorts() []SkiResort {
	var resorts []SkiResort
	dat, err := ioutil.ReadFile("resorts.json")
	err = json.Unmarshal(dat, &resorts)
	check(err)
	return resorts
}

func PeriodicUpdate(interval time.Duration) {
	resorts := LoadResorts()
	UpdateAllForecasts(resorts)
	makeTemplate("index.html", "index.html", resorts)
	fmt.Println("Created forecasts")
	for range time.Tick(interval) {
		UpdateAllForecasts(resorts)
		makeTemplate("index.html", "index.html", resorts)
		fmt.Println("Updated forecasts")
	}
}

func main() {
	files, err := ioutil.ReadDir("./")
	check(err)

	for _, f := range files {
		fmt.Println(f.Name())
	}

	go PeriodicUpdate(30 * time.Minute)
	fs := http.FileServer(http.Dir("pages"))
	http.Handle("/", fs)

	log.Println("Listening...")
	http.ListenAndServe(":80", nil)
}
