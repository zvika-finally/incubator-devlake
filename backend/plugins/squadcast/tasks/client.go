package tasks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type SquadcastIncidentAPIResponse struct {
	Data []SquadcastIncidentAPIItem `json:"data"`
}

type SquadcastIncidentAPIItem struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	Severity   string     `json:"severity"`
	Assignee   string     `json:"assignee"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	URL        string     `json:"url"`
}

type SquadcastClient struct {
	ApiKey  string
	BaseURL string
}

func NewSquadcastClient(apiKey string) *SquadcastClient {
	return &SquadcastClient{
		ApiKey:  apiKey,
		BaseURL: "https://api.squadcast.com/v3",
	}
}

func (c *SquadcastClient) FetchIncidents() ([]SquadcastIncidentAPIItem, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/incidents", c.BaseURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Squadcast API error: %s", string(body))
	}

	var apiResp SquadcastIncidentAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		return nil, err
	}
	return apiResp.Data, nil
}
