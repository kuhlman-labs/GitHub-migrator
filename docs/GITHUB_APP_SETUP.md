# GitHub App Setup Guide

This guide explains how to set up GitHub App authentication for enhanced discovery and profiling capabilities.

## 📋 Overview

**Do you need a GitHub App?**

GitHub Apps provide additional capabilities beyond Personal Access Tokens (PATs):

| Feature | PAT Only | PAT + GitHub App |
|---------|----------|------------------|
| **Migrations** | ✅ Supported | ✅ Supported |
| **Repository Discovery** | ✅ Basic | ✅ Enhanced |
| **Enterprise-wide Discovery** | ❌ Limited | ✅ Full access |
| **Multi-org Discovery** | ⚠️ Manual | ✅ Automatic |
| **Rate Limits** | User limits | Higher app limits |
| **Audit Trail** | User actions | App actions (clearer) |

**When to use GitHub Apps:**
- ✅ Discovering repos across multiple organizations
- ✅ Enterprise-wide migrations
- ✅ Need higher rate limits
- ✅ Want better audit trails
- ✅ Automated discovery workflows

**When PAT is sufficient:**
- ✅ Single organization
- ✅ Manual repo selection
- ✅ Small number of repos
- ✅ Simple setup preferred

## 🚀 Quick Setup

### Step 1: Create GitHub App

1. Go to your GitHub organization or personal account
   - **Organization**: `Settings → Developer settings → GitHub Apps → New GitHub App`
   - **Personal**: `Settings → Developer settings → GitHub Apps → New GitHub App`

2. Fill in the basic information:
   - **GitHub App name**: `GitHub Migrator - Source` (or your choice)
   - **Homepage URL**: `https://github.com/YOUR_USERNAME/github-migrator`
   - **Webhook**: Uncheck "Active" (not needed for this use case)

3. Set **Repository permissions**:
   - **Contents**: Read-only (to discover repos)
   - **Metadata**: Read-only (required, auto-selected)
   - **Administration**: Read-only (for settings discovery)

4. Set **Organization permissions** (if enterprise-wide discovery):
   - **Members**: Read-only (to list organizations)
   - **Administration**: Read-only (for org-level settings)

5. **Where can this GitHub App be installed?**
   - Select: **Only on this account** (or "Any account" if multi-tenant)

6. Click **Create GitHub App**

### Step 2: Generate Private Key

1. On the GitHub App settings page, scroll to **Private keys**
2. Click **Generate a private key**
3. A `.pem` file will download automatically
4. **Save this file securely** - you'll need it for configuration

### Step 3: Install the GitHub App

1. On the GitHub App settings page, click **Install App** in the left sidebar
2. Select the organization(s) where you want to install it
3. Choose repository access:
   - **All repositories** (for full discovery)
   - **Only select repositories** (for limited discovery)
4. Click **Install**
5. **Note the installation ID** from the URL (e.g., `installations/12345678`)

### Step 4: Get App Credentials

You now have these credentials:

| Credential | Where to Find It | Example |
|------------|------------------|---------|
| **App ID** | App settings page, top section | `123456` |
| **Private Key** | Downloaded `.pem` file | `-----BEGIN RSA PRIVATE KEY-----...` |
| **Installation ID** | URL after installation | `12345678` |

## 🔧 Configuration

### Option 1: Using GitHub Environments (Recommended)

#### Dev Environment

Navigate to: **Settings → Environments → dev**

**Add Variables:**
| Variable Name | Value |
|--------------|-------|
| `SOURCE_APP_ID` | `123456` (your App ID) |
| `SOURCE_APP_INSTALLATION_ID` | `12345678` (your installation ID) |

**Add Secrets:**
| Secret Name | Value |
|------------|-------|
| `SOURCE_APP_PRIVATE_KEY` | Entire contents of `.pem` file |

#### Production Environment

Repeat for production environment (can use same app or different one).

#### Update Workflow

Uncomment the GitHub App lines in your Terraform workflows:

```yaml
# In .github/workflows/terraform-dev.yml and terraform-prod.yml
# Change from:
# source_app_id = ${{ vars.SOURCE_APP_ID }}

# To:
source_app_id = ${{ vars.SOURCE_APP_ID }}
source_app_private_key = "${{ secrets.SOURCE_APP_PRIVATE_KEY }}"
source_app_installation_id = ${{ vars.SOURCE_APP_INSTALLATION_ID }}
```

### Option 2: Direct Configuration (Local Development)

For local development, add to your `config.yaml`:

```yaml
source:
  type: github
  base_url: "https://api.github.com"
  token: "ghp_xxxxx"  # Still required for migrations
  
  # GitHub App configuration
  app_id: 123456
  app_private_key: "./data/app-private-key.pem"  # Path to PEM file
  app_installation_id: 12345678  # Optional: omit to auto-discover
```

**Security Note:** Never commit the private key file to git! Keep it in `./data/` which is gitignored.

## 🎯 Usage Scenarios

### Scenario 1: Single Organization Discovery

**Setup:**
```yaml
source:
  app_id: 123456
  app_private_key: "./data/source-key.pem"
  app_installation_id: 12345678  # Specific org installation
```

**Result:** App discovers all repos in that specific organization.

### Scenario 2: Enterprise-wide Discovery

**Setup:**
```yaml
source:
  app_id: 123456
  app_private_key: "./data/source-key.pem"
  # app_installation_id: omitted
```

**Result:** App auto-discovers all organizations where it's installed.

