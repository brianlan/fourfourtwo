package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var PREFIX = "http://www.fourfourtwo.com"

// Match is the overview information about match, not the stats / details in a match.
type Match struct {
	id                   string
	season, date, time   string
	leagueId             string
	homeTeam, awayTeam   string
	homeScore, awayScore string
	url                  string
}

type League struct {
	id   string
	name string
}

type Player struct {
	id   string
	name string
}

type PlayerStats struct {
	matchId  string
	playerId string
	url      string
	events   *[]PlayerEvents
}

type Point struct {
	x, y float64
}

// If the event has no directions, then the startPoint would store the position of this event, leaving endPoint empty.
// Pitch range is from (57, 58) to (680, 470) in the raw D3 position, need to transform it.
type PlayerEvents struct {
	half                 string
	minute               string
	eventType            string
	startPoint, endPoint Point
}

var MonthMap = map[string]string{
	"January":   "01",
	"February":  "02",
	"March":     "03",
	"April":     "04",
	"May":       "05",
	"June":      "06",
	"July":      "07",
	"August":    "08",
	"September": "09",
	"October":   "10",
	"November":  "11",
	"December":  "12"}

func ConstructDate(season, day, month string) string {
	seasonInt, err := strconv.Atoi(season)
	if err != nil {
		log.Fatal(err)
	}
	seasonPlus := strconv.Itoa(seasonInt + 1)

	if month == "08" || month == "09" || month == "10" || month == "11" || month == "12" {
		return fmt.Sprintf("%s-%s-%s", season, month, day)
	} else {
		return fmt.Sprintf("%s-%s-%s", seasonPlus, month, day)
	}
}

func ToDigitDateFormat(season, strDate string) string {
	dateElements := strings.Split(strDate, " ")
	re := regexp.MustCompile(`[^0-9]`)
	day, month := re.ReplaceAllString(dateElements[1], ""), MonthMap[dateElements[2]]
	return ConstructDate(season, day, month)
}

func GetIdGeneric(url, pattern string) string {
	re := regexp.MustCompile(pattern)
	res := re.FindStringSubmatch(url)
	if len(res) > 1 {
		return res[1]
	} else {
		return ""
	}
}

func GetLeagueIdAndSeasonFromMatchUrl(matchUrl string) (string, string) {
	re := regexp.MustCompile(`statszone/(\d+)-(\d+)/.*`)
	res := re.FindStringSubmatch(matchUrl)
	if len(res) > 2 {
		return res[1], res[2]
	} else {
		return "", ""
	}
}

func GetIdFromMatchUrl(matchUrl string) string {
	return GetIdGeneric(matchUrl, `statszone/.*/matches/(\d+)`)
}

func GetIdFromPlayerStatsUrl(playerStatsUrl string) string {
	return GetIdGeneric(playerStatsUrl, `statszone/.*/matches/.*/player-stats/(\d+).*`)
}

func GetEventTime(eventTimeStr string) (string, string) {
	re := regexp.MustCompile(`timer-(\d)-(\d+)`)
	res := re.FindStringSubmatch(eventTimeStr)
	return res[1], res[2]
}

func GetPos(s *goquery.Selection, attr string) float64 {
	pos := s.AttrOr(attr, "-1.0")
	flt_pos, err := strconv.ParseFloat(pos, 64)
	if err != nil {
		log.Fatal(err)
	}
	return flt_pos
}

func GetSinglePoint(s *goquery.Selection) Point {
	return Point{x: GetPos(s, "x"), y: GetPos(s, "y")}
}

func GetStartEndPoints(s *goquery.Selection) (p1, p2 Point) {
	p1 = Point{x: GetPos(s, "x1"), y: GetPos(s, "y1")}
	p2 = Point{x: GetPos(s, "x2"), y: GetPos(s, "y2")}
	return
}

func CrawlMatch(matches *[]Match, date string, matchRow *goquery.Selection) {
	time := matchRow.Find(".time").Text()
	homeTeam := matchRow.Find(".home-team").Text()
	awayTeam := matchRow.Find(".away-team").Text()
	scores := strings.Split(matchRow.Find(".score").Text(), " - ")
	url, _ := matchRow.Find(".link-to-match a").Attr("href")
	leagueId, season := GetLeagueIdAndSeasonFromMatchUrl(url)

	match := Match{
		id:        GetIdFromMatchUrl(url),
		leagueId:  leagueId,
		season:    season,
		date:      date,
		time:      time,
		homeTeam:  homeTeam,
		awayTeam:  awayTeam,
		homeScore: scores[0],
		awayScore: scores[1],
		url:       PREFIX + url + "/player-stats#tabs-wrapper-anchor"}

	*matches = append(*matches, match)
}

