# Terraform Infrastructure

This directory contains Terraform configurations for deploying the GitHub Migrator application to Azure App Services.

## Structure

```
terraform/
├── modules/
│   ├── app-service/       # Azure App Service module
│   └── postgresql/        # PostgreSQL Flexible Server module
└── environments/
    ├── dev/              # Development environment
    └── prod/             # Production environment
```

## Modules

### app-service

Deploys an Azure App Service with:
- Linux App Service Plan
- Docker container support
- GHCR integration
- Health check configuration
- Managed identity
- Minimal application settings (database + port only)

### postgresql

Deploys Azure PostgreSQL Flexible Server with:
- Configurable SKU and storage
- Automated backups
- High availability options
- Firewall rules
- Database creation

## Environments

### Development (dev)

- **Database**: SQLite (embedded, persisted via Azure File Share)
- **App Service**: Basic tier (B1)
- **Always On**: Disabled (to save costs)
- **Purpose**: Testing and development

### Production (prod)

- **Database**: PostgreSQL Flexible Server
- **App Service**: Standard tier (S1) or higher
- **Always On**: Enabled
- **High Availability**: Zone redundant
- **Backups**: 30-day retention with geo-redundancy
- **Purpose**: Production workloads

## Application Configuration

Terraform deploys only the **minimal infrastructure settings** required to run the application:
- Server port (8080)
- Database connection (SQLite for dev, PostgreSQL for prod)

**All other application configuration is done via the Settings UI after deployment:**
- Destination GitHub instance
- Source repositories (via Sources page)
- Migration settings (workers, visibility handling, etc.)
- Authentication (OAuth, authorization rules)
- Logging settings

This approach provides several benefits:
1. **Simpler Terraform**: Fewer variables and secrets to manage
2. **Flexibility**: Configuration can be updated without redeployment
3. **Security**: Sensitive tokens are stored in the application database, not Terraform state
4. **Sensible Defaults**: The application has production-ready defaults for all settings

## Quick Start

### Prerequisites

