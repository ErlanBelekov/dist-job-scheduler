resource "hcloud_ssh_key" "default" {
  name       = "dist-scheduler"
  public_key = file(pathexpand(var.ssh_public_key_path))
}

resource "hcloud_firewall" "default" {
  name = "dist-scheduler"

  # SSH — restrict to known IPs only
  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "22"
    source_ips = var.allowed_admin_ips
  }

  # HTTP (server API)
  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "80"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  # HTTPS
  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "443"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  # Kubernetes API — restrict to known IPs only
  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "6443"
    source_ips = var.allowed_admin_ips
  }
}

resource "hcloud_server" "k3s" {
  name        = "dist-scheduler"
  server_type = var.server_type
  location    = var.location
  image       = "ubuntu-24.04"

  ssh_keys = [hcloud_ssh_key.default.id]

  firewall_ids = [hcloud_firewall.default.id]

  labels = {
    purpose = "dist-scheduler"
  }
}
