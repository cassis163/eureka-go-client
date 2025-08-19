package pkg

import (
	"context"
	"fmt"
	"net"

	eurekaapi "github.com/cassis163/eureka-go-client/client/internal/eureka-api"
)

type Client struct {
	eurekaAPIClient eurekaapi.EurekaAPI
	AppID           string
	Host            string
	Port            int
}

type ClientAPI interface {
	RegisterInstance(ctx context.Context, ip net.IP, ttl uint, useSSL bool) (*Instance, error)
	Heartbeat(ctx context.Context, instanceID string) error
	GetAllApplications(ctx context.Context) (eurekaapi.Applications, error)
	UnregisterInstance(ctx context.Context, instanceID string) error
	GetApplication(ctx context.Context, appID string) (eurekaapi.Application, error)
	GetInstance(ctx context.Context, appID, instanceID string) (eurekaapi.Instance, error)
	GetByVIP(ctx context.Context, vip string) (eurekaapi.Applications, error)
	GetBySecureVIP(ctx context.Context, svip string) (eurekaapi.Applications, error)
	SetStatus(ctx context.Context, instanceID, status string) error
	ClearStatusOverride(ctx context.Context, instanceID string, suggestedFallback string) error
	UpdateMetadata(ctx context.Context, instanceID string, kv map[string]string) error
}

func NewClient(eurekaServiceURLs []string, appID string, host string, port int) (ClientAPI, error) {
	eurekaAPIClient, err := eurekaapi.NewEurekaAPIClient(eurekaServiceURLs...)
	if err != nil {
		return nil, err
	}

	return &Client{
		eurekaAPIClient: eurekaAPIClient,
		AppID:           appID,
		Host:            host,
		Port:            port,
	}, nil
}

type Instance struct {
	ID string
}

func (c *Client) RegisterInstance(ctx context.Context, ip net.IP, ttl uint, useSSL bool) (*Instance, error) {
	instanceID := fmt.Sprintf("%s:%s:%d", c.Host, c.AppID, c.Port)
	dataCenterInfo := &eurekaapi.DataCenter{
		Name: eurekaapi.DefaultDataCenter,
	}
	leaseInfo := &eurekaapi.LeaseInfo{
		EvictionDurationInSecs: ttl,
	}
	instance := &eurekaapi.Instance{
		HostName:         c.Host,
		InstanceID:       instanceID,
		App:              c.AppID,
		IPAddr:           ip.To4().String(),
		Status:           eurekaapi.UP,
		DataCenterInfo:   *dataCenterInfo,
		LeaseInfo:        leaseInfo,
		SecureVipAddress: c.AppID,
		VipAddress:       c.AppID,
		SecurePort: &eurekaapi.Port{
			Value:   c.Port,
			Enabled: useSSL,
		},
		Port: &eurekaapi.Port{
			Value:   c.Port,
			Enabled: !useSSL,
		},
	}

	err := c.eurekaAPIClient.RegisterInstance(ctx, c.AppID, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to register instance: %w", err)
	}
	return &Instance{
		ID: instanceID,
	}, nil
}

func (c *Client) Heartbeat(ctx context.Context, instanceID string) error {
	exists, err := c.eurekaAPIClient.Heartbeat(ctx, c.AppID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	if !exists {
		return fmt.Errorf("instance %s does not exist", instanceID)
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

func (c *Client) UnregisterInstance(ctx context.Context, instanceID string) error {
	err := c.eurekaAPIClient.UnregisterInstance(ctx, c.AppID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to unregister instance: %w", err)
	}
	return nil
}

func (c *Client) GetApplication(ctx context.Context, appID string) (eurekaapi.Application, error) {
	application, err := c.eurekaAPIClient.GetApplication(ctx, appID)
	if err != nil {
		return eurekaapi.Application{}, fmt.Errorf("failed to get application %s: %w", appID, err)
	}
	return application, nil
}

func (c *Client) GetInstance(ctx context.Context, appID, instanceID string) (eurekaapi.Instance, error) {
	instance, err := c.eurekaAPIClient.GetInstance(ctx, appID, instanceID)
	if err != nil {
		return eurekaapi.Instance{}, fmt.Errorf("failed to get instance %s of application %s: %w", instanceID, appID, err)
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

func (c *Client) SetStatus(ctx context.Context, instanceID, status string) error {
	err := c.eurekaAPIClient.SetStatus(ctx, c.AppID, instanceID, status)
	if err != nil {
		return fmt.Errorf("failed to set status %s for instance %s: %w", status, instanceID, err)
	}
	return nil
}

func (c *Client) ClearStatusOverride(ctx context.Context, instanceID string, suggestedFallback string) error {
	err := c.eurekaAPIClient.ClearStatusOverride(ctx, c.AppID, instanceID, suggestedFallback)
	if err != nil {
		return fmt.Errorf("failed to clear status override for instance %s: %w", instanceID, err)
	}
	return nil
}

func (c *Client) UpdateMetadata(ctx context.Context, instanceID string, kv map[string]string) error {
	err := c.eurekaAPIClient.UpdateMetadata(ctx, c.AppID, instanceID, kv)
	if err != nil {
		return fmt.Errorf("failed to update metadata for instance %s: %w", instanceID, err)
	}
	return nil
}
