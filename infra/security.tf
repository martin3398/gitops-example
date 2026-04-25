resource "aws_security_group" "control_plane" {
  name        = "${local.name_prefix}-control-plane-sg"
  description = "Security group for Kubernetes control-plane nodes"
  vpc_id      = aws_vpc.main.id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-control-plane-sg"
    Role = "control-plane"
  })
}

resource "aws_vpc_security_group_ingress_rule" "control_plane_admin_ssh" {
  for_each = var.enable_ssh_from_admin_cidrs ? toset(var.allowed_admin_cidrs) : toset([])

  security_group_id = aws_security_group.control_plane.id
  cidr_ipv4         = each.value
  from_port         = 22
  ip_protocol       = "tcp"
  to_port           = 22
  description       = "SSH from admin CIDRs"
}

resource "aws_vpc_security_group_ingress_rule" "control_plane_admin_api" {
  for_each = toset(var.allowed_admin_cidrs)

  security_group_id = aws_security_group.control_plane.id
  cidr_ipv4         = each.value
  from_port         = 6443
  ip_protocol       = "tcp"
  to_port           = 6443
  description       = "Kubernetes API from admin CIDRs"
}

resource "aws_vpc_security_group_ingress_rule" "control_plane_k8s_api_nlb_health" {
  for_each = var.enable_public_k8s_api ? toset(var.lb_public_subnet_cidrs) : toset([])

  security_group_id = aws_security_group.control_plane.id
  cidr_ipv4         = each.value
  from_port         = 6443
  ip_protocol       = "tcp"
  to_port           = 6443
  description       = "Kubernetes API health checks from NLB subnets"
}

resource "aws_security_group" "worker" {
  name        = "${local.name_prefix}-worker-sg"
  description = "Security group for Kubernetes worker nodes"
  vpc_id      = aws_vpc.main.id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-worker-sg"
    Role = "worker"
  })
}

resource "aws_vpc_security_group_ingress_rule" "worker_admin_ssh" {
  for_each = var.enable_ssh_from_admin_cidrs ? toset(var.allowed_admin_cidrs) : toset([])

  security_group_id = aws_security_group.worker.id
  cidr_ipv4         = each.value
  from_port         = 22
  ip_protocol       = "tcp"
  to_port           = 22
  description       = "SSH from admin CIDRs"
}

resource "aws_vpc_security_group_ingress_rule" "control_plane_from_workers_tcp" {
  security_group_id            = aws_security_group.control_plane.id
  referenced_security_group_id = aws_security_group.worker.id
  from_port                    = 0
  ip_protocol                  = "tcp"
  to_port                      = 65535
  description                  = "All TCP from workers"
}

resource "aws_vpc_security_group_ingress_rule" "control_plane_from_workers_udp" {
  security_group_id            = aws_security_group.control_plane.id
  referenced_security_group_id = aws_security_group.worker.id
  from_port                    = 0
  ip_protocol                  = "udp"
  to_port                      = 65535
  description                  = "All UDP from workers"
}

resource "aws_vpc_security_group_ingress_rule" "worker_from_control_plane_tcp" {
  security_group_id            = aws_security_group.worker.id
  referenced_security_group_id = aws_security_group.control_plane.id
  from_port                    = 0
  ip_protocol                  = "tcp"
  to_port                      = 65535
  description                  = "All TCP from control planes"
}

resource "aws_vpc_security_group_ingress_rule" "worker_from_control_plane_udp" {
  security_group_id            = aws_security_group.worker.id
  referenced_security_group_id = aws_security_group.control_plane.id
  from_port                    = 0
  ip_protocol                  = "udp"
  to_port                      = 65535
  description                  = "All UDP from control planes"
}

resource "aws_vpc_security_group_ingress_rule" "control_plane_self_tcp" {
  security_group_id            = aws_security_group.control_plane.id
  referenced_security_group_id = aws_security_group.control_plane.id
  from_port                    = 0
  ip_protocol                  = "tcp"
  to_port                      = 65535
  description                  = "All TCP from control planes"
}

resource "aws_vpc_security_group_ingress_rule" "control_plane_self_udp" {
  security_group_id            = aws_security_group.control_plane.id
  referenced_security_group_id = aws_security_group.control_plane.id
  from_port                    = 0
  ip_protocol                  = "udp"
  to_port                      = 65535
  description                  = "All UDP from control planes"
}

resource "aws_vpc_security_group_ingress_rule" "worker_self_tcp" {
  security_group_id            = aws_security_group.worker.id
  referenced_security_group_id = aws_security_group.worker.id
  from_port                    = 0
  ip_protocol                  = "tcp"
  to_port                      = 65535
  description                  = "All TCP from workers"
}

resource "aws_vpc_security_group_ingress_rule" "worker_self_udp" {
  security_group_id            = aws_security_group.worker.id
  referenced_security_group_id = aws_security_group.worker.id
  from_port                    = 0
  ip_protocol                  = "udp"
  to_port                      = 65535
  description                  = "All UDP from workers"
}

resource "aws_vpc_security_group_ingress_rule" "worker_ingress_http_public" {
  for_each = var.enable_public_ingress ? toset(var.ingress_allowed_cidrs) : toset([])

  security_group_id = aws_security_group.worker.id
  cidr_ipv4         = each.value
  from_port         = var.ingress_nodeport_http
  ip_protocol       = "tcp"
  to_port           = var.ingress_nodeport_http
  description       = "Ingress HTTP NodePort from ingress CIDRs"
}

resource "aws_vpc_security_group_ingress_rule" "worker_ingress_https_public" {
  for_each = var.enable_public_ingress ? toset(var.ingress_allowed_cidrs) : toset([])

  security_group_id = aws_security_group.worker.id
  cidr_ipv4         = each.value
  from_port         = var.ingress_nodeport_https
  ip_protocol       = "tcp"
  to_port           = var.ingress_nodeport_https
  description       = "Ingress HTTPS NodePort from ingress CIDRs"
}

resource "aws_vpc_security_group_ingress_rule" "worker_ingress_http_nlb_health" {
  for_each = var.enable_public_ingress ? toset(var.lb_public_subnet_cidrs) : toset([])

  security_group_id = aws_security_group.worker.id
  cidr_ipv4         = each.value
  from_port         = var.ingress_nodeport_http
  ip_protocol       = "tcp"
  to_port           = var.ingress_nodeport_http
  description       = "Ingress HTTP health checks from NLB subnets"
}

resource "aws_vpc_security_group_ingress_rule" "worker_ingress_https_nlb_health" {
  for_each = var.enable_public_ingress ? toset(var.lb_public_subnet_cidrs) : toset([])

  security_group_id = aws_security_group.worker.id
  cidr_ipv4         = each.value
  from_port         = var.ingress_nodeport_https
  ip_protocol       = "tcp"
  to_port           = var.ingress_nodeport_https
  description       = "Ingress HTTPS health checks from NLB subnets"
}
