package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	gziphandler "github.com/NYTimes/gziphandler"
	trainmapdb "github.com/rom-vtn/trainmap-db"
	"gorm.io/driver/sqlite"
)

// a ServerConfig represents the config needed to run a server
type ServerConfig struct {
	HostPort             uint16 `json:"host_port"`
	FrontendWebroot      string `json:"frontend_root"`
	SightPreviewDayCount uint   `json:"sight_preview_day_count"`
	DatabaseFilepath     string `json:"database_filepath"`
	ServeFrontend        bool   `json:"serve_frontend"`
}

var serverFetcher *trainmapdb.Fetcher
var serverConfig ServerConfig

func gzipHandler(f func(http.ResponseWriter, *http.Request)) http.Handler {
	return gziphandler.GzipHandler(http.HandlerFunc(f))
}

func main() {
	//load config
	if len(os.Args) < 2 {
		log.Fatalf("Syntax: %s <config_file.json>", os.Args[0])
	}
	configContents, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("got error while reading config file: %s", err.Error())
	}
	err = json.Unmarshal(configContents, &serverConfig)
	if err != nil {
		log.Fatalf("got error while decoding server config: %s", err.Error())
	}

	//init fetcher
	dial, useMutex := sqlite.Open(serverConfig.DatabaseFilepath), true
	// dsn := "host=localhost user=postgres password=password dbname=postgres port=5432 TimeZone=Europe/Paris"
	// dial, useMutex := postgres.Open(dsn), false
	fetcher, err := trainmapdb.NewFetcher(dial, useMutex, nil)
	if err != nil {
		log.Fatalf("could not open database, exiting: %s", err.Error())
	}
	serverFetcher = fetcher

	if serverConfig.ServeFrontend {
		//basic file server
		http.Handle("/", http.FileServer(http.Dir(serverConfig.FrontendWebroot)))
	}

	// add sights api
	http.Handle("GET /api/sights/{lat}/{lon}/", gzipHandler(sightsHandler))
	http.Handle("GET /api/sights/{lat}/{lon}/{date}/", gzipHandler(sightsHandler))

	//moving sights api
	http.Handle("GET /api/aboard/{feedId}/{tripId}/", gzipHandler(aboardHandler))
	http.Handle("GET /api/aboard/{feedId}/{tripId}/{date}", gzipHandler(aboardHandler))
	http.Handle("GET /api/aboard/{feedId}/{tripId}/{date}/{lateSeconds}", gzipHandler(aboardHandler))

	// add DB query entries
	http.Handle("GET /api/data/{dataType}/{firstKey}/{secondKey}/", gzipHandler(databaseHandler))
	http.Handle("GET /api/data/{dataType}/{firstKey}/", gzipHandler(databaseHandler))
	http.Handle("GET /api/data/{dataType}/", gzipHandler(databaseHandler))

	listenString := ":" + fmt.Sprint(serverConfig.HostPort)
	println("Listening on ", listenString)
	log.Fatal(http.ListenAndServe(listenString, nil))
}

type APIErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func sendError(w http.ResponseWriter, err error) {
	w.WriteHeader(400)
	errResponse := APIErrorResponse{
		Success: false,
		Error:   err.Error(),
	}
	response, err := json.Marshal(errResponse)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(response)
}

// API response in case of (static) sights
type APISightsResponse struct {
	Success      bool                        `json:"success"`
	Error        string                      `json:"error"`
	ObsPoint     trainmapdb.Point            `json:"observation_point"`
	FirstDate    time.Time                   `json:"first_date"`
	LastDate     time.Time                   `json:"last_date"`
	PassingTimes []trainmapdb.RealTrainSight `json:"passing_times"`
}

// API response for sights aboard a train
type APIMovingSightsResponse struct {
	Success bool                              `json:"success"`
	Error   string                            `json:"error"`
	Trip    trainmapdb.Trip                   `json:"trip"`
	Date    time.Time                         `json:"date"`
	Sights  []trainmapdb.RealMovingTrainSight `json:"sights"`
}

