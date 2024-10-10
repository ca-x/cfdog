package cfclient

import (
	"cfdog/internal/pkg/myip"
	"context"
	"errors"
	"github.com/cloudflare/cloudflare-go"
	"github.com/sourcegraph/conc"
	"log/slog"
)

type SimpleClient struct {
	cf     *cloudflare.API
	logger *slog.Logger
}

func NewSimpleClient(apikey string, email string, logger *slog.Logger) (*SimpleClient, error) {
	options := []cloudflare.Option{
		cloudflare.UsingRetryPolicy(5, 1, 1),
	}
	cf, err := cloudflare.New(apikey, email, options...)
	if err != nil {
		return nil, err
	}
	return &SimpleClient{
		cf:     cf,
		logger: logger,
	}, nil
}

func (s *SimpleClient) GetZoneId(zoneName string) (string, error) {
	return s.cf.ZoneIDByName(zoneName)
}

func (s *SimpleClient) GetCloudflareAccountId(ctx context.Context) (string, error) {
	accounts, info, err := s.cf.Accounts(ctx, cloudflare.AccountsListParams{})
	if err != nil {
		return "", err
	}
	if info.Count > 0 {
		return accounts[0].ID, nil
	}
	return "", errors.New("no account found")
}

func (s *SimpleClient) UpdateDnsRecord(ctx context.Context, zoneId string, recs []cloudflare.DNSRecord) {
	var (
		ipv4 = ""
		ipv6 = ""
		err  error
	)
	logger := s.logger
	wg := &conc.WaitGroup{}
	wg.Go(func() {
		ipv4, err = myip.GetIPv4Address()
		if err != nil {
			logger.Error("fetch ip v4 address failed", err)
		}

	})
	wg.Go(func() {
		ipv6, err = myip.GetIPv6Address()
		if err != nil {
			logger.Error("fetch ip v6 address failed", err)
		}
	})
	wg.Wait()
	logger.Info("fetch ips success!")
	for _, r := range recs {
		ip := ""
		switch r.Type {
		case "A":
			ip = ipv4
		case "AAAA":
			ip = ipv6
		}
		if ip == "" {
			continue
		}
		_, updateDnsErr := s.cf.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneId), cloudflare.UpdateDNSRecordParams{
			ID:      r.ID,
			Type:    r.Type,
			Content: ip,
			TTL:     1,
			Proxied: r.Proxied,
		})
		if updateDnsErr != nil {
			continue
		} else {
			logger.Info("update  dns record success", "name", r.Name, "type", r.Type)
		}

	}
}

func (s *SimpleClient) ListDNSRecords(ctx context.Context, identifier *cloudflare.ResourceContainer, params cloudflare.ListDNSRecordsParams) ([]cloudflare.DNSRecord, *cloudflare.ResultInfo, error) {
	return s.cf.ListDNSRecords(ctx, identifier, params)
}

func (s *SimpleClient) ListPagesProjects(background context.Context, identifier *cloudflare.ResourceContainer, opt cloudflare.ListPagesProjectsParams) ([]cloudflare.PagesProject, cloudflare.ResultInfo, error) {
	return s.cf.ListPagesProjects(background, identifier, opt)
}

func (s *SimpleClient) UpdatePagesProject(background context.Context, identifier *cloudflare.ResourceContainer, params cloudflare.UpdatePagesProjectParams) (cloudflare.PagesProject, error) {
	return s.cf.UpdatePagesProject(background, identifier, params)
}

func (s *SimpleClient) ListPagesDeployments(background context.Context, identifier *cloudflare.ResourceContainer, params cloudflare.ListPagesDeploymentsParams) ([]cloudflare.PagesProjectDeployment, *cloudflare.ResultInfo, error) {
	return s.cf.ListPagesDeployments(background, identifier, params)
}

func (s *SimpleClient) UpdateDNSRecord(ctx context.Context, identifier *cloudflare.ResourceContainer, params cloudflare.UpdateDNSRecordParams) (cloudflare.DNSRecord, interface{}) {
	return s.cf.UpdateDNSRecord(ctx, identifier, params)
}

func (s *SimpleClient) DeletePagesDeployment(background context.Context, identifier *cloudflare.ResourceContainer, params cloudflare.DeletePagesDeploymentParams) error {
	return s.cf.DeletePagesDeployment(background, identifier, params)
}
