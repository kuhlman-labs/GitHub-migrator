output "app_service_id" {
  description = "ID of the App Service"
  value       = azurerm_linux_web_app.main.id
}

output "app_service_name" {
  description = "Name of the App Service"
  value       = azurerm_linux_web_app.main.name
}

output "app_service_default_hostname" {
  description = "Default hostname of the App Service"
  value       = azurerm_linux_web_app.main.default_hostname
}

output "app_service_url" {
  description = "URL of the App Service"
  value       = "https://${azurerm_linux_web_app.main.default_hostname}"
}

output "app_service_plan_id" {
  description = "ID of the App Service Plan"
  value       = azurerm_service_plan.main.id
}

output "app_service_identity_principal_id" {
  description = "Principal ID of the App Service managed identity"
  value       = azurerm_linux_web_app.main.identity[0].principal_id
}

output "app_service_identity_tenant_id" {
  description = "Tenant ID of the App Service managed identity"
  value       = azurerm_linux_web_app.main.identity[0].tenant_id
}

# Staging Slot Outputs
output "staging_slot_id" {
  description = "ID of the staging deployment slot"
  value       = var.enable_staging_slot ? azurerm_linux_web_app_slot.staging[0].id : null
}

output "staging_slot_name" {
  description = "Name of the staging deployment slot"
  value       = var.enable_staging_slot ? azurerm_linux_web_app_slot.staging[0].name : null
}

output "staging_slot_hostname" {
  description = "Default hostname of the staging slot"
  value       = var.enable_staging_slot ? azurerm_linux_web_app_slot.staging[0].default_hostname : null
}

output "staging_slot_url" {
  description = "URL of the staging deployment slot"
  value       = var.enable_staging_slot ? "https://${azurerm_linux_web_app_slot.staging[0].default_hostname}" : null
}

# Dev Slot Outputs
output "dev_slot_id" {
  description = "ID of the dev deployment slot"
  value       = var.enable_dev_slot ? azurerm_linux_web_app_slot.dev[0].id : null
}

output "dev_slot_name" {
  description = "Name of the dev deployment slot"
  value       = var.enable_dev_slot ? azurerm_linux_web_app_slot.dev[0].name : null
}

output "dev_slot_hostname" {
  description = "Default hostname of the dev slot"
  value       = var.enable_dev_slot ? azurerm_linux_web_app_slot.dev[0].default_hostname : null
}

output "dev_slot_url" {
  description = "URL of the dev deployment slot"
  value       = var.enable_dev_slot ? "https://${azurerm_linux_web_app_slot.dev[0].default_hostname}" : null
}