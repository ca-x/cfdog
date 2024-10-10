package main

import (
	"cfdog/internal/pkg/cfclient"
	"cfdog/internal/pkg/ghrelase"
	"context"
	"github.com/RussellLuo/timingwheel"
	"github.com/ServiceWeaver/weaver"
	"github.com/cloudflare/cloudflare-go"
	"github.com/sourcegraph/conc/iter"
	"time"
)

var _ CloudflareUpdateExecutor = (*cloudflareUpdateExecute)(nil)

type CloudflareUpdateExecutor interface {
	// Execute update dns record
	Execute(ctx context.Context) error
}

type cloudflareUpdateExecute struct {
	weaver.Implements[CloudflareUpdateExecutor]
	weaver.WithConfig[executorConfig]
	cf *cfclient.SimpleClient
}

type EveryScheduler struct {
	Interval time.Duration
}

func (s *EveryScheduler) Next(prev time.Time) time.Time {
	return prev.Add(s.Interval)
}

func (c *cloudflareUpdateExecute) Execute(ctx context.Context) error {
	config := c.Config()
	go c.doJobs(ctx, config)
	return nil
}

func (c *cloudflareUpdateExecute) doJobs(ctx context.Context, config *executorConfig) {
	tw := timingwheel.NewTimingWheel(time.Millisecond, 20)
	tw.Start()
	defer tw.Stop()
	notifyJob := make(chan struct{})
	t := tw.ScheduleFunc(&EveryScheduler{time.Second * time.Duration(config.JobsIntervalSeconds)}, func() {
		notifyJob <- struct{}{}
	})

	for {
		select {
		case <-notifyJob:
			for _, updateOpt := range config.DNSUpdate {
				zoneId, err := c.cf.GetZoneId(updateOpt.ZoneName)
				if err != nil {
					continue
				}
				// update dns
				for _, name := range updateOpt.DnsRecordNames {
					records, _, err := c.cf.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneId), cloudflare.ListDNSRecordsParams{Name: name})
					if err != nil {
						continue
					}
					go c.cf.UpdateDnsRecord(ctx, zoneId, records)
				}

			}
			// cleanup pages deployments
			// update pages env variable
			go c.handlePages(ctx, config.Pages)
		case <-ctx.Done():
			t.Stop()
		}
	}
}

func (c *cloudflareUpdateExecute) handlePages(ctx context.Context, pages PagesOperations) error {
	logger := c.Logger(ctx)
	accountId, err := c.cf.GetCloudflareAccountId(ctx)
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
				logger.Info("start to update project build env", "project name", project.Name, "project id", project.ID, "release version", releaseVersion)
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
		if pages.Cleanup.Enabled {
			latestId := project.LatestDeployment.ID
			logger.Info("get latest deploy info", "latest id", latestId)
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
			logger.Info("start to delete deployments", "count", len(deployments))
			iter.ForEach(deployments, func(deployment *cloudflare.PagesProjectDeployment) {
				if deployment.ID == latestId {
					logger.Info("skip latest deployment", "deployment id", latestId)
					return
				}
				logger.Info("try to delete  deployment", "id", deployment.ID)
				if err := c.cf.DeletePagesDeployment(
					context.Background(),
					cloudflare.AccountIdentifier(accountId),
					cloudflare.DeletePagesDeploymentParams{
						DeploymentID: deployment.ID,
						ProjectName:  project.Name,
						Force:        true,
					},
				); err != nil {
					logger.Error("delete deployment failed", err)
					return
				}
			})
		}
	})
	return nil
}

func (c *cloudflareUpdateExecute) Init(ctx context.Context) error {
	config := c.Config()
	cf, err := cfclient.NewSimpleClient(config.ApiKey, config.Email, c.Logger(ctx))
	if err != nil {
		return err
	}
	c.cf = cf
	return nil
}
