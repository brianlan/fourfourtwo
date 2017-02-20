package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/jmoiron/sqlx"
	_ "github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var PREFIX = "http://www.fourfourtwo.com"

//var NUM_MATCH_CRAWLER = 3
var NUM_PLAYER_STATS_CRAWLER = 10

// Match is the overview information about match, not the stats / details in a match.
type Match struct {
	Id           string `db:"id"`
	Season       string `db:"season"`
	MatchDate    string `db:"match_date"`
	MatchTime    string `db:"match_time"`
	LeagueId     string `db:"league_id"`
	HomeTeamName string `db:"home_team_name"`
	AwayTeamName string `db:"away_team_name"`
	HomeScore    string `db:"home_score"`
	AwayScore    string `db:"away_score"`
	Url          string `db:"url"`
	IsCrawled    string `db:"is_crawled"`
}

type League struct {
	Id   string `db:"id"`
	Name string `db:"name"`
}

type Player struct {
	Id   string `db:"id"`
	Name string `db:"name"`
}

type PlayerStats struct {
	Id           int64  `db:id`
	MatchId      string `db:"match_id"`
	TeamName     string `db:"team_name"`
	PlayerId     string `db:"player_id"`
	PlayerName   string `db:"player_name"`
	IsSubstitute string `db:"is_substitute"`
	Url          string `db:"url"`
	Events       *[]PlayerEvent
}

type Point struct {
	x, y float64
}

