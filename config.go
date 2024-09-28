package main

import "github.com/ServiceWeaver/weaver"

type executorConfig struct {
	weaver.AutoMarshal
	Email     string            `toml:"email"`
	ApiKey    string            `toml:"api_key"`
	DNSUpdate []DNSUpdateOption `toml:"dns_update"`
	Pages     PagesOperations   `toml:"pages"`
}

type PagesOperations struct {
	weaver.AutoMarshal
	Cleanup  PageCleanupOption  `toml:"cleanup"`
	BuildEnv PageBuildEnvOption `toml:"build_env"`
}

type PageCleanupOption struct {
	weaver.AutoMarshal
	Enabled        bool `toml:"enabled"`
	OnlyKeepLatest bool `json:"only_keep_latest"`
}

type PageBuildEnvOption struct {
	weaver.AutoMarshal
	Enabled bool `toml:"enabled"`
	//key:value variable_name:release_version
	GithubRelease map[string]string `toml:"github_release"`
}

type DNSUpdateOption struct {
	weaver.AutoMarshal
	ZoneName       string   `toml:"zone_name"`
	DnsRecordNames []string `toml:"dns_record_names"`
}
