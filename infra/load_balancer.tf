resource "aws_lb" "k8s_api" {
  count = var.enable_public_k8s_api ? 1 : 0

  name               = "${var.project_name}-${var.environment}-k8s-api"
  internal           = false
  load_balancer_type = "network"
  subnets            = [for subnet in aws_subnet.public_lb : subnet.id]

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-k8s-api"
    Role = "kubernetes-api"
  })
}

resource "aws_lb_target_group" "k8s_api" {
  count = var.enable_public_k8s_api ? 1 : 0

  name        = "${var.project_name}-${var.environment}-k8s-api"
  port        = 6443
  protocol    = "TCP"
  target_type = "instance"
  vpc_id      = aws_vpc.main.id

  health_check {
    enabled  = true
    protocol = "TCP"
    port     = "6443"
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-k8s-api"
    Role = "kubernetes-api"
  })
}

resource "aws_lb_listener" "k8s_api" {
  count = var.enable_public_k8s_api ? 1 : 0

  load_balancer_arn = aws_lb.k8s_api[0].arn
  port              = 6443
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.k8s_api[0].arn
  }
}

resource "aws_lb_target_group_attachment" "k8s_api_control_plane" {
  for_each = var.enable_public_k8s_api ? aws_instance.control_plane : {}

  target_group_arn = aws_lb_target_group.k8s_api[0].arn
  target_id        = each.value.id
  port             = 6443
}

resource "aws_lb" "ingress" {
  count = var.enable_public_ingress ? 1 : 0

  name               = "${var.project_name}-${var.environment}-ingress"
  internal           = false
  load_balancer_type = "network"
  subnets            = [for subnet in aws_subnet.public_lb : subnet.id]

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-ingress"
    Role = "ingress"
  })
}

resource "aws_lb_target_group" "ingress_http" {
  count = var.enable_public_ingress ? 1 : 0

  name        = "${var.project_name}-${var.environment}-ing-http"
  port        = var.ingress_nodeport_http
  protocol    = "TCP"
  target_type = "instance"
  vpc_id      = aws_vpc.main.id

  health_check {
    enabled  = true
    protocol = "TCP"
    port     = tostring(var.ingress_nodeport_http)
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-ing-http"
    Role = "ingress"
  })
}

resource "aws_lb_target_group" "ingress_https" {
  count = var.enable_public_ingress ? 1 : 0

  name        = "${var.project_name}-${var.environment}-ing-https"
  port        = var.ingress_nodeport_https
  protocol    = "TCP"
  target_type = "instance"
  vpc_id      = aws_vpc.main.id

  health_check {
    enabled  = true
    protocol = "TCP"
    port     = tostring(var.ingress_nodeport_https)
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-ing-https"
    Role = "ingress"
  })
}

resource "aws_lb_listener" "ingress_http" {
  count = var.enable_public_ingress ? 1 : 0

  load_balancer_arn = aws_lb.ingress[0].arn
  port              = 80
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ingress_http[0].arn
  }
}

resource "aws_lb_listener" "ingress_https" {
  count = var.enable_public_ingress ? 1 : 0

  load_balancer_arn = aws_lb.ingress[0].arn
  port              = 443
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ingress_https[0].arn
  }
}

resource "aws_lb_target_group_attachment" "ingress_http_workers" {
  for_each = var.enable_public_ingress ? aws_instance.worker : {}

  target_group_arn = aws_lb_target_group.ingress_http[0].arn
  target_id        = each.value.id
  port             = var.ingress_nodeport_http
}

resource "aws_lb_target_group_attachment" "ingress_https_workers" {
  for_each = var.enable_public_ingress ? aws_instance.worker : {}

  target_group_arn = aws_lb_target_group.ingress_https[0].arn
  target_id        = each.value.id
  port             = var.ingress_nodeport_https
}