// handle API requests for sights aboard trains
func aboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	//parse URL
	feedId := r.PathValue("feedId")
	tripId := r.PathValue("tripId")
	dateString := r.PathValue("date")
	lateSecondsString := r.PathValue("lateSeconds")
	if lateSecondsString == "" {
		lateSecondsString = "0"
	}
	lateSecondsInt, err := strconv.ParseInt(lateSecondsString, 10, 64)
	if err != nil {
		sendError(w, err)
		return
	}
	var date time.Time
	lateTime := time.Duration(lateSecondsInt) * time.Second
	if dateString == "" {
		date = time.Now()
	} else {
		date, err = time.Parse("2006-01-02", dateString)
		if err != nil {
			sendError(w, err)
			return
		}
	}

	//respond
	trip, err := serverFetcher.GetTrip(feedId, tripId)
	if err != nil {
		sendError(w, err)
		return
	}
	realMovingTrainSights, newTrip, err := serverFetcher.GetSightsFromTrip(trip, trainmapdb.NewDate(date), lateTime)
	if err != nil {
		sendError(w, err)
		return
	}
	response := APIMovingSightsResponse{
		Success: true,
		Error:   "",
		Trip:    newTrip,
		Date:    date.Truncate(24 * time.Hour),
		Sights:  realMovingTrainSights,
	}

	raw, err := json.Marshal(response)
	if err != nil {
		sendError(w, err)
		return
	}
	w.Write(raw)
}

// handle API requests for sights at a given point
func sightsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	lat, err := strconv.ParseFloat(r.PathValue("lat"), 64)
	if err != nil {
		sendError(w, err)
		return
	}
	lon, err := strconv.ParseFloat(r.PathValue("lon"), 64)
	if err != nil {
		sendError(w, err)
		return
	}
	// obsPoint := tq.Point{Lat: lat, Lon: lon}
	// startDate, endDate := tq.GetDateInterval()
	// realTrainSights, err := tq.GetRealTrainSights(obsPoint, nil, startDate, endDate)
	var startDate, endDate trainmapdb.Date
	var startDateTime time.Time
	dateString := r.PathValue("date")
	if dateString == "" {
		startDateTime = time.Now()
	} else {
		startDateTime, err = time.Parse("2006-01-02", dateString)
		if err != nil {
			sendError(w, err)
			return
		}
	}
	startDate, endDate = trainmapdb.GetDateInterval(serverConfig.SightPreviewDayCount, startDateTime)
	fetcher := serverFetcher
	obsPoint := trainmapdb.Point{Lat: lat, Lon: lon}
	realTrainSights, err := fetcher.GetRealTrainSights(obsPoint, startDate, endDate)
	if err != nil {
		sendError(w, err)
		return
	}

	response := APISightsResponse{
		Success:      true,
		Error:        "",
		FirstDate:    time.Time(startDate), //cast back to time.Time
		LastDate:     time.Time(endDate),   //cast back to time.Time
		ObsPoint:     obsPoint,
		PassingTimes: realTrainSights,
	}

	raw, err := json.Marshal(response)
	if err != nil {
		sendError(w, err)
		return
	}
	w.Write(raw)
}

// handle API queries for a specific item in the DB
func databaseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fetcher := serverFetcher
	var err error
	dataType := r.PathValue("dataType")
	firstKey := r.PathValue("firstKey")
	secondKey := r.PathValue("secondKey")

	var content any
	switch dataType {
	case "trips":
		content, err = fetcher.GetTrip(firstKey, secondKey)
	case "routes":
		content, err = fetcher.GetRoute(firstKey, secondKey)
	case "stops":
		content, err = fetcher.GetStop(firstKey, secondKey)
	case "feeds":
		if firstKey == "" {
			//if no more details, return all feeds
			content, err = fetcher.GetFeeds()
		} else {
			//otherwise, return a specific feed
			content, err = fetcher.GetFeed(firstKey)
		}
	case "stoptimes":
		content, err = fetcher.GetStopTimesAtStop(firstKey, secondKey)
	default:
		sendError(w, fmt.Errorf("invalid data type requested: %s", dataType))
		return
	}
	if err != nil {
		sendError(w, err)
		return
	}
	raw, err := json.Marshal(APIDatabaseResponse{
		Success: true,
		Error:   "",
		Content: content,
	})
	if err != nil {
		sendError(w, err)
		return
	}
	w.Write(raw)
}

// Reponse for API DB queries
type APIDatabaseResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Content any    `json:"content"`
}
