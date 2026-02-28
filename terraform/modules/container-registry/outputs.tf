output "id" {
  description = "Resource ID of the container registry"
  value       = azurerm_container_registry.main.id
}

output "login_server" {
  description = "Login server URL of the container registry (e.g., myregistry.azurecr.io)"
  value       = azurerm_container_registry.main.login_server
}

output "admin_username" {
  description = "Admin username for the container registry"
  value       = azurerm_container_registry.main.admin_username
  sensitive   = true
}

output "admin_password" {
  description = "Admin password for the container registry"
  value       = azurerm_container_registry.main.admin_password
  sensitive   = true
}
