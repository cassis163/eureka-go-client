package pkg

import (
	"context"
	"fmt"
	"net"
	"net/http"

	eurekaapi "github.com/cassis163/eureka-go-client/internal/eureka-api"
)

type Client struct {
	appID      string
	host       string
	port       int
	instanceID string

	eurekaAPIClient eurekaapi.EurekaAPI
}

type ClientAPI interface {
    WrapTransport(wrap func(http.RoundTripper) http.RoundTripper)

	RegisterInstance(ctx context.Context, ip net.IP, ttl uint, useSSL bool) (*Instance, error)
	Heartbeat(ctx context.Context) error
	GetAllApplications(ctx context.Context) (eurekaapi.Applications, error)
	UnregisterInstance(ctx context.Context) error
	GetApplication(ctx context.Context) (eurekaapi.Application, error)
	GetInstance(ctx context.Context) (eurekaapi.Instance, error)
	GetByVIP(ctx context.Context, vip string) (eurekaapi.Applications, error)
	GetBySecureVIP(ctx context.Context, svip string) (eurekaapi.Applications, error)
	SetStatus(ctx context.Context, status string) error
	ClearStatusOverride(ctx context.Context, suggestedFallback string) error
	UpdateMetadata(ctx context.Context, kv map[string]string) error

    // Getters
    InstanceID() string
}

func (c *Client) InstanceID() string {
	return c.instanceID
}

func NewClient(eurekaServiceURLs []string, appID string, host string, port int) (ClientAPI, error) {
	eurekaAPIClient, err := eurekaapi.NewEurekaAPIClient(eurekaServiceURLs...)
	if err != nil {
		return nil, err
	}

	return &Client{
		appID:      appID,
		host:       host,
		port:       port,
		instanceID: fmt.Sprintf("%s:%s:%d", host, appID, port),

		eurekaAPIClient: eurekaAPIClient,
	}, nil
}

func (c *Client) WrapTransport(wrap func(http.RoundTripper) http.RoundTripper) {
    if wrap == nil {
        return
    }
    c.eurekaAPIClient.WrapTransport(wrap)
}

type Instance struct {
	ID string
}

func (c *Client) RegisterInstance(ctx context.Context, ip net.IP, ttl uint, useSSL bool) (*Instance, error) {
	dataCenterInfo := &eurekaapi.DataCenter{
		Name: eurekaapi.DefaultDataCenter,
	}
	leaseInfo := &eurekaapi.LeaseInfo{
		EvictionDurationInSecs: ttl,
	}
	instance := &eurekaapi.Instance{
		HostName:         c.host,
		InstanceID:       c.instanceID,
		App:              c.appID,
		IPAddr:           ip.To4().String(),
		Status:           eurekaapi.UP,
		DataCenterInfo:   *dataCenterInfo,
		LeaseInfo:        leaseInfo,
		SecureVipAddress: c.appID,
		VipAddress:       c.appID,
		SecurePort: &eurekaapi.Port{
			Value:   c.port,
			Enabled: useSSL,
		},
		Port: &eurekaapi.Port{
			Value:   c.port,
			Enabled: !useSSL,
		},
	}

	err := c.eurekaAPIClient.RegisterInstance(ctx, c.appID, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to register instance: %w", err)
	}
	return &Instance{
		ID: c.instanceID,
	}, nil
}

func (c *Client) Heartbeat(ctx context.Context) error {
	exists, err := c.eurekaAPIClient.Heartbeat(ctx, c.appID, c.instanceID)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	if !exists {
		return fmt.Errorf("instance %s does not exist", c.instanceID)
	}
	return nil
}

func (c *Client) GetAllApplications(ctx context.Context) (eurekaapi.Applications, error) {
	applications, err := c.eurekaAPIClient.GetAllApplications(ctx)
	if err != nil {
		return eurekaapi.Applications{}, fmt.Errorf("failed to get all applications: %w", err)
	}
	return applications, nil
}

func (c *Client) UnregisterInstance(ctx context.Context) error {
	err := c.eurekaAPIClient.UnregisterInstance(ctx, c.appID, c.instanceID)
	if err != nil {
		return fmt.Errorf("failed to unregister instance: %w", err)
	}
	return nil
}

func (c *Client) GetApplication(ctx context.Context) (eurekaapi.Application, error) {
	application, err := c.eurekaAPIClient.GetApplication(ctx, c.appID)
	if err != nil {
		return eurekaapi.Application{}, fmt.Errorf("failed to get application %s: %w", c.appID, err)
	}
	return application, nil
}

func (c *Client) GetInstance(ctx context.Context) (eurekaapi.Instance, error) {
	instance, err := c.eurekaAPIClient.GetInstance(ctx, c.appID, c.instanceID)
	if err != nil {
		return eurekaapi.Instance{}, fmt.Errorf("failed to get instance %s of application %s: %w", c.instanceID, c.appID, err)
	}
	return instance, nil
}

func (c *Client) GetByVIP(ctx context.Context, vip string) (eurekaapi.Applications, error) {
	applications, err := c.eurekaAPIClient.GetByVIP(ctx, vip)
	if err != nil {
		return eurekaapi.Applications{}, fmt.Errorf("failed to get applications by VIP %s: %w", vip, err)
	}
	return applications, nil
}

func (c *Client) GetBySecureVIP(ctx context.Context, svip string) (eurekaapi.Applications, error) {
	applications, err := c.eurekaAPIClient.GetBySecureVIP(ctx, svip)
	if err != nil {
		return eurekaapi.Applications{}, fmt.Errorf("failed to get applications by secure VIP %s: %w", svip, err)
	}
	return applications, nil
}

func (c *Client) SetStatus(ctx context.Context, status string) error {
	err := c.eurekaAPIClient.SetStatus(ctx, c.appID, c.instanceID, status)
	if err != nil {
		return fmt.Errorf("failed to set status %s for instance %s: %w", status, c.instanceID, err)
	}
	return nil
}

func (c *Client) ClearStatusOverride(ctx context.Context, suggestedFallback string) error {
	err := c.eurekaAPIClient.ClearStatusOverride(ctx, c.appID, c.instanceID, suggestedFallback)
	if err != nil {
		return fmt.Errorf("failed to clear status override for instance %s: %w", c.instanceID, err)
	}
	return nil
}

func (c *Client) UpdateMetadata(ctx context.Context, kv map[string]string) error {
	err := c.eurekaAPIClient.UpdateMetadata(ctx, c.appID, c.instanceID, kv)
	if err != nil {
		return fmt.Errorf("failed to update metadata for instance %s: %w", c.instanceID, err)
	}
	return nil
}
