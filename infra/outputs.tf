output "vpc_id" {
  description = "VPC ID"
  value       = aws_vpc.main.id
}

output "public_nat_subnet_id" {
  description = "Public subnet ID that hosts the NAT gateway"
  value       = aws_subnet.public_nat.id
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
