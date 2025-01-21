package aws

import (
	"context"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	awstypes "github.com/openshift/installer/pkg/types/aws"
)

// GetConfig returns the default config AWS configuration.
func GetConfig() (awsv2.Config, error) { return config.LoadDefaultConfig(context.TODO()) }

// ConfigWithRegion returns the LoadOption with the region set.
func ConfigWithRegion(region string) config.LoadOptionsFunc {
	return config.WithRegion(region)
}

// ConfigWithEndpoint returns the LoadOption with the service endpoint set for the specified service.
func ConfigWithEndpoint(service string, endpoints []awstypes.ServiceEndpoint) config.LoadOptionsFunc {
	for _, endpoint := range endpoints {
		if endpoint.Name == service {
			return config.WithEndpointResolverWithOptions(
				awsv2.EndpointResolverWithOptionsFunc(func(service, region string, opts ...interface{}) (awsv2.Endpoint, error) { //nolint: staticcheck
					return awsv2.Endpoint{URL: endpoint.URL, SigningRegion: region}, nil //nolint: staticcheck
				}))
		}
	}
	return nil
}

// ConfigWithDualStackEndpoint returns the LoadOption with the dual stack endpoint state set.
func ConfigWithDualStackEndpoint(state awsv2.DualStackEndpointState) config.LoadOptionsFunc {
	return config.WithUseDualStackEndpoint(state)
}

// GetConfigWithOptionsFromPlatform returns the AWS Config with options set from the AWS Platform.
func GetConfigWithOptionsFromPlatform(platform awstypes.Platform) (awsv2.Config, error) {
	return GetConfigWithOptions(platform.Region, "ec2", platform.ServiceEndpoints)
}

// GetConfigWithOptions returns the AWS Config with options set by the user information.
func GetConfigWithOptions(region, service string, endpoints []awstypes.ServiceEndpoint) (awsv2.Config, error) {
	loadOptions := []func(*config.LoadOptions) error{
		ConfigWithRegion(region),
	}

	cfgOption := ConfigWithEndpoint(service, endpoints)
	if cfgOption != nil {
		loadOptions = append(loadOptions, cfgOption)
	}

	return config.LoadDefaultConfig(context.TODO(), loadOptions...)
}
