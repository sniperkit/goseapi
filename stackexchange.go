// Package stackexchange provides access to the Stack Exchange 2.0 API.
//
// http://api.stackexchange.com/
package goseapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// Root is the Stack Exchange API endpoint.
const Root = "https://api.stackexchange.com/" + Version

// Version is the API version identifier.
const Version = "2.1"

// Well-known Stack Exchange sites
const (
	StackOverflow = "stackoverflow"
)

// Sort orders
const (
	SortActivity     = "activity"
	SortCreationDate = "creation"
	SortHot          = "hot"
	SortWeek         = "week"
	SortMonth        = "month"
	SortScore        = "votes"
)

// API paths
const (
	PathAllAnswers     = "/answers"
	PathAnswers        = "/answers/{ids}"
	PathAnswerComments = "/answers/{ids}/comments"

	PathAllQuestions     = "/questions"
	PathQuestions        = "/questions/{ids}"
	PathQuestionAnswers  = "/questions/{ids}/answers"
	PathQuestionComments = "/questions/{ids}/comments"
)

// Params is common set of arguments that can be sent to an API request.
type Params struct {
	Site string

	Sort     string
	Order    string
	Page     int
	PageSize int

	Filter string

	Tagged string

	// If len(Args) != 0, then the strings will substitute placeholders
	// surrounded by braces in the path.
	Args []string
}

func (p *Params) values() url.Values {
	vals := url.Values{
		"site": {p.Site},
	}
	if p.Sort != "" {
		vals.Set("sort", p.Sort)
	}
	if p.Order != "" {
		vals.Set("order", p.Order)
	}
	if p.Page != 0 {
		vals.Set("page", strconv.Itoa(p.Page))
	}
	if p.PageSize != 0 {
		vals.Set("pagesize", strconv.Itoa(p.PageSize))
	}
	if p.Filter != "" {
		vals.Set("filter", p.Filter)
	}
	if p.Tagged != "" {
		vals.Set("tagged", p.Tagged)
	}
	return vals
}

// DefaultClient uses the default HTTP client and API root.
var DefaultClient *Client = nil

// Do performs an API request using the default client.
func Do(path string, v interface{}, params *Params) (*Wrapper, error) {
	return DefaultClient.Do(path, v, params)
}

// A Client can make API requests.
type Client struct {
	Client *http.Client
	Root   string

	// Pass these fields if you have an OAuth 2.0 application registered with stackapps.com.
	AccessToken string
	Key         string
}

var Verbose bool

// Do performs an API request.
func (c *Client) Do(path string, v interface{}, params *Params) (*Wrapper, error) {
	// Get arguments
	client := http.DefaultClient
	if c != nil && c.Client != nil {
		client = c.Client
	}
	root := Root
	if c != nil && c.Root != "" {
		root = c.Root
	}

	// Build URL parameters
	vals := params.values()
	if c != nil && c.AccessToken != "" {
		vals.Set("access_token", c.AccessToken)
	}
	if c != nil && c.Key != "" {
		vals.Set("key", c.Key)
	}

	req := root + fillPlaceholders(path, params.Args) + "?" + vals.Encode()
	if Verbose {
		fmt.Println(req)
	}
	// Send request
	resp, err := client.Get(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	return parseResponse(resp.Body, v)
}

func fillPlaceholders(s string, args []string) string {
	if len(s) == 0 || len(args) == 0 {
		return s
	}

	b := []byte(s)
	buf := make([]byte, 0, len(b)*2)
	argIdx := 0
	for argIdx < len(args) && len(b) > 0 {
		if i := bytes.IndexByte(b, '{'); i == -1 {
			break
		} else {
			buf, b = append(buf, b[:i]...), b[i:]
			if i := bytes.IndexByte(b, '}'); i == -1 {
				break
			} else {
				buf, b = append(buf, args[argIdx]...), b[i+1:]
				argIdx++
			}
		}
	}
	if len(b) > 0 {
		buf = append(buf, b...)
	}
	return string(buf)
}

// JoinIDs builds a string of semicolon-separated IDs.
func JoinIDs(ids []int) string {
	const bytesPerID = 9
	buf := make([]byte, 0, bytesPerID*len(ids))
	for _, id := range ids {
		if len(buf) > 0 {
			buf = append(buf, ';')
		}
		buf = strconv.AppendInt(buf, int64(id), 10)
	}
	return string(buf)
}

func parseResponse(r io.Reader, v interface{}) (*Wrapper, error) {
	var result struct {
		Items items `json:"items"`

		ErrorID      int    `json:"error_id"`
		ErrorName    string `json:"error_name"`
		ErrorMessage string `json:"error_message"`

		Page     int  `json:"page"`
		PageSize int  `json:"page_size"`
		HasMore  bool `json:"has_more"`

		Backoff        int `json:"backoff"`
		QuotaMax       int `json:"quota_max"`
		QuotaRemaining int `json:"quota_remaining"`

		Total int    `json:"total"`
		Type  string `json:"type"`
	}
	result.Items = items{v}
	err := json.NewDecoder(r).Decode(&result)
	return &Wrapper{
		Error: Error{
			ID:      result.ErrorID,
			Name:    result.ErrorName,
			Message: result.ErrorMessage,
		},
		Page:           result.Page,
		PageSize:       result.PageSize,
		HasMore:        result.HasMore,
		Backoff:        result.Backoff,
		QuotaMax:       result.QuotaMax,
		QuotaRemaining: result.QuotaRemaining,
		Total:          result.Total,
		Type:           result.Type,
	}, err
}

type items struct {
	val interface{}
}

func (i items) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, i.val)
}
