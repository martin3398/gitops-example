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
  default     = "lab"
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
  default     = "t3.medium"
}

variable "root_volume_size_gb" {
  description = "Root EBS volume size in GiB"
  type        = number
  default     = 40
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
