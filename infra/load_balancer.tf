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
