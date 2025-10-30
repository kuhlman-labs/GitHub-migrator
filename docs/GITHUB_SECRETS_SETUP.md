# GitHub Secrets Setup Guide

This document lists all GitHub Secrets required for CI/CD pipelines and Terraform deployments.

## 🆕 **Recommended: Use GitHub Environments**

**We now recommend using GitHub Environments** for better organization and security:
- Separate secrets per environment (dev/prod)
- Protection rules for production
- Environment-specific variables for non-sensitive config

👉 **Follow the new guide:** [GITHUB_ENVIRONMENTS_SETUP.md](./GITHUB_ENVIRONMENTS_SETUP.md)

> This guide (GITHUB_SECRETS_SETUP.md) shows the traditional repository-level secrets approach, which still works but is less organized.

## 📍 Where to Add Secrets (Repository-Level)

Navigate to: **Your Repository → Settings → Secrets and variables → Actions**

## 🔐 Required Secrets

### **Azure Infrastructure** (Required for All Workflows)

| Secret Name | Description | How to Get It | Used By |
|------------|-------------|---------------|---------|
| `AZURE_CREDENTIALS` | Service principal JSON for Azure login | See [Create Service Principal](#create-azure-service-principal) | All deploy workflows |
| `AZURE_SUBSCRIPTION_ID` | Your Azure subscription ID | `az account show --query id -o tsv` | Terraform workflows |
| `AZURE_RESOURCE_GROUP` | Azure resource group name | Your existing resource group | Terraform workflows |
| `AZURE_APP_SERVICE_NAME_DEV` | Dev App Service name | `github-migrator-dev` (before terraform) or from terraform output | Deploy Dev workflow |
| `AZURE_APP_SERVICE_NAME_PROD` | Production App Service name | `github-migrator-prod` (before terraform) or from terraform output | Deploy Prod workflow |

### **Application Configuration** (Required for Terraform)

| Secret Name | Description | Example/Notes |
|------------|-------------|---------------|
| `SOURCE_GITHUB_TOKEN` | GitHub PAT for source repositories | `ghp_xxxxx` - needs repo access |
| `DEST_GITHUB_TOKEN` | GitHub PAT for destination repositories | `ghp_xxxxx` - needs repo access (can be same as source) |

### **Authentication - Dev Environment** (Optional)

| Secret Name | Description | How to Get It |
|------------|-------------|---------------|
| `AUTH_ENABLED_DEV` | Enable authentication in dev | `true` or `false` |
| `AUTH_GITHUB_OAUTH_CLIENT_ID_DEV` | GitHub OAuth Client ID (dev) | GitHub OAuth App |
| `AUTH_GITHUB_OAUTH_CLIENT_SECRET_DEV` | GitHub OAuth Client Secret (dev) | GitHub OAuth App |
| `AUTH_SESSION_SECRET_DEV` | Session signing secret (dev) | `openssl rand -base64 32` |

### **Authentication - Production Environment** (Recommended)

| Secret Name | Description | How to Get It |
|------------|-------------|---------------|
| `AUTH_ENABLED_PROD` | Enable authentication in production | `true` |
| `AUTH_GITHUB_OAUTH_CLIENT_ID_PROD` | GitHub OAuth Client ID (prod) | GitHub OAuth App (separate from dev) |
| `AUTH_GITHUB_OAUTH_CLIENT_SECRET_PROD` | GitHub OAuth Client Secret (prod) | GitHub OAuth App |
| `AUTH_SESSION_SECRET_PROD` | Session signing secret (prod) | `openssl rand -base64 32` (different from dev) |

## 🛠️ Step-by-Step Setup

### 1. Create Azure Service Principal

```bash
# Login to Azure
az login

# Get your subscription ID
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
echo "Subscription ID: $SUBSCRIPTION_ID"

# Create service principal with contributor access
az ad sp create-for-rbac \
  --name "github-migrator-deploy" \
  --role contributor \
  --scopes /subscriptions/$SUBSCRIPTION_ID/resourceGroups/YOUR_RESOURCE_GROUP \
  --sdk-auth
```

**Copy the entire JSON output** and save it as the `AZURE_CREDENTIALS` secret.

The output should look like:
```json
{
  "clientId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "clientSecret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "subscriptionId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "tenantId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  ...
}
```

### 2. Get Azure Subscription and Resource Group

```bash
# Get subscription ID
az account show --query id -o tsv

# Verify resource group exists (or create it)
az group show --name YOUR_RESOURCE_GROUP_NAME

# If it doesn't exist, create it:
# az group create --name YOUR_RESOURCE_GROUP_NAME --location eastus
```

### 3. Create GitHub Personal Access Tokens

Create tokens with appropriate scopes:

**For Source/Destination:**
1. Go to: GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Name: `GitHub Migrator - Source`
4. Expiration: Your choice (recommend 90 days or no expiration with rotation)
5. Scopes:
   - ✅ `repo` (Full control of private repositories)
   - ✅ `admin:org` (if migrating org-level resources)
6. Generate and copy the token

Repeat for destination if using different token.

### 4. Set Up GitHub OAuth Apps (Optional - for Authentication)

#### Dev OAuth App

1. Go to: GitHub → Settings → Developer settings → OAuth Apps → New OAuth App
2. Fill in:
   - **Application name:** `GitHub Migrator (Dev)`
   - **Homepage URL:** `https://github-migrator-dev.azurewebsites.net`
   - **Authorization callback URL:** `https://github-migrator-dev.azurewebsites.net/api/v1/auth/callback`
3. Click "Register application"
4. Copy the **Client ID**
5. Click "Generate a new client secret" and copy it
6. Save as `AUTH_GITHUB_OAUTH_CLIENT_ID_DEV` and `AUTH_GITHUB_OAUTH_CLIENT_SECRET_DEV`

#### Production OAuth App

Repeat the above with:
- **Application name:** `GitHub Migrator (Production)`
- **Homepage URL:** `https://github-migrator-prod.azurewebsites.net`
- **Authorization callback URL:** `https://github-migrator-prod.azurewebsites.net/api/v1/auth/callback`

Save as `AUTH_GITHUB_OAUTH_CLIENT_ID_PROD` and `AUTH_GITHUB_OAUTH_CLIENT_SECRET_PROD`

### 5. Generate Session Secrets

```bash
# For dev
openssl rand -base64 32

# For prod (generate a different one)
openssl rand -base64 32
```

Save as `AUTH_SESSION_SECRET_DEV` and `AUTH_SESSION_SECRET_PROD`

## 📋 Quick Checklist

Before running workflows, verify you have:

### Minimum Required (For Deployment Only)
- [ ] `AZURE_CREDENTIALS`
- [ ] `AZURE_APP_SERVICE_NAME_DEV`
- [ ] `AZURE_APP_SERVICE_NAME_PROD`

### For Terraform Infrastructure
- [ ] `AZURE_CREDENTIALS`
- [ ] `AZURE_SUBSCRIPTION_ID`
- [ ] `AZURE_RESOURCE_GROUP`
- [ ] `SOURCE_GITHUB_TOKEN`
- [ ] `DEST_GITHUB_TOKEN`

### For Authentication (Optional)
- [ ] `AUTH_ENABLED_DEV` (or `AUTH_ENABLED_PROD`)
- [ ] `AUTH_GITHUB_OAUTH_CLIENT_ID_DEV`
- [ ] `AUTH_GITHUB_OAUTH_CLIENT_SECRET_DEV`
- [ ] `AUTH_SESSION_SECRET_DEV`
- [ ] (Same for PROD if enabling)

## 🔒 Security Best Practices

1. **Never commit secrets to Git**
   - `.gitignore` excludes `terraform.tfvars`
   - `config.yaml` should use placeholders only

2. **Rotate secrets regularly**
   - Service principals: Every 90 days
   - PATs: Every 90 days
   - OAuth secrets: When compromised

3. **Use different credentials per environment**
   - Dev and prod should have separate OAuth apps
   - Separate session secrets

4. **Limit scope of tokens**
   - Use minimal required permissions
   - Consider using GitHub Apps instead of PATs for better security

5. **Monitor secret usage**
   - Check GitHub Actions logs for failed authentications
   - Review Azure Activity Logs

## 🎯 Testing Your Setup

After adding secrets, test each workflow:

### 1. Test Terraform Dev

```
Actions → Terraform Deploy - Dev → Run workflow
Select: plan
```

Should complete without errors and show planned infrastructure.

### 2. Test Terraform Prod

```
Actions → Terraform Deploy - Production → Run workflow
Select: plan
```

Should complete without errors.

### 3. Apply Infrastructure

After verifying plans:
```
Actions → Terraform Deploy - Dev → Run workflow
Select: apply
```

## 🐛 Troubleshooting

### "Not all values are present" error
- Verify `AZURE_CREDENTIALS` contains complete JSON with `clientId` and `tenantId`

### "Subscription not found" error
- Verify `AZURE_SUBSCRIPTION_ID` is correct
- Verify service principal has access to subscription

### "Resource group not found" error
- Verify resource group exists: `az group show --name YOUR_RG`
- Verify service principal has access to resource group

### "Authentication failed" error
- Verify `SOURCE_GITHUB_TOKEN` and `DEST_GITHUB_TOKEN` are valid
- Check token hasn't expired
- Verify token has required scopes

### OAuth errors at runtime
- Verify callback URLs match exactly (no trailing slashes)
- Check OAuth app is active
- Verify client secret hasn't expired

## 📚 Additional Resources

- [Azure Service Principals Documentation](https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals)
- [GitHub Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
- [GitHub OAuth Apps](https://docs.github.com/en/developers/apps/building-oauth-apps)
- [Terraform Best Practices](https://www.terraform.io/docs/cloud/guides/recommended-practices/index.html)