// If the event has no directions, then the startPoint would store the position of this event, leaving endPoint empty.
// Pitch range is from (57, 58) to (680, 470) in the raw D3 position, need to transform it.
type PlayerEvent struct {
	EventHalf            string `db:"event_half"`
	EventMinute          string `db:"event_minute"`
	EventType            string `db:"event_type"`
	StartPoint, EndPoint Point
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

var EventTypeMap = map[string]string{
	"smallblue":               "pass_success",
	"smallred":                "pass_fail",
	"smallyellow":             "pass_goal_assist",
	"smalldeepskyblue":        "pass_chance_created",
	"bigblue":                 "shot_on_target",
	"bigred":                  "shot_off_target",
	"bigyellow":               "shot_goal",
	"bigdarkgrey":             "shot_blocked",
	"success":                 "take_on_success",
	"fail":                    "take_on_fail",
	"won":                     "aerial_duel_won",
	"lost":                    "aerial_duel_lost",
	"commited":                "foul_commited",
	"suffered":                "foul_suffered",
	"error-leading-goal":      "error_leading_goal",
	"error-leading-shot":      "error_leading_shot",
	"successful_tackle":       "def_tackle_success",
	"failed_tackle":           "def_tackle_fail",
	"successful_clearance":    "def_clearance_success",
	"failed_clearance":        "def_clearance_fail",
	"interceptions":           "def_interception",
	"defensive-ball-recovery": "def_ball_recovery",
	"blocks":                  "def_block_shot",
	"blocks-cross":            "def_block_cross"}

//func Filter(vs []string, f func(string) bool) []string {
//	vsf := make([]string, 0)
//	for _, v := range vs {
//		if f(v) {
//			vsf = append(vsf, v)
//		}
//	}
//	return vsf
//}

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

func ConcurrentCheckMatchExistsInDB(db *sqlx.DB, match *Match, ch chan<- *Match) {
	var count int64
	err := db.Get(&count, fmt.Sprintf(`SELECT count(*) FROM match where url = "%s"`, match.Url))
	fmt.Printf("%s (%d)\n", match.Url, count)
	if err != nil {
		log.Fatal(err)
	}
	if count == 0 {
		ch <- match
	}
}

func CrawlMatch(matches *[]Match, date string, matchRow *goquery.Selection) {
	time := matchRow.Find(".time").Text()
	homeTeam := matchRow.Find(".home-team").Text()
	awayTeam := matchRow.Find(".away-team").Text()
	scores := strings.Split(matchRow.Find(".score").Text(), " - ")
	url, _ := matchRow.Find(".link-to-match a").Attr("href")
	leagueId, season := GetLeagueIdAndSeasonFromMatchUrl(url)

	match := Match{
		Id:           GetIdFromMatchUrl(url),
		LeagueId:     leagueId,
		Season:       season,
		MatchDate:    date,
		MatchTime:    time,
		HomeTeamName: homeTeam,
		AwayTeamName: awayTeam,
		HomeScore:    scores[0],
		AwayScore:    scores[1],
		Url:          PREFIX + url + "/player-stats#tabs-wrapper-anchor"}

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

func ConcurrentProcessMatches(matches *[]Match, db *sqlx.DB, maxPlayerStatsId *int64, ch chan *PlayerStats, ch2 chan *PlayerStats) {
	mch := make(chan *Match)
	go ConcurrentCrawlPlayerStatsOfMatch(db, maxPlayerStatsId, mch, ch, ch2)
	for i, _ := range *matches {
		m := (*matches)[i]
		go ConcurrentCheckMatchExistsInDB(db, &m, mch)
	}
}

func ConcurrentCrawlPlayerEventsOfPlayerStats(db *sqlx.DB, playerStats *PlayerStats, ch chan<- *PlayerStats, ch2 <-chan *PlayerStats) {
	q := `INSERT INTO player_stats (match_id, team_name, player_id, player_name, is_substitute, url)
			VALUES (:match_id, :team_name, :player_id, :player_name, :is_substitute, :url)`
	ch <- playerStats
	finishedPlayerStats := <-ch2

	// Save player events to DB
	for i, _ := range *finishedPlayerStats.Events {
		e := (*finishedPlayerStats.Events)[i]
		go ConcurrentSavePlayerEvents(db, playerStats, &e)
	}

	// Save player stats to DB
	_, err := db.NamedExec(q, *finishedPlayerStats)
	if err != nil {
		log.Fatal(err)
	}
}

func ConcurrentCrawlPlayerStatsOfMatch(db *sqlx.DB, maxPlayerStatsId *int64, mch <-chan *Match, ch chan *PlayerStats, ch2 chan *PlayerStats) {
	mq := `INSERT INTO match (id, season, match_date, match_time, league_id, home_team_name, away_team_name, home_score, away_score, url, is_crawled)
			VALUES (:id, :season, :match_date, :match_time, :league_id, :home_team_name, :away_team_name, :home_score, :away_score, :url, "0")`
	for {
		m := <-mch

		_, err := db.NamedExec(mq, m)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("[%s] %s\n", m.Id, m.Url)
		doc, err := goquery.NewDocument(m.Url)
		if err != nil {
			log.Fatal(err)
		}
		playerStatsArray := make([]PlayerStats, 0)

		// crawl starting sqaud
		doc.Find(".lineup").Each(func(i int, s *goquery.Selection) {
			classText, _ := s.Attr("class")

			var teamName string
			if homeOrAway := classText[7:]; homeOrAway == "home" {
				teamName = m.HomeTeamName
			} else {
				teamName = m.AwayTeamName
			}

			playerStatsUrl, _ := s.Find("span a").Attr("href")
			emptyEventArray := make([]PlayerEvent, 0)
			playerStats := PlayerStats{
				Id:           *maxPlayerStatsId + 1,
				MatchId:      m.Id,
				TeamName:     teamName,
				PlayerId:     GetIdFromPlayerStatsUrl(playerStatsUrl),
				IsSubstitute: "0",
				Url:          PREFIX + playerStatsUrl,
				Events:       &emptyEventArray}
			*maxPlayerStatsId++
			playerStatsArray = append(playerStatsArray, playerStats)
		})

		// crawl substitution
		doc.Find("#substitutes .subs").Each(func(i int, s1 *goquery.Selection) {
			classText, _ := s1.Attr("class")

			var teamName string
			if homeOrAway := classText[:4]; homeOrAway == "home" {
				teamName = m.HomeTeamName
			} else {
				teamName = m.AwayTeamName
			}

			s1.Find("li").Each(func(i int, s2 *goquery.Selection) {
				playerStatsUrl, exists := s2.Find("div ul").Find(".first").Find("a").Attr("href")
				if exists {
					emptyEventArray := make([]PlayerEvent, 0)
					playerStats := PlayerStats{
						Id:           *maxPlayerStatsId + 1,
						MatchId:      m.Id,
						TeamName:     teamName,
						PlayerId:     GetIdFromPlayerStatsUrl(playerStatsUrl),
						IsSubstitute: "1",
						Url:          PREFIX + playerStatsUrl,
						Events:       &emptyEventArray}
					*maxPlayerStatsId++
					playerStatsArray = append(playerStatsArray, playerStats)
				}
			})
		})

		for i, _ := range playerStatsArray {
			ps := playerStatsArray[i]
			ConcurrentCrawlPlayerEventsOfPlayerStats(db, &ps, ch, ch2)
		}

		tx := db.MustBegin()
		db.MustExec(fmt.Sprintf(`UPDATE match SET is_crawled = "1" WHERE id = "%s"`, m.Id))
		tx.Commit()

		time.Sleep(time.Minute * 1)
	}
}

func ConcurrentSavePlayerEvents(db *sqlx.DB, ps *PlayerStats, e *PlayerEvent) {
	q := `INSERT INTO player_event (player_stats_id, event_half, event_minute, event_type, x1, y1, x2, y2) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	fmt.Printf("[%s-%s] %s, (%.2f, %.2f) -> (%.2f, %.2f)\n", e.EventHalf, e.EventMinute, e.EventType, e.StartPoint.x, e.StartPoint.y, e.EndPoint.x, e.EndPoint.y)

	tx := db.MustBegin()
	db.MustExec(q, ps.Id, e.EventHalf, e.EventMinute, e.EventType, e.StartPoint.x, e.StartPoint.y, e.EndPoint.x, e.EndPoint.y)
	tx.Commit()
}

func ConcurrentCrawlPlayerRawEvents(db *sqlx.DB, ch <-chan *PlayerStats, ch2 chan<- *PlayerStats) {
	for {
		playerStats := <-ch
		fmt.Printf("[%s] %s\n", playerStats.PlayerId, playerStats.Url)

		doc, err := goquery.NewDocument(playerStats.Url)
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
				re := regexp.MustCompile(`url\(#(\w+)\)`)
				rawEventType := re.FindStringSubmatch(markerEnd)
				eventType = EventTypeMap[rawEventType[1]]
				startPoint, endPoint = GetStartEndPoints(s)
			} else {
				picUrl, _ := s.Attr("href")
				re := regexp.MustCompile(`/sites/fourfourtwo.com/modules/custom/statzone/files/icons/(\w+)\.png`)
				rawEventType := re.FindStringSubmatch(picUrl)
				eventType = EventTypeMap[rawEventType[1]]
				startPoint = GetSinglePoint(s)
				endPoint = startPoint
			}

			event := PlayerEvent{
				EventHalf:   half,
				EventMinute: minute,
				EventType:   eventType,
				StartPoint:  startPoint,
				EndPoint:    endPoint}

			*playerStats.Events = append(*playerStats.Events, event)
		})

		playerStats.PlayerName = doc.Find("#statzone_player_header h1").Text()
		ch2 <- playerStats

		time.Sleep(time.Second * 1)
	}
}

func main() {
	db, err := sqlx.Connect("sqlite3", "file:fourfourtwo.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	var maxPlayerStatsId int64
	err = db.Get(&maxPlayerStatsId, "SELECT max(id) FROM player_stats")

	matches := make([]Match, 0)
	ch := make(chan *PlayerStats, 10)
	ch2 := make(chan *PlayerStats, 10)

	//CrawlMatchesOfDay(&matches, "2016-09-10")
	CrawlMatchesOfSeason(&matches, "2016", "8")

	tmpMatches := matches[:6]
	go ConcurrentProcessMatches(&tmpMatches, db, &maxPlayerStatsId, ch, ch2)

	for i := 1; i < NUM_PLAYER_STATS_CRAWLER; i++ {
		go ConcurrentCrawlPlayerRawEvents(db, ch, ch2)
	}

	var input string
	fmt.Scanln(&input)
}
