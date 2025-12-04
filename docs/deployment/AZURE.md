# Azure App Service Deployment

This guide covers deploying GitHub Migrator to Azure App Services using Terraform and GitHub Actions.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Architecture](#architecture)
- [Azure Setup](#azure-setup)
- [Terraform Deployment](#terraform-deployment)
- [GitHub Actions Setup](#github-actions-setup)
- [Post-Deployment Configuration](#post-deployment-configuration)
- [Monitoring and Operations](#monitoring-and-operations)
- [Troubleshooting](#troubleshooting)

## Overview

This deployment creates:
- **Dev Environment**: Azure App Service with SQLite database
- **Production Environment**: Azure App Service with PostgreSQL Flexible Server
- **CI/CD Pipeline**: GitHub Actions for automated builds and deployments
- **Container Registry**: GitHub Container Registry (GHCR) for Docker images

## Prerequisites

### Required Tools
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) >= 2.0
- [Git](https://git-scm.com/downloads)
- GitHub account with repository admin access

### Required Access
- Azure subscription with contributor access
- Existing Azure Resource Group
- GitHub repository with admin permissions

## Architecture

### Dev Environment
```
GitHub Actions → GHCR → Azure App Service
                         └─ SQLite (local storage)
```

### Production Environment
```
GitHub Actions → GHCR → Azure App Service
                         └─ PostgreSQL Flexible Server
```

## Azure Setup

### 1. Create Azure Service Principal

Create a service principal for GitHub Actions to authenticate with Azure:

```bash
# Login to Azure
az login

# Get your subscription ID
az account show --query id -o tsv

# Create service principal
az ad sp create-for-rbac \
  --name "github-migrator-deploy" \
  --role contributor \
  --scopes /subscriptions/{SUBSCRIPTION_ID}/resourceGroups/{RESOURCE_GROUP_NAME} \
  --sdk-auth
```

Save the JSON output - you'll need it for GitHub secrets.

### 2. Verify Resource Group

Ensure your resource group exists:

```bash
az group show --name YOUR_RESOURCE_GROUP_NAME
```

If it doesn't exist, create it:

```bash
az group create --name YOUR_RESOURCE_GROUP_NAME --location eastus
```

## Terraform Deployment

### Dev Environment Deployment

#### 1. Navigate to Dev Environment

```bash
cd terraform/environments/dev
```

#### 2. Create terraform.tfvars

Copy the example file and fill in your values:

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` with your actual values:

```hcl
# Azure Configuration
azure_subscription_id = "your-subscription-id"
resource_group_name   = "your-resource-group"

# Docker Configuration
docker_image_repository  = "your-github-username/github-migrator"
docker_registry_username = "your-github-username"
docker_registry_password = "your-github-token"

# Application Configuration
source_token      = "your-source-github-token"
destination_token = "your-destination-github-token"
```

#### 3. Initialize and Apply Terraform

```bash
terraform init
terraform plan
terraform apply
```

#### 4. Save Outputs

```bash
terraform output app_service_name
terraform output app_service_url
```

### Production Environment Deployment

Follow the same steps for production:

```bash
cd terraform/environments/prod

cp terraform.tfvars.example terraform.tfvars
vim terraform.tfvars

terraform init
terraform plan
terraform apply

# Save outputs (including database credentials)
terraform output -json > outputs.json
```

**Important**: Store database credentials securely:

```bash
terraform output database_admin_password
```

## GitHub Actions Setup

### 1. Configure Repository Secrets

Navigate to your GitHub repository: Settings → Secrets and variables → Actions

Add the following secrets:

| Secret Name | Description | Example/Notes |
|------------|-------------|---------------|
| `AZURE_CREDENTIALS` | Service principal JSON | Full JSON output from `az ad sp create-for-rbac` |
| `AZURE_SUBSCRIPTION_ID` | Your Azure subscription ID | UUID format |
| `AZURE_RESOURCE_GROUP` | Resource group name | e.g., `github-migrator-rg` |
| `AZURE_APP_SERVICE_NAME_DEV` | Dev App Service name | Output from Terraform dev |
| `AZURE_APP_SERVICE_NAME_PROD` | Production App Service name | Output from Terraform prod |

### 2. Configure GitHub Environments

Create environments for deployment protection:

#### Dev Environment
1. Go to Settings → Environments → New environment
2. Name: `dev`
3. No protection rules needed for dev

#### Production Environment
1. Go to Settings → Environments → New environment
2. Name: `production`
3. Enable required reviewers (recommended)
4. Add yourself as a required reviewer
5. Enable wait timer if desired (e.g., 5 minutes)

### 3. Enable GitHub Container Registry

1. Go to repository Settings → Actions → General
2. Scroll to "Workflow permissions"
3. Select "Read and write permissions"
4. Check "Allow GitHub Actions to create and approve pull requests"

### 4. Test the Workflows

#### Build Workflow
The build workflow runs automatically on push to `main` branch, pull requests, or manual trigger.

#### Deploy to Dev
Automatically runs after successful build on `main` branch.

#### Deploy to Production
Manual trigger or on release - requires approval if configured.

## Post-Deployment Configuration

### 1. Verify Deployment

```bash
# Test health endpoint
curl https://github-migrator-dev.azurewebsites.net/health
# Expected: {"status":"healthy","time":"..."}
```

### 2. Configure GitHub OAuth (Optional)

If you enabled authentication, create an OAuth App on your **SOURCE** GitHub instance:

1. Create GitHub OAuth App:
   - Application name: `GitHub Migrator (Production)`
   - Homepage URL: `https://your-app-name-prod.azurewebsites.net`
   - Callback URL: `https://your-app-name-prod.azurewebsites.net/api/v1/auth/callback`

2. Update Terraform variables:
   ```hcl
   auth_enabled                    = true
   auth_github_oauth_client_id     = "your-oauth-client-id"
   auth_github_oauth_client_secret = "your-oauth-client-secret"
   auth_callback_url               = "https://your-app-url/api/v1/auth/callback"
   auth_frontend_url               = "https://your-app-url"
   ```

3. Reapply Terraform:
   ```bash
   cd terraform/environments/prod
   terraform apply
   ```

### 3. Configure Custom Domain (Optional)

1. In Azure Portal: App Service → Custom domains → Add custom domain
2. Update CORS and OAuth URLs to use custom domain

### 4. Configure SSL/TLS

Azure App Services automatically provides SSL for `*.azurewebsites.net` domains.

For custom domains:
1. Azure Portal → App Service → TLS/SSL settings
2. Upload certificate or use App Service Managed Certificate (free)

## Monitoring and Operations

### Application Logs

```bash
# Stream logs
az webapp log tail --name YOUR_APP_NAME --resource-group YOUR_RG

# Download logs
az webapp log download --name YOUR_APP_NAME --resource-group YOUR_RG
```

### Metrics and Alerts

Configure monitoring in Azure Portal:

1. Navigate to App Service → Monitoring → Metrics
2. Common metrics: CPU %, Memory %, HTTP 5xx errors, Response Time
3. Set up alerts: Alerts → New alert rule

### Database Monitoring (Production)

1. Navigate to PostgreSQL Flexible Server → Monitoring → Metrics
2. Monitor: CPU %, Memory %, Storage Used, Active Connections

## Troubleshooting

### Container Pull Failures

**Problem**: App Service can't pull container image from GHCR

**Solution**:
1. Verify GHCR token has `read:packages` permission
2. Check container image exists: `docker pull ghcr.io/your-username/github-migrator:dev`
3. Verify App Service settings have correct registry credentials

### Database Connection Issues

**Problem**: App can't connect to PostgreSQL

**Solution**:
1. Verify firewall rules allow Azure services
2. Check connection string in App Service configuration
3. Verify database exists and credentials are correct

### App Won't Start

**Problem**: App Service shows "Application Error"

**Solution**:
1. Check logs: `az webapp log tail --name YOUR_APP_NAME --resource-group YOUR_RG`
2. Verify all environment variables are set correctly
3. Ensure container image is valid
4. Check health endpoint path is `/health`

### Git-Sizer Binary Extraction Issues

**Problem**: Logs show errors like `fork/exec /tmp/github-migrator-binaries/git-sizer: no such file or directory`

**Solution**: This is automatically resolved in recent versions. If needed, set:
```bash
az webapp config appsettings set \
  --name YOUR_APP_NAME \
  --resource-group YOUR_RG \
  --settings GHMIG_TEMP_DIR="/home/site/tmp"
```

## Scaling

### Vertical Scaling (Scale Up)

```bash
az appservice plan update \
  --name github-migrator-plan-prod \
  --resource-group YOUR_RG \
  --sku P1v2
```

### Horizontal Scaling (Scale Out)

```bash
az appservice plan update \
  --name github-migrator-plan-prod \
  --resource-group YOUR_RG \
  --number-of-workers 2
```

Or configure auto-scaling in Azure Portal: App Service Plan → Scale out

## Cost Optimization

| Environment | Tier | Estimated Cost |
|-------------|------|----------------|
| Dev | B1 (Basic) | ~$13/month |
| Production | S1 + PostgreSQL B_Standard_B1ms | ~$100/month |

**Tips**:
- Stop dev environment when not in use
- Use deployment slots instead of separate environments
- Enable auto-scaling to scale down during low usage
- Consider reserved instances for long-term savings

## Security Best Practices

1. **Secrets Management**: Use Azure Key Vault for production secrets
2. **Network Security**: Configure CORS, enable HTTPS only, consider Private Endpoints
3. **Authentication**: Enable GitHub OAuth, use strong session secrets (32+ chars)
4. **Database Security**: Use strong passwords, enable SSL, configure firewall rules
5. **Monitoring**: Enable Azure Security Center, configure alerts

## Additional Resources

- [Azure App Services Documentation](https://docs.microsoft.com/en-us/azure/app-service/)
- [PostgreSQL Flexible Server Documentation](https://docs.microsoft.com/en-us/azure/postgresql/flexible-server/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Terraform Azure Provider Documentation](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs)

