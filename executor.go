package main

import (
	"cfdog/internal/pkg/ghrelase"
	"cfdog/internal/pkg/myip"
	"context"
	"errors"
	"github.com/ServiceWeaver/weaver"
	"github.com/cloudflare/cloudflare-go"
	"github.com/sourcegraph/conc"
	"github.com/sourcegraph/conc/iter"
	"log"
)

var _ CloudflareUpdateExecutor = (*cloudflareUpdateExecute)(nil)

type CloudflareUpdateExecutor interface {
	// Execute update dns record
	Execute(ctx context.Context) error
}

type cloudflareUpdateExecute struct {
	weaver.Implements[CloudflareUpdateExecutor]
	weaver.WithConfig[executorConfig]
	cf *cloudflare.API
}

func (c *cloudflareUpdateExecute) Execute(ctx context.Context) error {
	config := c.Config()
	for _, updateOpt := range config.DNSUpdate {
		zoneId, err := c.getZoneId(updateOpt.ZoneName)
		if err != nil {
			continue
		}
		// update dns
		for _, name := range updateOpt.DnsRecordNames {
			records, _, err := c.cf.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneId), cloudflare.ListDNSRecordsParams{Name: name})
			if err != nil {
				continue
			}
			go c.handleDnsRecord(context.Background(), zoneId, records)
		}

	}
	// cleanup pages deployments
	// update pages env variable
	go c.handlePages(config.Pages)
	return nil
}

func (c *cloudflareUpdateExecute) getZoneId(zoneName string) (string, error) {
	return c.cf.ZoneIDByName(zoneName)
}

func (c *cloudflareUpdateExecute) getCloudflareAccountId(ctx context.Context) (string, error) {
	accounts, info, err := c.cf.Accounts(ctx, cloudflare.AccountsListParams{})
	if err != nil {
		return "", err
	}
	if info.Count > 0 {
		return accounts[0].ID, nil
	}
	return "", errors.New("no account found")
}

func (c *cloudflareUpdateExecute) handleDnsRecord(ctx context.Context, zoneId string, recs []cloudflare.DNSRecord) {

	var (
		ipv4 = ""
		ipv6 = ""
		err  error
	)
	wg := &conc.WaitGroup{}
	wg.Go(func() {
		ipv4, err = myip.GetIPv4Address()
		if err != nil {
			log.Printf("fetch ip v4 address failed:%v \n", err)
		}

	})
	wg.Go(func() {
		ipv6, err = myip.GetIPv6Address()
		if err != nil {
			log.Printf("fetch ip v6 address failed:%v \n", err)
		}
	})

	wg.Wait()

	log.Println("fetch ips success!")
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
		_, updateDnsErr := c.cf.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneId), cloudflare.UpdateDNSRecordParams{
			ID:      r.ID,
			Type:    r.Type,
			Content: ip,
			TTL:     1,
			Proxied: r.Proxied,
		})
		if updateDnsErr != nil {
			continue
		} else {
			log.Printf("update %s dns record ok,type:%s \n", r.Name, r.Type)
		}

	}
}
func (c *cloudflareUpdateExecute) handlePages(pages PagesOperations) error {
	accountId, err := c.getCloudflareAccountId(context.Background())
	if err != nil {
		return err
	}
	pageOpt := cloudflare.PaginationOptions{Page: 1, PerPage: 5}
	listOpt := cloudflare.ListPagesProjectsParams{PaginationOptions: pageOpt}
	projects, _, err := c.cf.ListPagesProjects(context.Background(), cloudflare.UserIdentifier(accountId), listOpt)
	if err != nil {
		return err
	}
	iter.ForEach(projects, func(project *cloudflare.PagesProject) {

		if pages.BuildEnv.Enabled {
			for envVariableName, repo := range pages.BuildEnv.GithubRelease {
				releaseVersion, err := ghrelase.GetLatest(repo)
				if err != nil {
					continue
				}
				if releaseVersion == "" && len(releaseVersion) > 1 {
					continue
				}
				releaseVersion = releaseVersion[1:]

				ProductionEnv := cloudflare.EnvironmentVariableMap{}
				log.Printf("update project %s(id:%s) 's mdbok version to :%s", project.Name, project.ID, releaseVersion)
				ProductionEnv[envVariableName] = &cloudflare.EnvironmentVariable{Value: releaseVersion, Type: cloudflare.PlainText}
				project.DeploymentConfigs.Production.EnvVars = ProductionEnv
				_, err = c.cf.UpdatePagesProject(context.Background(),
					cloudflare.AccountIdentifier(accountId),
					cloudflare.UpdatePagesProjectParams{
						ID:                  project.ID,
						Name:                project.Name,
						SubDomain:           project.SubDomain,
						Domains:             project.Domains,
						Source:              project.Source,
						BuildConfig:         project.BuildConfig,
						DeploymentConfigs:   project.DeploymentConfigs,
						LatestDeployment:    project.LatestDeployment,
						CanonicalDeployment: project.CanonicalDeployment,
						ProductionBranch:    project.ProductionBranch,
					})
				if err != nil {
					continue
				}
			}
		}
		// pages deployment
		if pages.Cleanup.Enabled && pages.Cleanup.OnlyKeepLatest {
			latestId := project.LatestDeployment.ID
			log.Println("latest deploy id:", latestId)
			// remove history deploy
			deployments, _, err := c.cf.ListPagesDeployments(
				context.Background(),
				cloudflare.AccountIdentifier(accountId),
				cloudflare.ListPagesDeploymentsParams{
					ProjectName: project.Name,
					ResultInfo: cloudflare.ResultInfo{
						Page:    1,
						PerPage: 20,
					},
				},
			)
			if err != nil {
				return
			}
			log.Printf("start to delete deploy:%+v\r\n", deployments)
			iter.ForEach(deployments, func(deployment *cloudflare.PagesProjectDeployment) {
				if deployment.ID == latestId {
					log.Println("skip latest deploy id:", latestId)
					return
				}
				log.Println("try to delate  deployment with id:", deployment.ID)
				if err := c.cf.DeletePagesDeployment(
					context.Background(),
					cloudflare.AccountIdentifier(accountId),
					cloudflare.DeletePagesDeploymentParams{
						DeploymentID: deployment.ID,
						ProjectName:  project.Name,
						Force:        true,
					},
				); err != nil {
					log.Println(err)
					return
				}
			})
		}

	})

	return nil
}

func (c *cloudflareUpdateExecute) Init(context.Context) error {
	config := c.Config()
	options := []cloudflare.Option{
		cloudflare.UsingRetryPolicy(5, 1, 1),
	}
	cf, err := cloudflare.New(config.ApiKey, config.Email, options...)
	if err != nil {
		return err
	}
	c.cf = cf
	return nil
}