func CrawlMatchByLeague(matches *[]Match, date string, leagueTable *goquery.Selection) {
	leagueTable.Find("tbody .link").Each(func(i int, s *goquery.Selection) {
		CrawlMatch(matches, date, s)
	})
}

func CrawlMatchesOfDay(matches *[]Match, date string) {
	seedUrl := "http://www.fourfourtwo.com/statszone?date_req=" + date
	doc, err := goquery.NewDocument(seedUrl)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find(".match-table").Each(func(i int, s *goquery.Selection) {
		CrawlMatchByLeague(matches, date, s)
	})
}

func CrawlMatchesOfSeason(matches *[]Match, season, leagueId string) {
	seasonResultsUrl := fmt.Sprintf("%s/statszone/results/%s-%s", PREFIX, leagueId, season)
	doc, err := goquery.NewDocument(seasonResultsUrl)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find(".match-table").Each(func(i int, s1 *goquery.Selection) {
		date := s1.Find("caption span").Text()
		s1.Find("tbody .link").Each(func(i int, s2 *goquery.Selection) {
			CrawlMatch(matches, ToDigitDateFormat(season, date), s2)
		})
	})
}

func CrawlPlayerStatsOfMatch(match *Match) []PlayerStats {
	doc, err := goquery.NewDocument(match.url)
	if err != nil {
		log.Fatal(err)
	}
	playerStatsArray := make([]PlayerStats, 0)

	// crawl starting sqaud
	doc.Find(".lineup").Each(func(i int, s *goquery.Selection) {
		playerStatsUrl, _ := s.Find("span a").Attr("href")
		emptyEventArray := make([]PlayerEvents, 0)
		playerStats := PlayerStats{
			matchId:  match.id,
			playerId: GetIdFromPlayerStatsUrl(playerStatsUrl),
			url:      PREFIX + playerStatsUrl,
			events:   &emptyEventArray}
		playerStatsArray = append(playerStatsArray, playerStats)
	})

	// crawl substitution
	doc.Find("#substitutes .subs li").Each(func(i int, s *goquery.Selection) {
		playerStatsUrl, exists := s.Find("div ul").Find(".first").Find("a").Attr("href")
		if exists {
			emptyEventArray := make([]PlayerEvents, 0)
			playerStats := PlayerStats{
				matchId:  match.id,
				playerId: GetIdFromPlayerStatsUrl(playerStatsUrl),
				url:      PREFIX + playerStatsUrl,
				events:   &emptyEventArray}
			playerStatsArray = append(playerStatsArray, playerStats)
		}
	})

	return playerStatsArray
}

func CrawlPlayerRawEvents(playerStats *PlayerStats) {
	doc, err := goquery.NewDocument(playerStats.url)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find(".pitch-object").Each(func(i int, s *goquery.Selection) {
		class, _ := s.Attr("class")
		half, minute := GetEventTime(class)
		markerEnd, hasDirection := s.Attr("marker-end")
		startPoint, endPoint := Point{-1.0, -1.0}, Point{-1.0, -1.0}
		eventType := "unknown"

		if hasDirection {
			eventType = markerEnd
			startPoint, endPoint = GetStartEndPoints(s)
		} else {
			picUrl, _ := s.Attr("href")
			eventType = picUrl
			startPoint = GetSinglePoint(s)
			endPoint = startPoint
		}

		event := PlayerEvents{
			half:       half,
			minute:     minute,
			eventType:  eventType,
			startPoint: startPoint,
			endPoint:   endPoint}

		*playerStats.events = append(*playerStats.events, event)
	})
}

func main() {
	matches := make([]Match, 0)
	//CrawlMatchesOfSeason(&matches, "2015", "23")
	//for _, m := range matches {
	//	fmt.Printf("[%s] %s\n", m.date, m.url)
	//}

	CrawlMatchesOfDay(&matches, "2016-09-10")
	for _, m := range matches {
		fmt.Printf("[%s] %s\n", m.id, m.url)
	}

	playerStatsArray := CrawlPlayerStatsOfMatch(&matches[0])
	for _, p := range playerStatsArray {
		fmt.Printf("[%s] %s\n", p.playerId, p.url)
	}

	CrawlPlayerRawEvents(&playerStatsArray[6])
	for _, e := range *playerStatsArray[6].events {
		fmt.Printf("[%s-%s] %s, (%.2f, %.2f) -> (%.2f, %.2f)\n", e.half, e.minute, e.eventType, e.startPoint.x, e.startPoint.y, e.endPoint.x, e.endPoint.y)
	}
}
