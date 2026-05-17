variable "aws_region" {
  description = "AWS region for all resources"
  type        = string
  default     = "eu-central-1"
}

variable "project_name" {
  description = "Project name used in tags and resource naming"
  type        = string
  default     = "gitops-showcase"
}

variable "environment" {
  description = "Environment name for tags"
  type        = string
  default     = "dev"
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.42.0.0/16"
}

variable "private_subnet_cidrs" {
  description = "Three private subnet CIDRs for Kubernetes nodes"
  type        = list(string)
  default     = ["10.42.10.0/24", "10.42.20.0/24", "10.42.30.0/24"]

  validation {
    condition     = length(var.private_subnet_cidrs) == 3
    error_message = "Exactly three private subnet CIDRs are required."
  }
}

variable "public_subnet_cidr" {
  description = "Public subnet CIDR used for NAT gateway"
  type        = string
  default     = "10.42.100.0/24"
}

variable "enable_public_k8s_api" {
  description = "Expose Kubernetes API through an internet-facing NLB"
  type        = bool
  default     = false
}

variable "enable_public_ingress" {
  description = "Expose ingress traffic through an internet-facing NLB"
  type        = bool
  default     = false
}

variable "lb_public_subnet_cidrs" {
  description = "Public subnet CIDRs used by internet-facing NLBs"
  type        = list(string)
  default     = ["10.42.101.0/24", "10.42.102.0/24"]

  validation {
    condition     = length(var.lb_public_subnet_cidrs) >= 2 && length(var.lb_public_subnet_cidrs) <= 3
    error_message = "Kubernetes API NLB requires 2-3 public subnet CIDRs."
  }
}

variable "ingress_allowed_cidrs" {
  description = "CIDRs allowed to access ingress through the ingress NLB"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "ingress_nodeport_http" {
  description = "NodePort exposed by ingress-nginx for HTTP"
  type        = number
  default     = 30080

  validation {
    condition     = var.ingress_nodeport_http >= 30000 && var.ingress_nodeport_http <= 32767
    error_message = "ingress_nodeport_http must be in Kubernetes NodePort range 30000-32767."
  }
}

variable "ingress_nodeport_https" {
  description = "Reserved NodePort for HTTPS ingress (currently out of scope for this AWS lab)"
  type        = number
  default     = 30443

  validation {
    condition     = var.ingress_nodeport_https >= 30000 && var.ingress_nodeport_https <= 32767
    error_message = "ingress_nodeport_https must be in Kubernetes NodePort range 30000-32767."
  }
}

variable "allowed_admin_cidrs" {
  description = "CIDRs allowed for administrative access"
  type        = list(string)
  default     = []
}

variable "enable_ssh_from_admin_cidrs" {
  description = "Enable SSH ingress from allowed_admin_cidrs"
  type        = bool
  default     = false
}

variable "control_plane_instance_type" {
  description = "EC2 instance type for control-plane nodes"
  type        = string
  default     = "t3.medium"
}

variable "worker_instance_type" {
  description = "EC2 instance type for worker nodes"
  type        = string
  default     = "m6i.xlarge"
}

variable "root_volume_size_gb" {
  description = "Root EBS volume size in GiB"
  type        = number
  default     = 40
}

variable "ceph_osd_volume_size_gb" {
  description = "Ceph OSD EBS volume size in GiB for each worker node"
  type        = number
  default     = 20
}

variable "ssh_key_name" {
  description = "Optional EC2 key pair name for SSH access"
  type        = string
  default     = null
}

variable "ami_id" {
  description = "Optional explicit AMI ID. If null, latest Ubuntu 22.04 LTS is used."
  type        = string
  default     = null
}
