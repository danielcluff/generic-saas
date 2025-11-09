# Application Configuration

This document explains how to customize the branding and configuration of your SaaS platform.

## Overview

The application now uses centralized configuration to manage branding, URLs, and other customizable settings. This allows you to change the name and appearance of your platform in one place rather than hunting through multiple files.

## Backend Configuration

### Location
`backend/internal/config/app.go`

### Environment Variables

You can customize your application by setting these environment variables:

```bash
# Brand and Identity
APP_NAME="MyPlatform"                    # Internal app name
APP_DISPLAY_NAME="MyPlatform"            # Name shown to users
COMPANY_NAME="MyPlatform Inc."           # Company name for legal text
SUPPORT_EMAIL="support@myplatform.com"  # Support contact email

# URLs
APP_BASE_URL="https://app.myplatform.com"              # Base application URL
DASHBOARD_URL="https://app.myplatform.com/dashboard"   # Dashboard URL
VERIFICATION_BASE_URL="https://app.myplatform.com/verify"  # Email verification base
SECURITY_URL="https://app.myplatform.com/settings/security"  # Security settings URL

# Email Configuration
EMAIL_FROM_DOMAIN="myplatform.com"      # Domain for outgoing emails
EMAIL_FROM_NAME="MyPlatform"            # Name shown in email "From" field
EMAIL_MAILER="MyPlatform-Auth"          # X-Mailer header value

# Metadata
APP_DESCRIPTION="Your custom platform description"
APP_VERSION="1.0.0"
```

### Default Values

If no environment variables are set, the application defaults to "SaaSPlatform" branding.

## Frontend Configuration

### Location
`frontend/src/config/app.ts`

### Environment Variables

Set these in your `.env` file or deployment environment:

```bash
# Brand and Identity
PUBLIC_APP_NAME="MyPlatform"
PUBLIC_APP_DISPLAY_NAME="MyPlatform"
PUBLIC_COMPANY_NAME="MyPlatform Inc."
PUBLIC_SUPPORT_EMAIL="support@myplatform.com"

# URLs
PUBLIC_APP_BASE_URL="https://app.myplatform.com"
PUBLIC_DASHBOARD_URL="/dashboard"

# Metadata
PUBLIC_APP_DESCRIPTION="Your custom platform description"
PUBLIC_APP_VERSION="1.0.0"

# API
PUBLIC_API_BASE_URL="http://localhost:8080"
```

## Usage Examples

### Backend Usage

```go
import "github.com/danielsaas/generic-saas/internal/config"

// Get app configuration
appConfig := config.GetAppConfig()

// Use in email subjects
subject := appConfig.GetWelcomeSubject()  // "Welcome to MyPlatform!"

// Get branded URLs
verificationURL := appConfig.GetVerificationURL(token)
```

### Frontend Usage

```typescript
import config, { getBrandingTitle, getPageTitle, getCopyright } from '../config/app.ts';

// Use in components
<span>{config.appDisplayName}</span>

// Use in page titles
<Layout title={getPageTitle("Settings")} />  // "Settings - MyPlatform"

// Use for copyright
<footer>{getCopyright()}</footer>  // "© 2024 MyPlatform Inc. All rights reserved."
```

## Quick Start

To rebrand your platform:

1. **Choose your platform name** (e.g., "MyPlatform")

2. **Set backend environment variables:**
   ```bash
   export APP_NAME="MyPlatform"
   export APP_DISPLAY_NAME="MyPlatform"
   export COMPANY_NAME="MyPlatform Inc."
   export EMAIL_FROM_DOMAIN="myplatform.com"
   ```

3. **Set frontend environment variables:**
   ```bash
   # In frontend/.env
   PUBLIC_APP_NAME="MyPlatform"
   PUBLIC_APP_DISPLAY_NAME="MyPlatform"
   PUBLIC_COMPANY_NAME="MyPlatform Inc."
   ```

4. **Restart your application** to pick up the new configuration

## Files Updated

The following files now use the centralized configuration:

### Backend
- `internal/email/email.go` - Email headers
- `internal/email/sendgrid.go` - Email templates and subjects
- `internal/email/smtp.go` - Email subjects
- `internal/email/ses.go` - Email subjects
- `internal/email/tokens.go` - Verification URLs

### Frontend
- `src/components/layout/Layout.astro` - Default page description
- `src/components/layout/Header.astro` - Brand name in header
- `src/pages/index.astro` - Page title
- `src/pages/auth.astro` - Page title

## Benefits

- **Single source of truth** for branding
- **Environment-based configuration** for different deployments
- **Easy rebranding** without code changes
- **Consistent branding** across all components
- **Professional appearance** with customizable company information