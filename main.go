package main

import (
  "fmt"
  "net/http"
  "net/url"
  "os"
  "time"
  "log"

  "github.com/kurrik/oauth1a"
  "github.com/kurrik/twittergo"
)

const MINWAIT = time.Duration(10) * time.Second

var KleethoVotes int = 0
var WyldVotes int = 0

func LoadCredentials() (client *twittergo.Client, err error) {
  config := &oauth1a.ClientConfig{
    ConsumerKey:    os.Getenv("TWITTER_CONSUMER_KEY"),
    ConsumerSecret: os.Getenv("TWITTER_CONSUMER_SECRET"),
  }
  client = twittergo.NewClient(config, nil)
  return
}

func GetHashtagCount(client *twittergo.Client, hashtag string) int {
  var (
    err     error
    req     *http.Request
    resp    *twittergo.APIResponse
    results *twittergo.SearchResults
    i       int
  )

  query := url.Values{}
  query.Set("q", hashtag)
  query.Set("count", "100")

  i = 1
  for {
    url := fmt.Sprintf("/1.1/search/tweets.json?%v", query.Encode())
    req, err = http.NewRequest("GET", url, nil)
    if err != nil {
      fmt.Printf("Could not parse request: %v\n", err)
      break
    }
    resp, err = client.SendRequest(req)
    if err != nil {
      fmt.Printf("Could not send request: %v\n", err)
      break
    }
    results = &twittergo.SearchResults{}
    if err = resp.Parse(results); err != nil {
      if rle, ok := err.(twittergo.RateLimitError); ok {
        dur := rle.Reset.Sub(time.Now()) + time.Second
        if dur < MINWAIT {
          // Don't wait less than minwait.
          dur = MINWAIT
        }
        msg := "Rate limited. Reset at %v. Waiting for %v\n"
        fmt.Printf(msg, rle.Reset, dur)
        time.Sleep(dur)
        continue // Retry request.
      } else {
        fmt.Printf("Problem parsing response: %v\n", err)
        break
      }
    }

    i = len(results.Statuses())

    if query, err = results.NextQuery(); err != nil {
      break
    }
    if resp.HasRateLimit() {
      fmt.Printf("Rate limit:           %v\n", resp.RateLimit())
      fmt.Printf("Rate limit remaining: %v\n", resp.RateLimitRemaining())
      fmt.Printf("Rate limit reset:     %v\n", resp.RateLimitReset())
    } else {
      fmt.Printf("Could not parse rate limit from response.\n")
    }
  }

  return i
}

func TallyVotes(client *twittergo.Client) {
  for {
    KleethoVotes = GetHashtagCount(client, "DontTellKleetho")
    WyldVotes = GetHashtagCount(client, "DontTellWyld")

    fmt.Printf("DontTellKleetho: %d | DontTellWyld: %d\n", KleethoVotes, WyldVotes)

    time.Sleep(30000 * time.Millisecond)
  }
}

func main() {

  var client *twittergo.Client
  var err error

  if client, err = LoadCredentials(); err != nil {
    fmt.Printf("Could not parse CREDENTIALS file: %v\n", err)
    os.Exit(1)
  }

  go TallyVotes(client)

  fs := http.Dir("public/")
  fileHandler := http.FileServer(fs)
  http.Handle("/", fileHandler)

  http.HandleFunc("/votes", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "[%d, %d]", KleethoVotes, WyldVotes)
  })

  port := os.Getenv("PORT")

  fmt.Println("Serving on Port: " + port)

  addr := fmt.Sprintf(":%s", port)
  log.Fatal(http.ListenAndServe(addr, nil))
}
