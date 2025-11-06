output "server_id" {
  description = "ID of the PostgreSQL Flexible Server"
  value       = azurerm_postgresql_flexible_server.main.id
}

output "server_name" {
  description = "Name of the PostgreSQL Flexible Server"
  value       = azurerm_postgresql_flexible_server.main.name
}

output "server_fqdn" {
  description = "FQDN of the PostgreSQL Flexible Server"
  value       = azurerm_postgresql_flexible_server.main.fqdn
}

output "database_name" {
  description = "Name of the database"
  value       = azurerm_postgresql_flexible_server_database.main.name
}

output "admin_username" {
  description = "Administrator username"
  value       = azurerm_postgresql_flexible_server.main.administrator_login
  sensitive   = true
}

output "admin_password" {
  description = "Administrator password"
  value       = random_password.admin_password.result
  sensitive   = true
}

output "connection_string" {
  description = "PostgreSQL connection string"
  value       = "host=${azurerm_postgresql_flexible_server.main.fqdn} port=5432 dbname=${azurerm_postgresql_flexible_server_database.main.name} user=${azurerm_postgresql_flexible_server.main.administrator_login} password=${random_password.admin_password.result} sslmode=require"
  sensitive   = true
}

output "dsn" {
  description = "PostgreSQL DSN for the application"
  value       = "postgres://${urlencode(azurerm_postgresql_flexible_server.main.administrator_login)}:${urlencode(random_password.admin_password.result)}@${azurerm_postgresql_flexible_server.main.fqdn}:5432/${azurerm_postgresql_flexible_server_database.main.name}?sslmode=require"
  sensitive   = true
}

