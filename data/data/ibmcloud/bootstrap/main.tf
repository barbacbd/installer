locals {
  prefix              = var.cluster_id
  port_kubernetes_api = 6443
  port_machine_config = 22623

  # If we need to setup SecurityGroupRules to SSH to bootstrap, for non-public clusters (no Floating IP)
  # combine the Control Plane and Compute subnet CIDRs, to create rules for ingress on those CIDR's
  all_subnet_cidrs = local.public_endpoints ? [] : concat(data.ibm_is_subnet.control_plane_subnets[*].ipv4_cidr_block, data.ibm_is_subnet.compute_subnets[*].ipv4_cidr_block)

  # If a boot volume encryption key CRN was supplied, create a list containing that CRN, otherwise an empty list for a dynamic block of boot volumes
  boot_volume_key_crns = var.ibmcloud_control_plane_boot_volume_key == "" ? [] : [var.ibmcloud_control_plane_boot_volume_key]
}

############################################
# Subnet lookup
############################################
data "ibm_is_subnet" "control_plane_subnets" {
  count = local.public_endpoints ? 0 : length(var.control_plane_subnet_id_list)

  identifier = var.control_plane_subnet_id_list[count.index]
}

data "ibm_is_subnet" "compute_subnets" {
  count = local.public_endpoints ? 0 : length(var.compute_subnet_id_list)

  identifier = var.compute_subnet_id_list[count.index]
}

############################################
# Bootstrap node
############################################

resource "ibm_is_instance" "bootstrap_node" {
  name           = "${local.prefix}-bootstrap"
  image          = var.vsi_image_id
  profile        = var.ibmcloud_bootstrap_instance_type
  resource_group = var.resource_group_id
  tags           = local.tags

  primary_network_interface {
    name            = "eth0"
    subnet          = var.control_plane_subnet_id_list[0]
    security_groups = concat(var.control_plane_security_group_id_list, [ibm_is_security_group.bootstrap.id])
  }

  dynamic "boot_volume" {
    for_each = local.boot_volume_key_crns
    content {
      encryption = boot_volume.value
    }
  }

  dedicated_host = length(var.control_plane_dedicated_host_id_list) > 0 ? var.control_plane_dedicated_host_id_list[0] : null

  vpc  = var.vpc_id
  zone = var.control_plane_subnet_zone_list[0]
  keys = []

  # Use custom ignition config that pulls content from COS bucket
  # TODO: Once support for the httpHeaders field is added to
  # terraform-provider-ignition, we should use it instead of this template.
  # https://github.com/community-terraform-providers/terraform-provider-ignition/issues/16
  user_data = templatefile("${path.module}/templates/bootstrap.ign", {
    HOSTNAME    = replace(ibm_cos_bucket.bootstrap_ignition.s3_endpoint_direct, "https://", "")
    BUCKET_NAME = ibm_cos_bucket.bootstrap_ignition.bucket_name
    OBJECT_NAME = ibm_cos_bucket_object.bootstrap_ignition.key
    IAM_TOKEN   = data.ibm_iam_auth_token.iam_token.iam_access_token
  })
}

############################################
# Floating IP
############################################

resource "ibm_is_floating_ip" "bootstrap_floatingip" {
  count = local.public_endpoints ? 1 : 0

  name           = "${local.prefix}-bootstrap-node-ip"
  resource_group = var.resource_group_id
  target         = ibm_is_instance.bootstrap_node.primary_network_interface.0.id
  tags           = local.tags
}

############################################
# Security group
############################################

resource "ibm_is_security_group" "bootstrap" {
  name           = "${local.prefix}-security-group-bootstrap"
  resource_group = var.resource_group_id
  tags           = local.tags
  vpc            = var.vpc_id
}

# SSH
resource "ibm_is_security_group_rule" "bootstrap_ssh_inbound" {
  count = local.public_endpoints ? 1 : length(local.all_subnet_cidrs)

  group     = ibm_is_security_group.bootstrap.id
  direction = "inbound"
  remote    = local.public_endpoints ? "0.0.0.0/0" : local.all_subnet_cidrs[count.index]
  tcp {
    port_min = 22
    port_max = 22
  }
}

############################################
# Load balancer backend pool members
############################################

resource "ibm_is_lb_pool_member" "kubernetes_api_public" {
  count = local.public_endpoints ? 1 : 0

  lb             = var.lb_kubernetes_api_public_id
  pool           = var.lb_pool_kubernetes_api_public_id
  port           = local.port_kubernetes_api
  target_address = ibm_is_instance.bootstrap_node.primary_network_interface.0.primary_ipv4_address
}

resource "ibm_is_lb_pool_member" "kubernetes_api_private" {
  lb             = var.lb_kubernetes_api_private_id
  pool           = var.lb_pool_kubernetes_api_private_id
  port           = local.port_kubernetes_api
  target_address = ibm_is_instance.bootstrap_node.primary_network_interface.0.primary_ipv4_address
}

resource "ibm_is_lb_pool_member" "machine_config" {
  lb             = var.lb_kubernetes_api_private_id
  pool           = var.lb_pool_machine_config_id
  port           = local.port_machine_config
  target_address = ibm_is_instance.bootstrap_node.primary_network_interface.0.primary_ipv4_address
}
