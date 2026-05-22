package netatmo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GetMeasureBlock is one contiguous block of measurements as returned by the
// getmeasure endpoint with optimize=false. Each entry in Value contains the
// requested measurement types in the same order as the type= parameter.
type GetMeasureBlock struct {
	BegTime  int64       `json:"beg_time"`
	StepTime int64       `json:"step_time"`
	Value    [][]float64 `json:"value"`
}

// GetMeasureResponse is the response shape of the getmeasure endpoint when
// called with optimize=false.
type GetMeasureResponse struct {
	Body       []GetMeasureBlock `json:"body"`
	Status     string            `json:"status"`
	TimeExec   float64           `json:"time_exec"`
	TimeServer int64             `json:"time_server"`
}

// LatestValues returns the most recent value tuple and its timestamp.
// The values are in the same order as the type= argument passed to GetMeasure.
// Returns ok=false if the response contained no measurement data.
func (r *GetMeasureResponse) LatestValues() (time.Time, []float64, bool) {
	for i := len(r.Body) - 1; i >= 0; i-- {
		block := r.Body[i]
		if len(block.Value) == 0 {
			continue
		}
		lastIdx := len(block.Value) - 1
		ts := block.BegTime + block.StepTime*int64(lastIdx)
		return time.Unix(ts, 0).UTC(), block.Value[lastIdx], true
	}
	return time.Time{}, nil, false
}

// GetMeasure fetches the latest measurement values for the given types from the
// Netatmo getmeasure endpoint. moduleID must be empty when querying the main
// device and the module's _id when querying a sub-module.
func (c *Client) GetMeasure(ctx context.Context, deviceID, moduleID string, types []string, scale string) (*GetMeasureResponse, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}
	if len(types) == 0 {
		return nil, fmt.Errorf("at least one measurement type is required")
	}
	if scale == "" {
		scale = "max"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.netatmo.com/api/getmeasure", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := url.Values{}
	q.Add("access_token", c.accessToken)
	q.Add("device_id", deviceID)
	if moduleID != "" {
		q.Add("module_id", moduleID)
	}
	q.Add("scale", scale)
	q.Add("type", strings.Join(types, ","))
	// A bounded lookback prevents the API from scanning the full retention window
	// when the device went quiet for a while.
	q.Add("date_begin", strconv.FormatInt(time.Now().Add(-1*time.Hour).Unix(), 10))
	q.Add("optimize", "true")
	q.Add("real_time", "true")
	req.URL.RawQuery = q.Encode()

	fmt.Println(req.URL.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch getmeasure from Netatmo: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("netatmo getmeasure returned status %d: %s", resp.StatusCode, string(body))
	}

	var out GetMeasureResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to parse getmeasure response: %w\nResponse:\n%s", err, string(body))
	}

	return &out, nil
}
