// API should support the Eureka REST operations specified here: https://github.com/netflix/eureka/wiki/eureka-rest-operations

package eurekaapi

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout    = 15 * time.Second
	xmlContentType    = "application/xml"
	xmlAccept         = "application/xml"
	defaultBasePath   = "/eureka/v2"
	UP                = "UP"
	DOWN              = "DOWN"
	STARTING          = "STARTING"
	DefaultDataCenter = "MyOwn"
)

type EurekaAPI interface {
	// Register new application instance: POST /apps/{appID}
	RegisterInstance(ctx context.Context, appID string, inst *Instance) error
	// De-register application instance: DELETE /apps/{appID}/{instanceID}
	UnregisterInstance(ctx context.Context, appID, instanceID string) error
	// Heartbeat: PUT /apps/{appID}/{instanceID}
	Heartbeat(ctx context.Context, appID, instanceID string) (exists bool, err error)
	// Query registry: GET /apps
	GetAllApplications(ctx context.Context) (Applications, error)
	// Query app: GET /apps/{appID}
	GetApplication(ctx context.Context, appID string) (Application, error)
	// Query app/instance: GET /apps/{appID}/{instanceID}
	GetInstance(ctx context.Context, appID, instanceID string) (Instance, error)
	// Query by vip/svip: GET /vips/{vip}, /svips/{svip}
	GetByVIP(ctx context.Context, vip string) (Applications, error)
	GetBySecureVIP(ctx context.Context, svip string) (Applications, error)
	// Status override: OUT_OF_SERVICE/UP
	SetStatus(ctx context.Context, appID, instanceID, status string) error
	ClearStatusOverride(ctx context.Context, appID, instanceID string, suggestedFallback string) error
	// Update metadata: PUT /apps/{appID}/{instanceID}/metadata?key=value
	UpdateMetadata(ctx context.Context, appID, instanceID string, kv map[string]string) error
}

type EurekaAPIClient struct {
	client   *http.Client
	baseURLs []string // Use multiple URLs for failover
}

func NewEurekaAPIClient(baseURLs ...string) (EurekaAPI, error) {
	if len(baseURLs) == 0 {
		return nil, errors.New("at least one Eureka base URL is required")
	}
	norm := make([]string, 0, len(baseURLs))
	for _, u := range baseURLs {
		nu, err := normalizeBaseURL(u)
		if err != nil {
			return nil, fmt.Errorf("invalid base URL %q: %w", u, err)
		}
		norm = append(norm, nu)
	}
	return &EurekaAPIClient{
		client: &http.Client{
			Timeout: defaultTimeout,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		baseURLs: norm,
	}, nil
}

func (c *EurekaAPIClient) WrapTransport(wrap func(http.RoundTripper) http.RoundTripper) {
	if wrap == nil {
		return
	}
	if c.client == nil || c.client.Transport == nil {
		return
	}
	c.client.Transport = wrap(c.client.Transport)
}

// ---------- XML Models ----------

type Instance struct {
	XMLName                 xml.Name   `xml:"instance"`
	HostName                string     `xml:"hostName"`
	App                     string     `xml:"app"`
	IPAddr                  string     `xml:"ipAddr"`
	VipAddress              string     `xml:"vipAddress,omitempty"`
	SecureVipAddress        string     `xml:"secureVipAddress,omitempty"`
	Status                  string     `xml:"status"`
	Port                    *Port      `xml:"port,omitempty"`
	SecurePort              *Port      `xml:"securePort,omitempty"`
	HomePageURL             string     `xml:"homePageUrl,omitempty"`
	StatusPageURL           string     `xml:"statusPageUrl,omitempty"`
	HealthCheckURL          string     `xml:"healthCheckUrl,omitempty"`
	DataCenterInfo          DataCenter `xml:"dataCenterInfo"`
	LeaseInfo               *LeaseInfo `xml:"leaseInfo,omitempty"`
	Metadata                *Metadata  `xml:"metadata,omitempty"`
	InstanceID              string     `xml:"instanceId,omitempty"`
	OverriddenStatus        string     `xml:"overriddenstatus,omitempty"`
	IsCoordinatingDiscovery string     `xml:"isCoordinatingDiscoveryServer,omitempty"`
	LastUpdatedTimestamp    string     `xml:"lastUpdatedTimestamp,omitempty"`
	LastDirtyTimestamp      string     `xml:"lastDirtyTimestamp,omitempty"`
	ActionType              string     `xml:"actionType,omitempty"`
	CountryID               string     `xml:"countryId,omitempty"`
}

type Port struct {
	Enabled bool `xml:"enabled,attr"`
	Value   int  `xml:",chardata"`
}

type DataCenter struct {
	XMLName xml.Name `xml:"dataCenterInfo"`
	Name    string   `xml:"name"` // "MyOwn" or "Amazon"
}

type LeaseInfo struct {
	EvictionDurationInSecs uint `xml:"evictionDurationInSecs,omitempty"`
}

type Metadata struct {
	Entries []MetaEntry `xml:",any"`
}

type MetaEntry struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type Applications struct {
	XMLName       xml.Name      `xml:"applications"`
	VersionsDelta string        `xml:"versions__delta,omitempty"`
	AppsHashCode  string        `xml:"apps__hashcode,omitempty"`
	Application   []Application `xml:"application"`
}

type Application struct {
	XMLName  xml.Name   `xml:"application"`
	Name     string     `xml:"name"`
	Instance []Instance `xml:"instance"`
}

// ---------- Util ----------

func (c *EurekaAPIClient) doRequestWithFailOver(doRequest func(baseURL string) (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	for _, baseURL := range c.baseURLs {
		resp, err := doRequest(baseURL)
		if err == nil {
			return resp, nil
		}
		lastErr = fmt.Errorf("request to %s failed: %w", baseURL, err)
	}
	return nil, lastErr
}

// ---------- Requests ----------

func (c *EurekaAPIClient) RegisterInstance(ctx context.Context, appID string, inst *Instance) error {
	body, err := xml.Marshal(inst)
	if err != nil {
		return fmt.Errorf("failed to marshal instance: %w", err)
	}

	doRequest := func(baseURL string) (*http.Response, error) {
		log.Printf("%s", fmt.Sprintf("%s/apps/%s", baseURL, appID))

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/apps/%s", baseURL, appID), strings.NewReader(string(body)))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", xmlContentType)
		req.Header.Set("Accept", xmlAccept)

		return c.client.Do(req)
	}

	resp, err := c.doRequestWithFailOver(doRequest)
	if err != nil {
		return fmt.Errorf("failed to register instance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected response status: %s", resp.Status)
	}
	return nil
}

func (c *EurekaAPIClient) Heartbeat(ctx context.Context, appID, instanceID string) (bool, error) {
	doRequest := func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/apps/%s/%s", baseURL, appID, instanceID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create heartbeat request: %w", err)
		}
		req.Header.Set("Accept", xmlAccept)

		return c.client.Do(req)
	}

	resp, err := c.doRequestWithFailOver(doRequest)
	if err != nil {
		return false, fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	} else if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected response status for heartbeat: %s", resp.Status)
	}
	return true, nil
}

