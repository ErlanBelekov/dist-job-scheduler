output "server_ip" {
  description = "Public IPv4 address of the k3s server"
  value       = hcloud_server.k3s.ipv4_address
}

output "ssh_command" {
  description = "SSH command to connect to the server"
  value       = "ssh root@${hcloud_server.k3s.ipv4_address}"
}
