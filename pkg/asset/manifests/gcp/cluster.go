package gcp

import (
	"fmt"
	"net"
	"os"

	"github.com/apparentlymart/go-cidr/cidr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	capg "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/manifests/capiutils"
)

const (
	// autoCreateSubnets is not used as the subnet information is provided through the install config.
	autoCreateSubnets = false

	defaultSubnetCreationPurpose = "PRIVATE_RFC_1918"
)

var (
	subnetName = func(infraID, machineType string) string { return fmt.Sprintf("%s-%s-subnet", infraID, machineType) }
)

// GenerateClusterAssets generates the manifests for the cluster-api.
func GenerateClusterAssets(installConfig *installconfig.InstallConfig, clusterID *installconfig.ClusterID) (*capiutils.GenerateClusterAssetsOutput, error) {
	manifests := []*asset.RuntimeFile{}

	networkName := installConfig.Config.GCP.Network
	if installConfig.Config.GCP.NetworkProjectID != "" {
		// Subnets should exist in the network with name NetworkProjectID when provided.
		networkName = installConfig.Config.GCP.NetworkProjectID
	}
	if networkName == "" {
		networkName = fmt.Sprintf("%s-network", clusterID.InfraID)
	}

	enableFlowLogs := os.Getenv("GCP_ENABLE_FLOW_LOGS") == "true"

	masterSubnet := installConfig.Config.GCP.ControlPlaneSubnet
	masterSubnetDescription := "Control Plane Subnet used by the installer"
	if masterSubnet == "" {
		masterSubnet = subnetName(clusterID.InfraID, "master")
		masterSubnetDescription = "Control Plane Subnet owned by the installer"
	}
	master := capg.SubnetSpec{
		Name:           masterSubnet,
		CidrBlock:      "",
		Description:    ptr.To(masterSubnetDescription),
		Region:         installConfig.Config.GCP.Region,
		EnableFlowLogs: ptr.To(enableFlowLogs),
		Purpose:        ptr.To(defaultSubnetCreationPurpose),
	}

	workerSubnet := installConfig.Config.GCP.ComputeSubnet
	workerSubnetDescription := "Compute Subnet used by the installer"
	if workerSubnet == "" {
		workerSubnet = subnetName(clusterID.InfraID, "worker")
		workerSubnetDescription = "Compute Subnet owned by the installer"
	}
	worker := capg.SubnetSpec{
		Name:           workerSubnet,
		CidrBlock:      "",
		Description:    ptr.To(workerSubnetDescription),
		Region:         installConfig.Config.GCP.Region,
		EnableFlowLogs: ptr.To(enableFlowLogs),
		Purpose:        ptr.To(defaultSubnetCreationPurpose),
	}

	// Add the CIDR information.
	machineV4CIDRs := []string{}
	for _, network := range installConfig.Config.Networking.MachineNetwork {
		if network.CIDR.IPNet.IP.To4() != nil {
			machineV4CIDRs = append(machineV4CIDRs, network.CIDR.IPNet.String())
		}
	}

	if len(machineV4CIDRs) == 0 {
		return nil, fmt.Errorf("failed to parse machine CIDRs")
	}

	_, ipv4Net, err := net.ParseCIDR(machineV4CIDRs[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine network CIDR: %w", err)
	}

	if installConfig.Config.GCP.ControlPlaneSubnet == "" {
		masterCIDR, err := cidr.Subnet(ipv4Net, 1, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to create the master subnet %w", err)
		}
		// The subnet must be created, provide the CIDR.
		master.CidrBlock = masterCIDR.String()
	}

	if installConfig.Config.GCP.ComputeSubnet == "" {
		workerCIDR, err := cidr.Subnet(ipv4Net, 1, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to create the worker subnet %w", err)
		}
		// The subnet must be created, provide the CIDR.
		worker.CidrBlock = workerCIDR.String()
	}

	subnets := []capg.SubnetSpec{master, worker}

	labels := make(map[string]string, len(installConfig.Config.GCP.UserLabels)+1)
	// add OCP default label.
	labels[fmt.Sprintf("kubernetes-io-cluster-%s", clusterID.InfraID)] = "owned"
	for _, label := range installConfig.Config.GCP.UserLabels {
		labels[label.Key] = label.Value
	}

	gcpCluster := &capg.GCPCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterID.InfraID,
			Namespace: capiutils.Namespace,
		},
		Spec: capg.GCPClusterSpec{
			Project: installConfig.Config.GCP.ProjectID,
			Region:  installConfig.Config.GCP.Region,
			Network: capg.NetworkSpec{
				Name:                  ptr.To(networkName),
				AutoCreateSubnetworks: ptr.To(autoCreateSubnets),
				Subnets:               subnets,
			},
			AdditionalLabels: labels,
		},
	}

	manifests = append(manifests, &asset.RuntimeFile{
		Object: gcpCluster,
		File:   asset.File{Filename: "02_gcp-cluster.yaml"},
	})

	return &capiutils.GenerateClusterAssetsOutput{
		Manifests: manifests,
		InfrastructureRef: &corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
			Kind:       "GCPCluster",
			Name:       gcpCluster.Name,
			Namespace:  gcpCluster.Namespace,
		},
	}, nil
}
