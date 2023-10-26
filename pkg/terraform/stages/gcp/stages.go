package gcp

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/openshift/installer/pkg/asset/lbconfig"
	assetstore "github.com/openshift/installer/pkg/asset/store"
	"github.com/openshift/installer/pkg/terraform"
	"github.com/openshift/installer/pkg/terraform/providers"
	"github.com/openshift/installer/pkg/terraform/stages"
	gcptypes "github.com/openshift/installer/pkg/types/gcp"
)

// PlatformStages are the stages to run to provision the infrastructure in GCP.
var PlatformStages = []terraform.Stage{
	stages.NewStage(
		"gcp",
		"cluster",
		[]providers.Provider{providers.Google},
		stages.WithCustomExtractLBConfig(extractGCPLBConfig),
	),
	stages.NewStage(
		"gcp",
		"bootstrap",
		[]providers.Provider{providers.Google, providers.Ignition},
		stages.WithNormalBootstrapDestroy(),
	),
	stages.NewStage(
		"gcp",
		"post-bootstrap",
		[]providers.Provider{providers.Google},
		stages.WithCustomBootstrapDestroy(removeFromLoadBalancers),
	),
}

func removeFromLoadBalancers(s stages.SplitStage, directory string, terraformDir string, varFiles []string) error {
	opts := make([]tfexec.ApplyOption, 0, len(varFiles)+1)
	for _, varFile := range varFiles {
		opts = append(opts, tfexec.VarFile(varFile))
	}
	opts = append(opts, tfexec.Var("gcp_bootstrap_lb=false"))
	return errors.Wrap(
		terraform.Apply(directory, gcptypes.Name, s, terraformDir, opts...),
		"failed disabling bootstrap load balancing",
	)
}

func extractGCPLBConfig(s stages.SplitStage, directory string, terraformDir string, file *asset.File, tfvarsFile *asset.File) (string, error) {
	outputs := map[string]interface{}{}
	err := json.Unmarshal(file.Data, &outputs)
	if err != nil {
		return "", err
	}

	// Extract the Load Balancer ip addresses from the terraform output.
	apiLBIpRaw, ok := outputs["cluster_public_ip"]
	if !ok {
		return "", fmt.Errorf("failed to read External API LB DNS Name from terraform outputs")
	}
	apiIntLBIpRaw, ok := outputs["cluster_ip"]
	if !ok {
		return "", fmt.Errorf("failed to read Internal API LB DNS Name from terraform outputs")
	}

	// Create the load balancer configuration file, and add the parsed terraform variables
	// from above.
	lbConfigContents, err := lbconfig.CreateLBConfigMap("openshift-lbConfigForDNS", apiIntLBIpRaw.(string), apiLBIpRaw.(string), s.Platform())
	if err != nil {
		return "", fmt.Errorf("failed to create load balancer config contents: %w", err)
	}
	pth := fmt.Sprintf("%s/%s", directory, lbconfig.ConfigName)
	if err := os.WriteFile(pth, []byte(lbConfigContents), 0o640); err != nil {
		return "", fmt.Errorf("failed to rewrite %s: %w", lbconfig.ConfigName, err)
	}

	// Assets to be loaded, edited/updated, and regenerated.
	assetsToRegen := []asset.WritableAsset{
		&bootstrap.Bootstrap{},
	}

	store, err := assetstore.NewStore(directory)
	if err != nil {
		return "", fmt.Errorf("failed to create asset store: %w", err)
	}
	for _, a := range assetsToRegen {
		if err := store.Destroy(a); err != nil {
			logrus.Warnf("failed to destroy %s", a.Name())
		}
	}

	// Regenerate the bootstrap ignition file. The regenerated file should use the lbconfig
	// that was generated above with known load balancer issues
	bs := &bootstrap.Bootstrap{}
	if err := store.Fetch(bs); err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", bs.Name(), err)
	}

	// Parse the terraform values. The terraform variables need to be updated with the latest bootstrap ignition data.
	tfvarData := map[string]interface{}{}
	err = json.Unmarshal(tfvarsFile.Data, &tfvarData)
	if err != nil {
		return "", err
	}
	// Update the ignition bootstrap variable to include the lbconfig.
	tfvarData["ignition_bootstrap"] = string(bs.Files()[0].Data)

	// Convert the bootstrap data and write the data back to a file. This will overwrite the original tfvars file.
	jsonBootstrap, err := json.Marshal(tfvarData)
	if err != nil {
		return "", fmt.Errorf("failed to convert bootstrap ignition to bytes: %w", err)
	}

	tfvarsFile.Data = jsonBootstrap

	// update the value on disk to match
	if err := os.WriteFile(fmt.Sprintf("%s/%s", directory, tfvarsFile.Filename), jsonBootstrap, 0o640); err != nil {
		return "", fmt.Errorf("failed to rewrite %s: %w", tfvarsFile.Filename, err)
	}

	return "", nil
}