### Scenario 3: Hybrid Approach

**Setup:**
- PAT for migrations (required by GitHub API)
- GitHub App for discovery and profiling

```yaml
source:
  token: "ghp_xxxxx"           # For migrations
  app_id: 123456                # For discovery
  app_private_key: "./data/key.pem"
```

**Result:** Best of both worlds - enhanced discovery + migration capability.

## 🔒 Security Best Practices

### 1. Protect Private Keys

```bash
# Store in gitignored directory
mkdir -p data
mv ~/Downloads/github-migrator.pem data/app-private-key.pem
chmod 600 data/app-private-key.pem

# Verify it's gitignored
git check-ignore data/app-private-key.pem  # Should output the path
```

### 2. Use Separate Apps for Dev/Prod

**Dev App:**
- Name: "GitHub Migrator - Dev"
- Install on test organizations only

**Production App:**
- Name: "GitHub Migrator - Production"
- Install on production organizations
- More restrictive permissions if possible

### 3. Rotate Keys Regularly

```bash
# Generate new key every 90 days
1. Go to GitHub App settings
2. Generate a private key
3. Update secrets
4. Delete old key after verification
```

### 4. Minimal Permissions

Only request permissions you actually need:
- ✅ **Contents: Read** - For discovering repos
- ✅ **Metadata: Read** - Required by GitHub
- ❌ **Contents: Write** - Not needed for discovery
- ❌ **Administration: Write** - Not needed

### 5. Monitor Usage

```bash
# Check app installations
GitHub App settings → Advanced → Recent Deliveries

# Review audit logs
Organization → Settings → Audit log → Filter by app
```

## 🐛 Troubleshooting

### "Private key is invalid"

**Problem:** Private key not accepted

**Solutions:**
1. Ensure entire PEM file is copied (including BEGIN/END lines)
2. Check for extra newlines or spaces
3. Verify PEM format:
   ```bash
   openssl rsa -in app-private-key.pem -check -noout
   ```
4. Regenerate key if corrupted

### "App not installed"

**Problem:** Cannot access organization

**Solutions:**
1. Verify app is installed: GitHub App → Install App
2. Check installation ID is correct
3. Ensure organization administrators approved installation

### "Insufficient permissions"

**Problem:** Cannot discover repos

**Solutions:**
1. Check app permissions include "Contents: Read"
2. Verify app has access to repositories (not just select ones)
3. Check organization settings allow GitHub Apps

### "Rate limit exceeded"

**Problem:** Hit API rate limits

**Solutions:**
1. GitHub Apps have higher rate limits than PATs
2. Check app is being used (not falling back to PAT)
3. Monitor rate limits in application logs

## 📊 Comparison: PAT vs GitHub App

| Aspect | PAT | GitHub App |
|--------|-----|------------|
| **Setup Complexity** | Simple | Moderate |
| **Multi-org Discovery** | Manual | Automatic |
| **Rate Limits** | 5,000/hour | 15,000/hour |
| **Audit Trail** | User actions | App actions |
| **Permissions** | User-level | Granular |
| **Rotation** | Manual | Key rotation |
| **Best For** | Single org, simple | Enterprise, complex |

## ✅ Verification

After setup, verify it's working:

### 1. Test Locally

```bash
# Run discovery with debug logging
GHMIG_LOGGING_LEVEL=debug ./server

# Check logs for:
# - "GitHub App authenticated"
# - "Discovered X organizations"
```

### 2. Check Terraform Apply

```bash
# Run terraform apply
# Check logs show app ID in configuration
grep "source_app_id" terraform.tfvars
```

### 3. Test Discovery Endpoint

```bash
# Start the server
./server

# Test discovery
curl http://localhost:8080/api/v1/organizations/list

# Should return organizations discovered via app
```

## 🎓 Advanced Configuration

### Multiple Apps for Different Sources

If migrating from multiple GitHub instances:

```yaml
# Dev environment
vars:
  SOURCE_APP_ID: "123456"              # App for github.com
  SOURCE_ENTERPRISE_APP_ID: "789012"  # App for GHE

secrets:
  SOURCE_APP_PRIVATE_KEY: "..."
  SOURCE_ENTERPRISE_APP_PRIVATE_KEY: "..."
```

### JWT-only Authentication (No Installation)

For enterprise-wide discovery without specific installations:

```yaml
source:
  app_id: 123456
  app_private_key: "./data/key.pem"
  # Omit installation_id - uses JWT to discover all installations
```

## 📚 Additional Resources

- [GitHub Apps Documentation](https://docs.github.com/en/developers/apps/getting-started-with-apps/about-apps)
- [Authenticating with GitHub Apps](https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps)
- [GitHub Apps Permissions](https://docs.github.com/en/developers/apps/building-github-apps/creating-a-github-app#permissions)
- [Rate Limits for GitHub Apps](https://docs.github.com/en/developers/apps/rate-limits-for-github-apps)

## 🎉 Summary

You now know how to:
- ✅ Create and configure GitHub Apps
- ✅ Generate and secure private keys
- ✅ Install apps on organizations
- ✅ Configure apps in workflows
- ✅ Use apps for enhanced discovery
- ✅ Troubleshoot common issues

**Next steps:**
1. Decide if you need a GitHub App (see "Do you need a GitHub App?" section)
2. If yes, follow the Quick Setup
3. Add credentials to your environment configuration
4. Test discovery functionality
5. Proceed with migration workflow

Need help? Check the troubleshooting section or refer to the GitHub Apps documentation!

