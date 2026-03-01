variable "hcloud_token" {
  description = "Hetzner Cloud API token"
  type        = string
  sensitive   = true
}

variable "ssh_public_key_path" {
  description = "Path to SSH public key file"
  type        = string
  default     = "~/.ssh/id_rsa_personal.pub"
}

variable "server_type" {
  description = "Hetzner server type"
  type        = string
  default     = "cx23"
}

variable "location" {
  description = "Hetzner datacenter location"
  type        = string
  default     = "nbg1" # Nuremberg, Germany
}

variable "allowed_admin_ips" {
  description = "CIDR blocks allowed to access SSH and Kubernetes API (e.g. your home/VPN IP)"
  type        = list(string)
  default = ["31.192.254.212/32"]
}
