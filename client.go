package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/buger/jsonparser"
)

const wikiURL string = "https://wiki.teamfortress.com/w/api.php"

// a fazer: estudar o que raios esses métodos precisam fazer
// https://golang.org/pkg/net/http/cookiejar/#PublicSuffixList
type suffixList struct {
}

func (s suffixList) PublicSuffix(domain string) string {

	return ""
}

func (s suffixList) String() string {
	return ""
}

type wikiClient struct {
	username string
	password string
	client   *http.Client
	token    string
	channel  chan []byte
}

type wikiPage struct {
	namespace int64
	article   string
}

func wiki(username string, password string) *wikiClient {
	suffixList := suffixList{}
	cookieOptions := cookiejar.Options{PublicSuffixList: suffixList}
	cookieJar, _ := cookiejar.New(&cookieOptions)
	webClient := http.Client{Jar: cookieJar, Timeout: time.Second * 60}
	token := getToken(&webClient, "login")

	parameters := fmt.Sprintf("?action=login&lgname=%s&lgpassword=%s&lgtoken=%s&format=json",
		username, password, token)
	req, err := http.NewRequest("POST", wikiURL+parameters, nil)
	if err != nil {
		log.Panicln(err)
	}

	_, err = webClient.Do(req)
	if err != nil {
		log.Panicln(err)
	}

	client := wikiClient{username, password, &webClient, token,
		make(chan []byte)}

	defer client.requestLoop()

	return &client
}

