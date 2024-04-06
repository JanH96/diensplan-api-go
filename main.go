package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type ADEintrag struct {
	Titel        string
	Beschreibung string
	Ort          string
	Startdatum   time.Time
	Startuhrzeit time.Time
	Enddatum     time.Time
	Enduhrzeit   time.Time
}

type DienstplanEintrag struct {
	Nr           int8      `json:"nr"`
	Datum        time.Time `json:"datum"`
	Startuhrzeit time.Time `json:"startuhrzeit"`
	Enduhrzeit   time.Time `json:"enduhrzeit"`
	Ort          string    `json:"ort"`
	Thema        string    `json:"thema"`
	Grundlage    string    `json:"grundlage"`
	Art          string    `json:"art"`
	Geraete      string    `json:"geraete"`
	Leitender    []string  `json:"leitender"`
	Kommentar    string    `json:"kommentar"`
}

func getStartAndEndTime(s string) (string, string) {
	str := strings.Split(s, "-")

	// catch if only start time given
	if len(str) < 2 {
		str = append(str, "23:59")
	}

	if str[0] == "" || str[0] == "offen" {
		str[0] = "00:00"
	}

	start := strings.TrimSpace(str[0])
	end := strings.TrimSpace(str[1])

	return start, end
}

func getInstructors(str string) []string {

	var instructors []string

	if strings.Contains(str, "/") {

		splitted := strings.Split(str, "/")

		for _, s := range splitted {
			instructors = append(instructors, strings.TrimSpace(s))
		}

	} else {
		instructors = append(instructors, str)
	}
	return instructors
}

func sortEntries(data []map[string]string) []DienstplanEintrag {

	var entries []DienstplanEintrag

	for i := 0; i < len(data); i++ {

		if data[i]["Art"] == "U" || data[i]["Art"] == "U/P" || data[i]["Art"] == "P" {

			var e = new(DienstplanEintrag)

			// Art
			e.Art = data[i]["Art"]

			date, _ := time.Parse("1/2/06", data[i]["Datum"])

			startTime, endTime := getStartAndEndTime(data[i+1]["Datum"])

			parsedStart, _ := time.Parse("15:04", startTime)
			parsedEnd, _ := time.Parse("15:04", endTime)

			// Startzeit
			e.Startuhrzeit = parsedStart

			// Endzeit
			e.Enduhrzeit = parsedEnd

			// Datum
			e.Datum = date

			// Leitender
			var instructors = getInstructors(data[i]["Leitender"])
			e.Leitender = append(e.Leitender, instructors...)
			ltd, ok := data[i+1]["Leitender"]
			if ok {
				var instructors = getInstructors(ltd)
				e.Leitender = append(e.Leitender, instructors...)
			}

			// Thema
			e.Thema = data[i]["AusbildungsgebietThema"]
			nr, err := strconv.Atoi(data[i]["Lfd"])
			if err != nil {
				nr = 0
			}

			e.Nr = int8(nr)

			e.Ort = data[i]["Ort"]
			e.Geraete = data[i]["Geräte"]
			e.Kommentar = data[i]["Erläuterung"]
			e.Grundlage = data[i]["Grundlage"]

			// append to output
			entries = append(entries, *e)
		}

		if data[i]["Lfd"] == "U" {
			break
		}

	}
	return entries
}

func handleSchedule(c *gin.Context) {
	var jsonData []DienstplanEintrag

	var adList []ADEintrag

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}

	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse JSON data"})
		return
	}

	for _, e := range jsonData {
		var adE = new(ADEintrag)

		adE.Titel = e.Thema

		// ort hardcoded
		adE.Ort = "Gerätehaus"
		adE.Startdatum = e.Datum
		adE.Enddatum = e.Datum
		adE.Startuhrzeit = e.Startuhrzeit
		adE.Enduhrzeit = e.Enduhrzeit
		adE.Beschreibung = ""

		adList = append(adList, *adE)
	}

	c.JSON(http.StatusAccepted, adList)
}

func handleExcel(c *gin.Context) {

	var jsonData []map[string]string

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}

	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse JSON data"})
		return
	}

	sortedEntries := sortEntries(jsonData)

	c.JSON(http.StatusAccepted, sortedEntries)

	// c.JSON(http.StatusAccepted, gin.H{
	// 	"message": "SortedJsonList",
	// 	"payload": sortedEntries,
	// })
}

func main() {

	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"} // Replace with your frontend URL
	r.Use(cors.New(config))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.POST("/excel", handleExcel)

	r.POST("/schedule", handleSchedule)

	r.Run(":8080")
}
