package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mohae/struct2csv"
)

type SchoolData struct {
	Name string `json:"name,omitempty"`
	Lat  string `json:"lat,omitempty"`
	Lng  string `json:"lng,omitempty"`
}

// var allSchools = make([]SchoolData, 0)
var latLngMatcher *regexp.Regexp
var csvWriter *csv.Writer
var wg sync.WaitGroup

func main() {

	tmpRegExp, err := regexp.Compile(".*var uluru.*\n")

	if err != nil {
		fmt.Println(err)
		return
	}

	latLngMatcher = tmpRegExp

	csvFile, err := os.OpenFile("Pune School Data.csv", os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}

	csvWriter = csv.NewWriter(csvFile)

	SchoolsURL := "https://schools.org.in/maharashtra/pune"

	c := colly.NewCollector()

	c.OnError(func(r *colly.Response, e error) {
		fmt.Println(e)
	})

	// Find and visit all links
	c.OnHTML("table", func(e *colly.HTMLElement) {
		visitLinkFromTable(e, visitBlock)
	})

	c.Visit(SchoolsURL)

	// writeToCSV(allSchools)
}

func visitBlock(blockURL string) {
	blockColly := colly.NewCollector()

	blockColly.OnHTML("table", func(e *colly.HTMLElement) {
		visitLinkFromTable(e, visitSchool)
	})

	blockColly.Visit(blockURL)
}

func visitSchool(clusterURL string) {
	clusterColly := colly.NewCollector()

	clusterColly.OnHTML("table", func(e *colly.HTMLElement) {
		visitLinkFromTable(e, getData)
	})

	clusterColly.Visit(clusterURL)
}

func getData(str string) {
	schoolColly := colly.NewCollector()

	schoolColly.OnHTML("html", func(e *colly.HTMLElement) {

		var finalScript string
		for _, script := range e.ChildTexts("script") {
			if strings.Contains(script, "initMap") {
				res := latLngMatcher.FindString(script)
				finalScript = res
			}
		}

		tmpData := fetchLatLng(finalScript)

		data := SchoolData{
			Name: strings.TrimSpace(e.ChildText(".shd")),
			Lat:  tmpData.Lat,
			Lng:  tmpData.Lng,
		}

		if data.Lat != "" {
			fmt.Println(data)

			tmp := make([]SchoolData, 0)
			tmp = append(tmp, data)

			writeToCSV(tmp)
			// allSchools = append(allSchools, data)
		}
	})

	schoolColly.Visit(str)
}

func fetchLatLng(script string) SchoolData {
	replacedString := strings.ReplaceAll(strings.TrimSpace(script), "var uluru = ", "")

	split := strings.Split(replacedString, ", ")

	lat := strings.ReplaceAll(split[0], "{lat:", "")
	lng := strings.ReplaceAll(strings.ReplaceAll(split[1], "lng:", ""), "};", "")

	return SchoolData{
		Name: "",
		Lat:  lat,
		Lng:  lng,
	}
}

func visitLinkFromTable(e *colly.HTMLElement, visiter func(link string)) {
	e.ForEachWithBreak("tr", func(i int, h *colly.HTMLElement) bool {
		h.ForEachWithBreak("a[href]", func(i int, j *colly.HTMLElement) bool {
			link := j.Attr("href")
			visiter(j.Request.AbsoluteURL(link))
			return false
		})
		return true
	})
}

func writeToCSV(data []SchoolData) {
	enc := struct2csv.New()

	var rows [][]string
	for _, v := range data {
		row, err := enc.GetRow(v)
		if err != nil {
			fmt.Println(err)
		}
		rows = append(rows, row)
	}

	err := csvWriter.WriteAll(rows)
	if err != nil {
		fmt.Println(err)
	}
}
