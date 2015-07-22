package pusher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var HttpClient = http.Client{}

const AuthVersion = "1.0"

type Client struct {
	appid, key, secret string
	secure             bool
	Host               string
	Scheme             string
}

type Payload struct {
	Name     string   `json:"name"`
	Channels []string `json:"channels"`
	Data     string   `json:"data"`
}

type ChannelList struct {
	List map[string]ChannelInfo `json:"channels"`
}

func (c *ChannelList) String() string {
	format := "[channel count: %d, list: %+v]"
	return fmt.Sprintf(format, len(c.List), c.List)
}

type ChannelInfo struct {
	UserCount int `json:"user_count"`
}

type UserList struct {
	List []UserInfo `json:"users"`
}

type UserInfo struct {
	Id int `json:"id"`
}

type Channel struct {
	Name              string
	Occupied          bool `json:"occupied"`
	UserCount         int  `json:"user_count",omitempty`
	SubscriptionCount int  `json:"subscription_count",omitempty`
}

func (c *Channel) String() string {
	format := "[name: %s, occupied: %t, user count: %d, subscription count: %d]"
	return fmt.Sprintf(format, c.Name, c.Occupied, c.UserCount, c.SubscriptionCount)
}

func NewClient(appid, key, secret string) *Client {
	return &Client{
		appid:  appid,
		key:    key,
		secret: secret,
		Host:   "api.pusherapp.com",
		Scheme: "http",
	}
}

func (c *Client) Publish(data, event string, channels ...string) error {
	timestamp := c.stringTimestamp()

	content, err := c.jsonifyData(data, event, channels)
	if err != nil {
		return fmt.Errorf("pusher: Publish failed: %s", err)
	}

	signature := Signature{c.key, c.secret, "POST", c.publishPath(), timestamp, AuthVersion, content, nil}

	err = c.post(content, c.fullUrl(c.publishPath()), signature.EncodedQuery())

	return err
}

func (c *Client) AllChannels() (*ChannelList, error) {
	return c.Channels(nil)
}

func (c *Client) Channels(queryParameters map[string]string) (*ChannelList, error) {
	timestamp := c.stringTimestamp()

	signature := Signature{c.key, c.secret, "GET", c.channelsPath(), timestamp, AuthVersion, nil, queryParameters}

	body, err := c.get(c.fullUrl(c.channelsPath()), signature.EncodedQuery())
	if err != nil {
		return nil, err
	}

	var channels *ChannelList
	err = c.parseResponse(body, &channels)

	if err != nil {
		return nil, fmt.Errorf("pusher: Channels failed: %s", err)
	}

	return channels, nil
}

func (c *Client) Channel(name string, queryParameters map[string]string) (*Channel, error) {
	timestamp := c.stringTimestamp()

	urlPath := c.channelPath(name)

	signature := Signature{c.key, c.secret, "GET", urlPath, timestamp, AuthVersion, nil, queryParameters}

	body, err := c.get(c.fullUrl(urlPath), signature.EncodedQuery())
	if err != nil {
		return nil, err
	}

	var channel *Channel
	err = c.parseResponse(body, &channel)

	if err != nil {
		return nil, fmt.Errorf("pusher: Channel failed: %s", err)
	}

	channel.Name = name

	return channel, nil
}

func (c *Client) Users(channelName string) (*UserList, error) {
	timestamp := c.stringTimestamp()

	signature := Signature{c.key, c.secret, "GET", c.usersPath(channelName), timestamp, AuthVersion, nil, nil}

	body, err := c.get(c.fullUrl(c.usersPath(channelName)), signature.EncodedQuery())
	if err != nil {
		return nil, err
	}
	fmt.Println(body)

	var users *UserList
	err = c.parseResponse(body, &users)

	if err != nil {
		return nil, fmt.Errorf("pusher: Users failed: %s", err)
	}

	return users, nil
}

func (c *Client) post(content []byte, fullUrl string, query string) error {
	buffer := bytes.NewBuffer(content)

	postUrl, err := url.Parse(fullUrl)
	if err != nil {
		return err
	}

	postUrl.Scheme = c.Scheme
	postUrl.RawQuery = query

	resp, err := HttpClient.Post(postUrl.String(), "application/json", buffer)
	if err != nil {
		return fmt.Errorf("pusher: POST failed: %s", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("pusher: POST failed: %s", b)
	}

	return nil
}

func (c *Client) get(fullUrl string, query string) (string, error) {
	getUrl, err := url.Parse(fullUrl)
	if err != nil {
		return "", fmt.Errorf("pusher: GET failed: %s", err)
	}

	getUrl.Scheme = c.Scheme
	getUrl.RawQuery = query

	resp, err := HttpClient.Get(getUrl.String())
	if err != nil {
		return "", fmt.Errorf("pusher: GET failed: %s", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("pusher: GET failed: %s", b)
	}

	fullBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", fmt.Errorf("pusher: GET failed: %s", err)
	}

	return string(fullBody), nil
}

func (c *Client) jsonifyData(data, event string, channels []string) ([]byte, error) {
	content := Payload{event, channels, data}
	b, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (c *Client) parseResponse(body string, response interface{}) error {
	err := json.Unmarshal([]byte(body), &response)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) publishPath() string {
	return fmt.Sprintf("/apps/%s/events", c.appid)
}

func (c *Client) channelsPath() string {
	return fmt.Sprintf("/apps/%s/channels", c.appid)
}

func (c *Client) channelPath(name string) string {
	return fmt.Sprintf("/apps/%s/channels/%s", c.appid, name)
}

func (c *Client) usersPath(channelName string) string {
	return fmt.Sprintf("/apps/%s/channels/%s/users", c.appid, channelName)
}

func (c *Client) fullUrl(path string) string {
	return fmt.Sprintf("http://%s%s", c.Host, path)
}

func (c *Client) stringTimestamp() string {
	t := time.Now()
	return strconv.FormatInt(t.Unix(), 10)
}
