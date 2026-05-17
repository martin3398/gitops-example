data "aws_ami" "ubuntu_2204" {
  count = var.ami_id == null ? 1 : 0

  owners      = ["099720109477"]
  most_recent = true

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

locals {
  selected_ami_id = var.ami_id != null ? var.ami_id : data.aws_ami.ubuntu_2204[0].id
}

resource "aws_instance" "control_plane" {
  for_each = local.control_plane_nodes

  ami                         = local.selected_ami_id
  instance_type               = var.control_plane_instance_type
  subnet_id                   = aws_subnet.private[tostring(each.value.az_index)].id
  vpc_security_group_ids      = [aws_security_group.control_plane.id]
  iam_instance_profile        = aws_iam_instance_profile.node.name
  key_name                    = var.ssh_key_name
  associate_public_ip_address = false

  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 2
  }

  root_block_device {
    volume_type           = "gp3"
    volume_size           = var.root_volume_size_gb
    encrypted             = true
    delete_on_termination = true
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-${each.key}"
    Role = "control-plane"
    Node = each.key
  })
}

resource "aws_instance" "worker" {
  for_each = local.worker_nodes

  ami                         = local.selected_ami_id
  instance_type               = var.worker_instance_type
  subnet_id                   = aws_subnet.private[tostring(each.value.az_index)].id
  vpc_security_group_ids      = [aws_security_group.worker.id]
  iam_instance_profile        = aws_iam_instance_profile.node.name
  key_name                    = var.ssh_key_name
  associate_public_ip_address = false

  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 2
  }

  root_block_device {
    volume_type           = "gp3"
    volume_size           = var.root_volume_size_gb
    encrypted             = true
    delete_on_termination = true
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-${each.key}"
    Role = "worker"
    Node = each.key
  })
}

resource "aws_ebs_volume" "worker_ceph_osd" {
  for_each = local.ceph_worker_nodes

  availability_zone = local.selected_azs[each.value.az_index]
  size              = var.ceph_osd_volume_size_gb
  type              = "gp3"
  encrypted         = true

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-${each.key}-ceph-osd-1"
    Role = "worker-ceph-osd"
    Node = each.key
  })
}

resource "aws_volume_attachment" "worker_ceph_osd" {
  for_each = local.ceph_worker_nodes

  device_name = "/dev/sdf"
  volume_id   = aws_ebs_volume.worker_ceph_osd[each.key].id
  instance_id = aws_instance.worker[each.key].id
}
