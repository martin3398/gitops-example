data "aws_availability_zones" "available" {
  state = "available"
}

locals {
  name_prefix = "${var.project_name}-${var.environment}"

  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "opentofu"
  }

  selected_azs = slice(data.aws_availability_zones.available.names, 0, 3)

  control_plane_nodes = {
    cp-1 = { az_index = 0 }
    cp-2 = { az_index = 1 }
    cp-3 = { az_index = 2 }
  }

  worker_nodes = {
    worker-1 = { az_index = 0 }
    worker-2 = { az_index = 1 }
  }
}
