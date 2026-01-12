output "app_service_url" {
  description = "URL of the App Service (production slot)"
  value       = module.app_service.app_service_url
}

output "app_service_name" {
  description = "Name of the App Service"
  value       = module.app_service.app_service_name
}

output "app_service_default_hostname" {
  description = "Default hostname of the App Service"
  value       = module.app_service.app_service_default_hostname
}

output "app_service_identity_principal_id" {
  description = "Principal ID of the App Service managed identity"
  value       = module.app_service.app_service_identity_principal_id
}

# Deployment Slot Outputs
output "staging_slot_url" {
  description = "URL of the staging deployment slot"
  value       = module.app_service.staging_slot_url
}

output "staging_slot_name" {
  description = "Name of the staging deployment slot"
  value       = module.app_service.staging_slot_name
}

output "dev_slot_url" {
  description = "URL of the dev deployment slot"
  value       = module.app_service.dev_slot_url
}

output "dev_slot_name" {
  description = "Name of the dev deployment slot"
  value       = module.app_service.dev_slot_name
}

# Database Outputs
output "database_server_fqdn" {
  description = "FQDN of the PostgreSQL server"
  value       = module.postgresql.server_fqdn
}

output "database_name" {
  description = "Name of the database"
  value       = module.postgresql.database_name
}

output "database_admin_username" {
  description = "Database admin username"
  value       = module.postgresql.admin_username
  sensitive   = true
}

output "database_admin_password" {
  description = "Database admin password"
  value       = module.postgresql.admin_password
  sensitive   = true
}

