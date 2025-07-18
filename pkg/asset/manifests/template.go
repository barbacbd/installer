package manifests

import (
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/baremetal"
)

// AwsCredsSecretData holds encoded credentials and is used to generate cloud-creds secret
type AwsCredsSecretData struct {
	Base64encodeAccessKeyID     string
	Base64encodeSecretAccessKey string
}

// AzureCredsSecretData holds encoded credentials and is used to generate cloud-creds secret
type AzureCredsSecretData struct {
	Base64encodeSubscriptionID string
	Base64encodeClientID       string
	Base64encodeClientSecret   string
	Base64encodeTenantID       string
	Base64encodeResourcePrefix string
	Base64encodeResourceGroup  string
	Base64encodeRegion         string
}

// GCPCredsSecretData holds encoded credentials and is used to generate cloud-creds secret
type GCPCredsSecretData struct {
	Base64encodeServiceAccount string
}

// IBMCloudCredsSecretData holds encoded credentials and is used to generate cloud-creds secret
type IBMCloudCredsSecretData struct {
	Base64encodeAPIKey string
}

// OpenStackCredsSecretData holds encoded credentials and is used to generate cloud-creds secret
type OpenStackCredsSecretData struct {
	Base64encodeCloudsYAML string
	Base64encodeCloudsConf string
	Base64encodeCACert     string
}

// VSphereCredsSecretData holds encoded credentials and is used to generated cloud-creds secret
type VSphereCredsSecretData struct {
	VCenter              string
	Base64encodeUsername string
	Base64encodePassword string
}

// OvirtCredsSecretData holds encoded credentials and is used to generated cloud-creds secret
type OvirtCredsSecretData struct {
	Base64encodeURL      string
	Base64encodeUsername string
	Base64encodePassword string
	Base64encodeInsecure string
	Base64encodeCABundle string
}

type cloudCredsSecretData struct {
	AWS       *AwsCredsSecretData
	Azure     *AzureCredsSecretData
	GCP       *GCPCredsSecretData
	IBMCloud  *IBMCloudCredsSecretData
	OpenStack *OpenStackCredsSecretData
	VSphere   *[]*VSphereCredsSecretData
	Ovirt     *OvirtCredsSecretData
}

type bootkubeTemplateData struct {
	CVOCapabilities            *types.Capabilities
	CVOClusterID               string
	EtcdCaBundle               string
	EtcdMetricCaCert           string
	EtcdMetricSignerCert       string
	EtcdMetricSignerClientCert string
	EtcdMetricSignerClientKey  string
	EtcdMetricSignerKey        string
	EtcdSignerCert             string
	EtcdSignerClientCert       string
	EtcdSignerClientKey        string
	EtcdSignerKey              string
	IsFCOS                     bool
	IsSCOS                     bool
	IsOKD                      bool
	McsTLSCert                 string
	McsTLSKey                  string
	McsHostName                string
	McsTLSCertNotAfter         string
	McsTLSCertNotBefore        string
	PullSecretBase64           string
	RootCaCert                 string
	RootCACertBase64           string
	RootCASignerKeyBase64      string
	RootCAIssuerName           string
	RootCANotAfter             string
	RootCANotBefore            string
	WorkerIgnConfig            string
}

type baremetalTemplateData struct {
	Baremetal                 *baremetal.Platform
	ProvisioningOSDownloadURL string
}

type openshiftTemplateData struct {
	CloudCreds                   cloudCredsSecretData
	Base64EncodedKubeadminPwHash string
}

type mcoTemplateData struct {
	DisableMachinesetBootMgmt bool
}