1. Install [Terraform](https://www.terraform.io/downloads.html) >= 1.0
2. Install [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
3. Login to Azure: `az login`

### Deploy Development Environment

```bash
# Navigate to dev environment
cd environments/dev

# Copy and configure variables
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values

# Initialize Terraform
terraform init

# Preview changes
terraform plan

# Deploy
terraform apply
```

### Deploy Production Environment

```bash
# Navigate to prod environment
cd environments/prod

# Copy and configure variables
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values

# Initialize Terraform
terraform init

# Preview changes
terraform plan

# Deploy
terraform apply

# Save outputs securely
terraform output -json > outputs.json
chmod 600 outputs.json
```

### Post-Deployment Configuration

After deployment, access the application and configure:

1. **Destination GitHub instance** - Go to Settings page and configure your destination GitHub instance (URL and authentication)
2. **Source repositories** - Go to Sources page and add your source GitHub/Azure DevOps instances
3. **Authentication** (optional) - Enable OAuth authentication in Settings
4. **Migration settings** - Customize worker count, visibility handling, etc.

## Required Variables

### Common Variables (both environments)

| Variable | Description | Required |
|----------|-------------|----------|
| `azure_subscription_id` | Azure subscription ID | Yes |
| `resource_group_name` | Resource group name | Yes |
| `docker_image_repository` | GHCR image repository | Yes |
| `docker_registry_username` | GitHub username | Yes |
| `docker_registry_password` | GitHub token with packages read | Yes |

### Production-Specific Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `database_sku` | PostgreSQL SKU | `GP_Standard_D2s_v3` |
| `database_storage_mb` | Storage size in MB | `32768` |
| `database_backup_retention_days` | Backup retention | `30` |
| `database_high_availability_mode` | HA mode | `ZoneRedundant` |

## Outputs

### Development Environment

- `app_service_url` - URL of the deployed application
- `app_service_name` - Name of the App Service
- `app_service_identity_principal_id` - Managed identity principal ID

### Production Environment

All dev outputs plus:
- `database_server_fqdn` - PostgreSQL server FQDN
- `database_name` - Database name
- `database_admin_username` - Database admin username (sensitive)
- `database_admin_password` - Database admin password (sensitive)

## State Management

### Local State (Default)

By default, Terraform state is stored locally in `terraform.tfstate`.

**Warning**: Local state should not be used for production. Multiple users or CI/CD pipelines will conflict.

### Remote State (Recommended for Production)

Configure Azure Storage backend for state:

1. Create storage account for Terraform state:

```bash
# Create resource group for Terraform state
az group create --name terraform-state-rg --location eastus

# Create storage account
az storage account create \
  --name tfstate$(uuidgen | tr -d '-' | tr '[:upper:]' '[:lower:]' | cut -c1-17) \
  --resource-group terraform-state-rg \
  --location eastus \
  --sku Standard_LRS

# Create container
az storage container create \
  --name tfstate \
  --account-name YOUR_STORAGE_ACCOUNT_NAME
```

2. Uncomment and configure backend in `main.tf`:

```hcl
backend "azurerm" {
  resource_group_name  = "terraform-state-rg"
  storage_account_name = "YOUR_STORAGE_ACCOUNT_NAME"
  container_name       = "tfstate"
  key                  = "github-migrator-prod.tfstate"
}
```

3. Initialize with backend:

```bash
terraform init -migrate-state
```

## Managing Secrets

### Sensitive Variables

Never commit sensitive values to Git. Use one of these methods:

#### 1. terraform.tfvars (Local Development)

```hcl
# terraform.tfvars (add to .gitignore)
docker_registry_password = "ghp_xxxxxxxxxx"
```

#### 2. Environment Variables

```bash
export TF_VAR_docker_registry_password="ghp_xxxxxxxxxx"
terraform apply
```

#### 3. Azure Key Vault (Recommended for Production)

Store secrets in Azure Key Vault and reference them:

```bash
# Store secret in Key Vault
az keyvault secret set \
  --vault-name YOUR_VAULT \
  --name docker-registry-password \
  --value "ghp_xxxxxxxxxx"

# Retrieve in Terraform (requires additional configuration)
```

### Application Secrets

Application secrets (GitHub tokens, OAuth credentials, etc.) are configured via the Settings UI and stored securely in the application database. They are not managed by Terraform.

## Updating the Infrastructure

### Update Docker Image Tag

```bash
# Update variable
terraform apply -var="docker_image_tag=v1.2.0"
```

Or update in `terraform.tfvars`:

```hcl
docker_image_tag = "v1.2.0"
```

Then apply:

```bash
terraform apply
```

### Scale Up/Down

Update the SKU in `terraform.tfvars`:

```hcl
# Scale up to Standard tier
app_service_sku = "S2"
```

Then apply:

```bash
terraform apply
```

## Destroying Infrastructure

**Warning**: This will permanently delete all resources and data.

### Development Environment

```bash
cd environments/dev
terraform destroy
```

### Production Environment

**Backup database first!**

```bash
cd environments/prod

# Backup database
az postgres flexible-server backup create \
  --resource-group YOUR_RG \
  --name YOUR_DB_SERVER

# Then destroy
terraform destroy
```

## Troubleshooting

### Error: Resource Group Not Found

Ensure the resource group exists:

```bash
az group show --name YOUR_RESOURCE_GROUP
```

Create if needed:

```bash
az group create --name YOUR_RESOURCE_GROUP --location eastus
```

### Error: Provider Configuration

Ensure Azure CLI is logged in:

```bash
az login
az account show
```

Set correct subscription:

```bash
az account set --subscription YOUR_SUBSCRIPTION_ID
```

### Error: State Lock

If Terraform state is locked:

```bash
# Force unlock (use with caution)
terraform force-unlock LOCK_ID
```

### Container Pull Errors

Verify GHCR credentials:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_USERNAME --password-stdin
docker pull ghcr.io/YOUR_USERNAME/github-migrator:dev
```

## Best Practices

1. **Always run `terraform plan` before `apply`**
2. **Use remote state for production**
3. **Store sensitive outputs securely**
4. **Tag all resources appropriately**
5. **Use workspaces or separate directories for environments**
6. **Keep modules version-pinned in production**
7. **Document all customizations**
8. **Regular state backups**
9. **Use `.tfvars` files, never hardcode secrets**
10. **Review and audit changes in production**

## Additional Resources

- [Terraform Azure Provider](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs)
- [Azure App Service Terraform](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/linux_web_app)
- [Azure PostgreSQL Terraform](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/postgresql_flexible_server)
- [Terraform Best Practices](https://www.terraform.io/docs/cloud/guides/recommended-practices/index.html)

## Support

For deployment issues, see [AZURE_DEPLOYMENT.md](../docs/AZURE_DEPLOYMENT.md) for detailed troubleshooting.