func (c *EurekaAPIClient) GetAllApplications(ctx context.Context) (Applications, error) {
	doRequest := func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/apps", baseURL), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for all applications: %w", err)
		}
		req.Header.Set("Accept", xmlAccept)

		return c.client.Do(req)
	}

	resp, err := c.doRequestWithFailOver(doRequest)
	if err != nil {
		return Applications{}, fmt.Errorf("failed to get all applications: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Applications{}, fmt.Errorf("unexpected response status for all applications: %s", resp.Status)
	}

	var apps Applications
	if err := xml.NewDecoder(resp.Body).Decode(&apps); err != nil {
		return Applications{}, fmt.Errorf("failed to decode applications response: %w", err)
	}
	return apps, nil
}

func (c *EurekaAPIClient) GetApplication(ctx context.Context, appID string) (Application, error) {
	doRequest := func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/apps/%s", baseURL, appID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for application %s: %w", appID, err)
		}
		req.Header.Set("Accept", xmlAccept)

		return c.client.Do(req)
	}

	resp, err := c.doRequestWithFailOver(doRequest)
	if err != nil {
		return Application{}, fmt.Errorf("failed to get application %s: %w", appID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Application{}, fmt.Errorf("unexpected response status for application %s: %s", appID, resp.Status)
	}

	var app Application
	if err := xml.NewDecoder(resp.Body).Decode(&app); err != nil {
		return Application{}, fmt.Errorf("failed to decode application response: %w", err)
	}
	return app, nil
}

func (c *EurekaAPIClient) GetInstance(ctx context.Context, appID, instanceID string) (Instance, error) {
	doRequest := func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/apps/%s/%s", baseURL, appID, instanceID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for instance %s of application %s: %w", instanceID, appID, err)
		}
		req.Header.Set("Accept", xmlAccept)

		return c.client.Do(req)
	}

	resp, err := c.doRequestWithFailOver(doRequest)
	if err != nil {
		return Instance{}, fmt.Errorf("failed to get instance %s of application %s: %w", instanceID, appID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Instance{}, fmt.Errorf("unexpected response status for instance %s of application %s: %s", instanceID, appID, resp.Status)
	}

	var inst Instance
	if err := xml.NewDecoder(resp.Body).Decode(&inst); err != nil {
		return Instance{}, fmt.Errorf("failed to decode instance response: %w", err)
	}
	return inst, nil
}