func getToken(client *http.Client, tokenType string) string {
	parameters := fmt.Sprintf(`?action=query&meta=tokens&type=%s&format=json`,
		tokenType)
	req, err := http.NewRequest("POST", wikiURL+parameters, nil)
	if err != nil {
		log.Panicln(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panicln(err)
	}

	str, err := jsonparser.GetString(bytes, "query", "tokens",
		fmt.Sprintf("%stoken", tokenType))
	if err != nil {
		log.Panicln(err)
	}
	return str
}

func (w *wikiClient) requestLoop() {
	go func() {
		for {
			request := <-w.channel
			resp, err := w.wikiAPIRequest(request)
			if err != nil {
				log.Printf("[RequestLoop] Error on API request: %s", err)
				w.channel <- make([]byte, 0)
			} else {
				w.channel <- resp
			}
			time.Sleep(time.Second)
		}
	}()
}

func (w *wikiClient) wikiAPIRequest(parameters []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", wikiURL+string(parameters), nil)
	if err != nil {
		return nil, err
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (w *wikiClient) doRequest(parameters string) []byte {
	w.channel <- []byte(parameters)
	return <-w.channel
}

type topic struct {
	Title    string
	Messages []string
}

func (w *wikiClient) getAllPages(namespace int) []string {
	cont := ""

	var flowPages []string

	for {
		params := fmt.Sprintf(
			"?action=query&generator=allpages&gapcontinue=%s&gaplimit=max&gapnamespace=%d&prop=info&format=json",
			cont, namespace)

		request := w.doRequest(params)

		if len(request) == 0 {
			panic("API error at GetAllPages")
		}
		cont, _ = jsonparser.GetString(request, "continue", "gapcontinue")

		pages, _, _, err := jsonparser.Get(request, "query", "pages")

		if err != nil {
			log.Fatalf("Error getting allpages: %s\n", err)
		}

		callback := func(_ []byte, data []byte, _ jsonparser.ValueType, _ int) error {
			contentmodel, err := jsonparser.GetString(data, "contentmodel")

			if err != nil {
				return err
			}

			if contentmodel == "flow-board" {
				title, err := jsonparser.GetString(data, "title")
				if err != nil {
					return err
				}
				flowPages = append(flowPages, title)
			}
			return nil
		}

		jsonparser.ObjectEach(pages, callback)

		if cont == "" {
			break
		}
	}

	return flowPages
}

func (w *wikiClient) getTopicList(page string) ([]topic, error) {
	params := fmt.Sprintf(
		"?action=flow&submodule=view-topiclist&page=%s&vtlsortby=newest&vtloffset-dir=fwd&format=json",
		strings.ReplaceAll(url.PathEscape(page), "&", "%26"))

	request := w.doRequest(params)
	if len(request) == 0 {
		return nil, fmt.Errorf("API error at getTopicList")
	}

	revisions, _, _, err := jsonparser.Get(request, "flow", "view-topiclist",
		"result", "topiclist", "revisions")

	if err != nil {
		return nil, fmt.Errorf("Error parsing json flow: %s\nRaw API response:\n%s",
			err, string(request))
	}

	pageTopics := make([]topic, 0)

	currentTopic := new(topic)

	// for each
	callback := func(_ []byte, data []byte, _ jsonparser.ValueType, _ int) error {
		changeType, err := jsonparser.GetString(data, "changeType")

		if err != nil {
			return fmt.Errorf("Error getting changeType:\n%s", err)
		}

		switch changeType {
		case "new-post", "edit-title", "lock-topic":
			if currentTopic.Title != "" {
				pageTopics = append(pageTopics, *currentTopic)
				currentTopic = new(topic)
			}
			title, err := jsonparser.GetString(data, "content", "content")
			if err != nil {
				return fmt.Errorf("Error getting %s content:\n%s",
					changeType, err)
			}
			currentTopic.Title = title

		case "edit-post", "reply":
			message, err := jsonparser.GetString(data, "content", "content")
			if err != nil {
				return fmt.Errorf("Error getting %s content:\n%s",
					changeType, err)
			}

			timestamp, err := jsonparser.GetString(data, "timestamp")
			if err != nil {
				return fmt.Errorf("Error getting timestamp:\n%s", err)
			}

			format := "20060102150405"
			prettyFormat := "15:04, 2 January 2006 (MST)"

			timeParse, err := time.Parse(format, timestamp)
			if err != nil {
				return fmt.Errorf("Error parsing timestamp:\n%s", err)
			}

			timeReadable := timeParse.Format(prettyFormat)

			author, err := jsonparser.GetString(data, "creator", "name")
			if err != nil {
				return fmt.Errorf("Error getting creator name:\n%s", err)
			}

			authorLink := fmt.Sprintf("[[User:%[1]s|%[1]s]] ([[User talk:%[1]s|talk]])",
				author)

			formatMessage := fmt.Sprintf("%s %s %s", message, authorLink,
				timeReadable)
			currentTopic.Messages = append(currentTopic.Messages, formatMessage)
		}

		return nil
	}

	// this is a hacky way to make it ignore empty revisions
	if len(revisions) == 2 {
		return pageTopics, nil
	}

	err = jsonparser.ObjectEach(revisions, callback)

	pageTopics = append(pageTopics, *currentTopic)
	if err != nil {
		return nil, fmt.Errorf("Error parsing json topics: %s\nRaw API revisions:\n%s",
			err, string(revisions))
	}

	for i, j := 0, len(pageTopics)-1; i < j; i, j = i+1, j-1 {
		pageTopics[i], pageTopics[j] = pageTopics[j], pageTopics[i]
	}

	return pageTopics, nil
}

func (t topic) formatTopic() string {
	for i := 1; i < len(t.Messages); i++ {
		// make an outdent after 7 colons
		j := i % 7

		if j == 0 {
			t.Messages[i] = "{{outdent|7}}" + t.Messages[i]
		} else {
			colons := strings.Repeat(":", j)
			t.Messages[i] = colons + strings.ReplaceAll(t.Messages[i], "\n",
				fmt.Sprintf("\n%s", colons))
		}
	}

	return fmt.Sprintf("== %s ==\n%s", t.Title,
		strings.Join(t.Messages, "\n\n"))
}

func (w wikiClient) formatFlow(page string) string {
	topics, err := w.getTopicList(page)

	if err != nil {
		log.Fatalf("Error getting topics from %s:\n%s", page, err)
	}

	topicsFormatted := make([]string, len(topics))

	for i, topic := range topics {
		topicsFormatted[i] = topic.formatTopic()
	}

	return strings.Join(topicsFormatted, "\n\n")
}
