package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Songmu/prompter"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const ScreenName string = "JIDORI_OHASHI"

var TwitterCredentialFile string = os.Getenv("HOME") + "/.TWITTER_CREDENTIALS"

type tweetModel struct {
	Text  string `json:"text"`
	IdStr string `json:"id_str"`
}

func (t *tweetModel) URL() string {
	return "https://twitter.com/" + ScreenName + "/status/" + t.IdStr
}

type tweetViewModel struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

func InitCredentials() {
	consumerKey := prompter.Prompt("Enter your Twitter Application Consumer Key", "")
	consumerSecret := prompter.Prompt("Enter your Twitter Application Consumer Secret", "")
	accessToken := prompter.Prompt("Enter your Twitter Application Access Token", "")
	accessTokenSecret := prompter.Prompt("Enter your Twitter Application Access Token Secret", "")
	bf := bytes.NewBufferString(consumerKey + "\n" + consumerSecret + "\n" + accessToken + "\n" + accessTokenSecret)
	err := ioutil.WriteFile(TwitterCredentialFile, bf.Bytes(), 0600)
	if err != nil {
		fmt.Errorf("%s", err.Error())
		os.Exit(1)
		return
	}
}

func LoadCredentials() (client *twittergo.Client, err error) {
	credentials, err := ioutil.ReadFile(TwitterCredentialFile)
	if err != nil {
		return
	}
	lines := strings.Split(string(credentials), "\n")
	config := &oauth1a.ClientConfig{
		ConsumerKey:    lines[0],
		ConsumerSecret: lines[1],
	}
	user := oauth1a.NewAuthorizedConfig(lines[2], lines[3])
	client = twittergo.NewClient(config, user)
	return
}

func main() {

	if len(os.Args) >= 3 {
		fmt.Println("Invalid argument count")
		os.Exit(1)
	}

	if len(os.Args) == 2 && os.Args[1] == "init" {
		InitCredentials()
	}

	var (
		err     error
		client  *twittergo.Client
		req     *http.Request
		resp    *twittergo.APIResponse
		max_id  uint64
		query   url.Values
		results *twittergo.Timeline
		text    []byte
	)
	if client, err = LoadCredentials(); err != nil {
		fmt.Printf("Could not parse TwitterCredentialFile: %v\n", err)
		os.Exit(1)
	}
	const (
		count   int = 1
		urltmpl     = "/1.1/statuses/user_timeline.json?%v"
		minwait     = time.Duration(10) * time.Second
	)
	query = url.Values{}
	query.Set("count", fmt.Sprintf("%v", count))
	query.Set("screen_name", ScreenName)

	if max_id != 0 {
		query.Set("max_id", fmt.Sprintf("%v", max_id))
	}
	endpoint := fmt.Sprintf(urltmpl, query.Encode())
	if req, err = http.NewRequest("GET", endpoint, nil); err != nil {
		fmt.Printf("Could not parse request: %v\n", err)
		os.Exit(1)
	}
	if resp, err = client.SendRequest(req); err != nil {
		fmt.Printf("Could not send request: %v\n", err)
		os.Exit(1)
	}
	results = &twittergo.Timeline{}
	if err = resp.Parse(results); err != nil {
		if rle, ok := err.(twittergo.RateLimitError); ok {
			dur := rle.Reset.Sub(time.Now()) + time.Second
			if dur < minwait {
				// Don't wait less than minwait.
				dur = minwait
			}
			msg := "Rate limited. Reset at %v. Waiting for %v\n"
			fmt.Printf(msg, rle.Reset, dur)
			os.Exit(-1)
		} else {
			fmt.Printf("Problem parsing response: %v\n", err)
		}
	}
	batch := len(*results)
	if batch == 0 {
		fmt.Printf("No more results, end of timeline.\n")
		os.Exit(-1)
	}
	for _, tweet := range *results {
		if text, err = json.Marshal(tweet); err != nil {
			fmt.Printf("Could not encode Tweet: %v\n", err)
			os.Exit(1)
		}
	}
	t := &tweetModel{}
	if err = json.Unmarshal(text, t); err != nil {
		fmt.Printf("Could not decode Tweet: %v\n", err)
		os.Exit(1)
	}

	if text, err = json.Marshal(tweetViewModel{t.Text, t.URL()}); err != nil {
		fmt.Printf("Could not encode Tweet: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", text)

	//if resp.HasRateLimit() {
	//	fmt.Printf(", %v calls available", resp.RateLimitRemaining())
	//}
}
