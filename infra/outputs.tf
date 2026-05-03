output "vpc_id" {
  description = "VPC ID"
  value       = aws_vpc.main.id
}

output "public_nat_subnet_id" {
  description = "Public subnet ID that hosts the NAT gateway"
  value       = aws_subnet.public_nat.id
}

output "public_lb_subnet_ids" {
  description = "Public subnet IDs used by internet-facing NLBs"
  value = {
    for idx, subnet in aws_subnet.public_lb : idx => subnet.id
  }
}

output "private_subnet_ids" {
  description = "Private subnet IDs keyed by index"
  value = {
    for idx, subnet in aws_subnet.private : idx => subnet.id
  }
}

output "nat_gateway_id" {
  description = "NAT gateway ID"
  value       = aws_nat_gateway.main.id
}

output "control_plane_private_ips" {
  description = "Private IPs of control-plane nodes"
  value = {
    for node, instance in aws_instance.control_plane : node => instance.private_ip
  }
}

output "worker_private_ips" {
  description = "Private IPs of worker nodes"
  value = {
    for node, instance in aws_instance.worker : node => instance.private_ip
  }
}

output "instance_ids" {
  description = "All node instance IDs by role"
  value = {
    control_plane = {
      for node, instance in aws_instance.control_plane : node => instance.id
    }
    workers = {
      for node, instance in aws_instance.worker : node => instance.id
    }
  }
}

output "ansible_inventory" {
  description = "Inventory-style node details for Ansible"
  value = {
    kube_api_internal_endpoint = "${aws_instance.control_plane["cp-1"].private_ip}:6443"
    kube_api_public_endpoint   = var.enable_public_k8s_api ? "${aws_lb.k8s_api[0].dns_name}:6443" : "${aws_instance.control_plane["cp-1"].private_ip}:6443"
    control_plane = {
      for node, instance in aws_instance.control_plane : node => {
        private_ip  = instance.private_ip
        instance_id = instance.id
        az          = instance.availability_zone
      }
    }
    workers = {
      for node, instance in aws_instance.worker : node => {
        private_ip  = instance.private_ip
        instance_id = instance.id
        az          = instance.availability_zone
      }
    }
  }
}

output "kubernetes_api_endpoint" {
  description = "Kubernetes API endpoint for external kubeconfig usage"
  value       = var.enable_public_k8s_api ? "${aws_lb.k8s_api[0].dns_name}:6443" : "${aws_instance.control_plane["cp-1"].private_ip}:6443"
}

output "kubernetes_api_internal_endpoint" {
  description = "Kubernetes API endpoint used by kubeadm control-plane bootstrap"
  value       = "${aws_instance.control_plane["cp-1"].private_ip}:6443"
}

output "ingress_public_endpoint" {
  description = "Ingress NLB endpoint (validated for HTTP in current lab scope)"
  value       = var.enable_public_ingress ? aws_lb.ingress[0].dns_name : ""
}