func (c *EurekaAPIClient) GetByVIP(ctx context.Context, vip string) (Applications, error) {
	doRequest := func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/vips/%s", baseURL, vip), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for VIP %s: %w", vip, err)
		}
		req.Header.Set("Accept", xmlAccept)

		return c.client.Do(req)
	}

	resp, err := c.doRequestWithFailOver(doRequest)
	if err != nil {
		return Applications{}, fmt.Errorf("failed to get by VIP %s: %w", vip, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Applications{}, fmt.Errorf("unexpected response status for VIP %s: %s", vip, resp.Status)
	}

	var apps Applications
	if err := xml.NewDecoder(resp.Body).Decode(&apps); err != nil {
		return Applications{}, fmt.Errorf("failed to decode VIP response: %w", err)
	}
	return apps, nil
}

func (c *EurekaAPIClient) GetBySecureVIP(ctx context.Context, svip string) (Applications, error) {
	doRequest := func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/svips/%s", baseURL, svip), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for secure VIP %s: %w", svip, err)
		}
		req.Header.Set("Accept", xmlAccept)

		return c.client.Do(req)
	}

	resp, err := c.doRequestWithFailOver(doRequest)
	if err != nil {
		return Applications{}, fmt.Errorf("failed to get by secure VIP %s: %w", svip, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Applications{}, fmt.Errorf("unexpected response status for secure VIP %s: %s", svip, resp.Status)
	}

	var apps Applications
	if err := xml.NewDecoder(resp.Body).Decode(&apps); err != nil {
		return Applications{}, fmt.Errorf("failed to decode secure VIP response: %w", err)
	}
	return apps, nil
}

func (c *EurekaAPIClient) SetStatus(ctx context.Context, appID, instanceID, status string) error {
	doRequest := func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/apps/%s/%s/status?value=%s", baseURL, appID, instanceID, status), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request to set status for instance %s of application %s: %w", instanceID, appID, err)
		}
		req.Header.Set("Accept", xmlAccept)

		return c.client.Do(req)
	}

	resp, err := c.doRequestWithFailOver(doRequest)
	if err != nil {
		return fmt.Errorf("failed to set status for instance %s of application %s: %w", instanceID, appID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected response status when setting status for instance %s of application %s: %s", instanceID, appID, resp.Status)
	}
	return nil
}

func (c *EurekaAPIClient) ClearStatusOverride(ctx context.Context, appID, instanceID string, suggestedFallback string) error {
	doRequest := func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%s/apps/%s/%s/status?value=%s", baseURL, appID, instanceID, suggestedFallback), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request to clear status override for instance %s of application %s: %w", instanceID, appID, err)
		}
		req.Header.Set("Accept", xmlAccept)

		return c.client.Do(req)
	}

	resp, err := c.doRequestWithFailOver(doRequest)
	if err != nil {
		return fmt.Errorf("failed to clear status override for instance %s of application %s: %w", instanceID, appID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected response status when clearing status override for instance %s of application %s: %s", instanceID, appID, resp.Status)
	}
	return nil
}

func (c *EurekaAPIClient) UpdateMetadata(ctx context.Context, appID, instanceID string, kv map[string]string) error {
	if len(kv) == 0 {
		return errors.New("metadata map cannot be empty")
	}

	query := ""
	for k, v := range kv {
		if query != "" {
			query += "&"
		}
		query += fmt.Sprintf("%s=%s", k, v)
	}

	doRequest := func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/apps/%s/%s/metadata?%s", baseURL, appID, instanceID, query), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request to update metadata for instance %s of application %s: %w", instanceID, appID, err)
		}
		req.Header.Set("Accept", xmlAccept)

		return c.client.Do(req)
	}

	resp, err := c.doRequestWithFailOver(doRequest)
	if err != nil {
		return fmt.Errorf("failed to update metadata for instance %s of application %s: %w", instanceID, appID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected response status when updating metadata for instance %s of application %s: %s", instanceID, appID, resp.Status)
	}
	return nil
}

func (c *EurekaAPIClient) UnregisterInstance(ctx context.Context, appID, instanceID string) error {
	doRequest := func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%s/apps/%s/%s", baseURL, appID, instanceID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request to unregister instance %s of application %s: %w", instanceID, appID, err)
		}
		req.Header.Set("Accept", xmlAccept)

		return c.client.Do(req)
	}

	resp, err := c.doRequestWithFailOver(doRequest)
	if err != nil {
		return fmt.Errorf("failed to unregister instance %s of application %s: %w", instanceID, appID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected response status when unregistering instance %s of application %s: %s", instanceID, appID, resp.Status)
	}
	return nil
}
