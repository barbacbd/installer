package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	awss "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/openshift/installer/pkg/types"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"strings"
	"time"
)

//go:generate mockgen -source=./route53.go -destination=mock/awsroute53_generated.go -package=mock

// API represents the calls made to the API.
type API interface {
	GetHostedZone(ctx context.Context, hostedZone string) (*route53.GetHostedZoneOutput, error)
	ValidateZoneRecords(ctx context.Context, zone *route53.HostedZone, zoneName string, zonePath *field.Path, ic *types.InstallConfig) field.ErrorList
	GetBaseDomain(ctx context.Context, baseDomainName string) (*route53.HostedZone, error)
}

// Client makes calls to the AWS Route53 API.
type Client struct {
	ssn *awss.Session
}

// NewClient initializes a client with a session.
func NewClient(ssn *awss.Session) *Client {
	client := &Client{
		ssn: ssn,
	}
	return client
}

// GetHostedZone attempts to get the hosted zone from the AWS Route53 instance
func (c *Client) GetHostedZone(ctx context.Context, hostedZone string) (*route53.GetHostedZoneOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	// build a new Route53 instance from the same session that made it here
	r53 := route53.New(c.ssn)

	// validate that the hosted zone exists
	hostedZoneOutput, err := r53.GetHostedZone(&route53.GetHostedZoneInput{Id: aws.String(hostedZone)})
	if err != nil {
		return nil, err
	}
	return hostedZoneOutput, nil
}

// ValidateZoneRecords Attempts to validate each of the candidate HostedZones against the Config
func (c *Client) ValidateZoneRecords(ctx context.Context, zone *route53.HostedZone, zoneName string, zonePath *field.Path, ic *types.InstallConfig) field.ErrorList {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	allErrs := field.ErrorList{}

	problematicRecords, err := c.getSubDomainDNSRecords(zone, ic)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(zonePath,
			errors.Wrapf(err, "could not list record sets for domain %q", zoneName)))
	}

	if len(problematicRecords) > 0 {
		detail := fmt.Sprintf(
			"the zone already has record sets for the domain of the cluster: [%s]",
			strings.Join(problematicRecords, ", "),
		)
		allErrs = append(allErrs, field.Invalid(zonePath, zoneName, detail))
	}

	return allErrs
}

func (c *Client) getSubDomainDNSRecords(hostedZone *route53.HostedZone, ic *types.InstallConfig) ([]string, error) {
	dottedClusterDomain := ic.ClusterDomain() + "."

	// validate that the domain of the hosted zone is the cluster domain or a parent of the cluster domain
	if !isHostedZoneDomainParentOfClusterDomain(hostedZone, dottedClusterDomain) {
		return nil, errors.Errorf("hosted zone domain %q is not a parent of the cluster domain %q", *hostedZone.Name, dottedClusterDomain)
	}

	r53 := route53.New(c.ssn)

	var problematicRecords []string
	// validate that the hosted zone does not already have any record sets for the cluster domain
	if err := r53.ListResourceRecordSetsPages(
		&route53.ListResourceRecordSetsInput{HostedZoneId: hostedZone.Id},
		func(out *route53.ListResourceRecordSetsOutput, lastPage bool) bool {
			for _, recordSet := range out.ResourceRecordSets {
				name := aws.StringValue(recordSet.Name)
				// skip record sets that are not sub-domains of the cluster domain. Such record sets may exist for
				// hosted zones that are used for other clusters or other purposes.
				if !strings.HasSuffix(name, dottedClusterDomain) {
					continue
				}
				// skip record sets that are the cluster domain. Record sets for the cluster domain are fine. If the
				// hosted zone has the name of the cluster domain, then there will be NS and SOA record sets for the
				// cluster domain.
				if len(name) == len(dottedClusterDomain) {
					continue
				}
				problematicRecords = append(problematicRecords, fmt.Sprintf("%s (%s)", name, aws.StringValue(recordSet.Type)))
			}
			return !lastPage
		},
	); err != nil {
		return nil, err
	}

	return problematicRecords, nil
}

func isHostedZoneDomainParentOfClusterDomain(hostedZone *route53.HostedZone, dottedClusterDomain string) bool {
	if *hostedZone.Name == dottedClusterDomain {
		return true
	}
	return strings.HasSuffix(dottedClusterDomain, "."+*hostedZone.Name)
}

// GetBaseDomain Gets the Domain Zone with the matching domain name from the session
func (c *Client) GetBaseDomain(ctx context.Context, baseDomainName string) (*route53.HostedZone, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	baseDomainZone, err := GetPublicZone(c.ssn, baseDomainName)
	if err != nil {
		return nil, err
	}
	return baseDomainZone, nil
}
