package gcp

import (
	"fmt"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	capg "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/manifests/capiutils"
	"github.com/openshift/installer/pkg/types/gcp"
)

// GenerateClusterAssets generates the manifests for the cluster-api.
func GenerateClusterAssets(installConfig *installconfig.InstallConfig, clusterID *installconfig.ClusterID) (*capiutils.GenerateClusterAssetsOutput, error) {
	manifests := []*asset.RuntimeFile{}

	const (
		description = "Created By OpenShift Installer"
	)

	networkName := fmt.Sprintf("%s-network", clusterID.InfraID)
	if installConfig.Config.GCP.Network != "" {
		networkName = installConfig.Config.GCP.Network
	}

	masterSubnet := gcp.DefaultSubnetName(clusterID.InfraID, "master")
	if installConfig.Config.GCP.ControlPlaneSubnet != "" {
		masterSubnet = installConfig.Config.GCP.ControlPlaneSubnet
	}

	master := capg.SubnetSpec{
		Name:        masterSubnet,
		CidrBlock:   "",
		Description: ptr.To(description),
		Region:      installConfig.Config.GCP.Region,
	}

	workerSubnet := gcp.DefaultSubnetName(clusterID.InfraID, "worker")
	if installConfig.Config.GCP.ComputeSubnet != "" {
		workerSubnet = installConfig.Config.GCP.ComputeSubnet
	}

	worker := capg.SubnetSpec{
		Name:        workerSubnet,
		CidrBlock:   "",
		Description: ptr.To(description),
		Region:      installConfig.Config.GCP.Region,
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
		master.CidrBlock = masterCIDR.String()
	}

	if installConfig.Config.GCP.ComputeSubnet == "" {
		workerCIDR, err := cidr.Subnet(ipv4Net, 1, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to create the worker subnet %w", err)
		}
		worker.CidrBlock = workerCIDR.String()
	}

	subnets := []capg.SubnetSpec{master, worker}

	labels := map[string]string{}
	labels[fmt.Sprintf("kubernetes-io-cluster-%s", clusterID.InfraID)] = "owned"
	labels[fmt.Sprintf("capg-cluster-%s", clusterID.InfraID)] = "owned"
	for _, label := range installConfig.Config.GCP.UserLabels {
		labels[label.Key] = label.Value
	}

	// Find the availability zones and use this information to configure the GCP Cluster Failure Domains.
	zones := sets.New[string]()
	if installConfig.Config.Platform.GCP != nil {
		if installConfig.Config.Platform.GCP.DefaultMachinePlatform != nil {
			for _, zone := range installConfig.Config.Platform.GCP.DefaultMachinePlatform.Zones {
				zones.Insert(zone)
			}
		}
	}
	if len(zones) == 0 {
		// Find the possible zones from the Control Plane
		if installConfig.Config.ControlPlane.Platform.GCP != nil {
			for _, zone := range installConfig.Config.ControlPlane.Platform.GCP.Zones {
				zones.Insert(zone)
			}
		}
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
				// TODO: Need a network project for installs where the network resources will exist in another
				// project such as shared vpc installs
				Name:    ptr.To(networkName),
				Subnets: subnets,
			},
			AdditionalLabels: labels,
			FailureDomains:   zones.UnsortedList(),
		},
	}

	manifests = append(manifests, &asset.RuntimeFile{
		Object: gcpCluster,
		File:   asset.File{Filename: "02_gcp-cluster.yaml"},
	})

	return &capiutils.GenerateClusterAssetsOutput{
		Manifests: manifests,
		InfrastructureRef: &corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "GCPCluster",
			Name:       gcpCluster.Name,
			Namespace:  gcpCluster.Namespace,
		},
	}, nil
}
