# OAuth Setup Guide

This guide provides step-by-step instructions for setting up OAuth authentication with Google and Microsoft providers in the Homa application.

## Overview

Homa supports OAuth2 authentication with:
- **Google** - Using Google OAuth 2.0
- **Microsoft** - Using Microsoft Azure AD

**Important**: OAuth login only works for existing users. Users cannot register through OAuth - they must be created by an administrator first.

## Prerequisites

1. Homa application running with database configured
2. Admin access to Google Cloud Console (for Google OAuth)
3. Admin access to Microsoft Azure Portal (for Microsoft OAuth)

## Google OAuth Setup

### Step 1: Create Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google+ API and Google OAuth2 API

### Step 2: Configure OAuth Consent Screen

1. Navigate to **APIs & Services** → **OAuth consent screen**
2. Choose **External** user type (for testing) or **Internal** (for organization-only)
3. Fill in the required information:
   - **App name**: Homa Support System
   - **User support email**: Your email
   - **Developer contact information**: Your email
4. Add scopes:
   - `userinfo.email`
   - `userinfo.profile`
5. Save and continue

### Step 3: Create OAuth Credentials

1. Navigate to **APIs & Services** → **Credentials**
2. Click **+ CREATE CREDENTIALS** → **OAuth client ID**
3. Choose **Web application**
4. Configure:
   - **Name**: Homa OAuth Client
   - **Authorized JavaScript origins**: `http://localhost:8000`
   - **Authorized redirect URIs**: `http://localhost:8000/auth/oauth/google/callback`
5. Click **Create**
6. Copy the **Client ID** and **Client Secret**

### Step 4: Configure Homa Application

Add the Google OAuth credentials to your application configuration:

```bash
# Set environment variables
export GOOGLE_OAUTH_CLIENT_ID="your-google-client-id"
export GOOGLE_OAUTH_CLIENT_SECRET="your-google-client-secret"

# Or add to your config file (config.dev.yml)
OAuth:
  Google:
    ClientID: "your-google-client-id"
    ClientSecret: "your-google-client-secret"
    Enabled: true
```

## Microsoft OAuth Setup

### Step 1: Register Application in Azure

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to **Azure Active Directory** → **App registrations**
3. Click **+ New registration**
4. Configure:
   - **Name**: Homa Support System
   - **Supported account types**: Accounts in any organizational directory and personal Microsoft accounts
   - **Redirect URI**: Web → `http://localhost:8000/auth/oauth/microsoft/callback`
5. Click **Register**

### Step 2: Configure API Permissions

1. In your app registration, go to **API permissions**
2. Click **+ Add a permission**
3. Choose **Microsoft Graph**
4. Select **Delegated permissions**
5. Add these permissions:
   - `User.Read` (to read user profile)
6. Click **Add permissions**
7. Click **Grant admin consent** (if you have admin rights)

### Step 3: Create Client Secret

1. Go to **Certificates & secrets**
2. Click **+ New client secret**
3. Add description: "Homa OAuth Secret"
4. Choose expiration (24 months recommended)
5. Click **Add**
6. **Copy the secret value immediately** (it won't be shown again)

### Step 4: Get Application ID

1. Go to **Overview** tab
2. Copy the **Application (client) ID**

### Step 5: Configure Homa Application

Add the Microsoft OAuth credentials to your application configuration:

```bash
# Set environment variables
export MICROSOFT_OAUTH_CLIENT_ID="your-azure-app-id"
export MICROSOFT_OAUTH_CLIENT_SECRET="your-azure-client-secret"

# Or add to your config file (config.dev.yml)
OAuth:
  Microsoft:
    ClientID: "your-azure-app-id"
    ClientSecret: "your-azure-client-secret"
    Enabled: true
```

## Application Configuration

### Environment Variables

The application looks for these environment variables:

```bash
# Google OAuth
GOOGLE_OAUTH_CLIENT_ID=your-google-client-id
GOOGLE_OAUTH_CLIENT_SECRET=your-google-client-secret

# Microsoft OAuth
MICROSOFT_OAUTH_CLIENT_ID=your-azure-app-id
MICROSOFT_OAUTH_CLIENT_SECRET=your-azure-client-secret
```

### Configuration File (Recommended)

Add OAuth configuration to your `config.dev.yml`:

```yaml
OAuth:
  Google:
    ClientID: "your-google-client-id"
    Secret: "your-google-client-secret"
    Enabled: true
  Microsoft:
    ClientID: "your-azure-app-id"
    Secret: "your-azure-client-secret"
    Enabled: true
```

**Enabling/Disabling Providers:**

To disable a provider, set `Enabled: false` or omit the provider configuration entirely:

```yaml
OAuth:
  Google:
    ClientID: "your-google-client-id"
    Secret: "your-google-client-secret"
    Enabled: true
  Microsoft:
    # Microsoft OAuth disabled by omitting configuration
    Enabled: false
```

**Dynamic Provider Loading:**

The `/auth/oauth/providers` API endpoint will only return enabled providers. OAuth routes are also conditionally registered based on the `Enabled` setting, so disabled providers won't have accessible endpoints.

## Testing OAuth Setup

### Prerequisites for Testing

1. Create a test user first:
```bash
go run main.go -c config.dev.yml --create-admin -email test@example.com -password testpass123 -name Test -lastname User
```

### Using the Test Interface

1. Start the application:
```bash
go run main.go -c config.dev.yml
```

2. Open the test login page:
```
http://localhost:8000/static/login.html
```

3. Click on "Login with Google" or "Login with Microsoft"
4. Complete the OAuth flow
5. Verify you receive a JWT token response

### API Endpoints

The following OAuth endpoints are available:

- `GET /auth/oauth/google` - Initiates Google OAuth login
- `GET /auth/oauth/google/callback` - Handles Google OAuth callback
- `GET /auth/oauth/microsoft` - Initiates Microsoft OAuth login
- `GET /auth/oauth/microsoft/callback` - Handles Microsoft OAuth callback
- `GET /api/oauth-providers` - Returns available OAuth providers

## Production Considerations

### Security

1. **HTTPS Required**: OAuth providers require HTTPS in production
2. **Update Redirect URIs**: Change `localhost:8000` to your production domain
3. **Environment Variables**: Store secrets in environment variables, not config files
4. **Secret Rotation**: Regularly rotate OAuth client secrets

### Configuration Updates for Production

1. **Google Console**:
   - Update authorized origins: `https://yourdomain.com`
   - Update redirect URI: `https://yourdomain.com/auth/oauth/google/callback`

2. **Azure Portal**:
   - Update redirect URI: `https://yourdomain.com/auth/oauth/microsoft/callback`

3. **Application Configuration**:
```bash
export GOOGLE_OAUTH_CLIENT_ID="your-production-google-client-id"
export GOOGLE_OAUTH_CLIENT_SECRET="your-production-google-client-secret"
export MICROSOFT_OAUTH_CLIENT_ID="your-production-azure-app-id"
export MICROSOFT_OAUTH_CLIENT_SECRET="your-production-azure-client-secret"
```

## Troubleshooting

### Common Issues

1. **"OAuth not configured" error**:
   - Verify client ID and secret are set
   - Check environment variables or config file

2. **"redirect_uri_mismatch" error**:
   - Ensure redirect URIs match exactly in OAuth provider settings
   - Check for trailing slashes or protocol mismatches

3. **"User not found" error**:
   - The user must exist in the database before OAuth login
   - Create users via admin interface or CLI command

4. **"Invalid client" error**:
   - Verify client ID and secret are correct
   - Check if OAuth application is enabled in provider console

### Debug Mode

Enable debug logging to troubleshoot OAuth issues:

```bash
# Set debug level in config
Debug: "3"

# Check application logs for detailed OAuth flow information
```

## User Management

### Creating Users for OAuth

Users must be created before they can use OAuth login:

```bash
# Create admin user
go run main.go -c config.dev.yml --create-admin -email admin@company.com -password securepass -name Admin -lastname User

# Users can then use OAuth with their existing email address
```

### OAuth Authentication Flow

When a user logs in via OAuth:
1. User initiates OAuth flow (Google or Microsoft)
2. After successful OAuth authentication, system retrieves user email from provider
3. System searches for existing user with matching email address
4. If user found: generates JWT tokens and logs them in
5. If user not found: returns error (no registration via OAuth)

**Simplified Design**: No separate OAuth account linking - authentication is purely email-based. The system only stores user login history, not OAuth provider associations.

## Frontend Implementation

### OAuth Flow Overview

The OAuth authentication flow for frontend applications follows these steps:

1. **Get Available Providers**: Query `/auth/oauth/providers` to get enabled OAuth providers
2. **Initiate OAuth**: Redirect user to OAuth endpoint with mandatory `redirect_url` parameter  
3. **Handle Callback**: Process OAuth callback response with JWT tokens
4. **Store Tokens**: Store access/refresh tokens for authenticated API calls

### Implementation Flow

```javascript
// Step 1: Get available OAuth providers
async function getOAuthProviders() {
    const response = await fetch('/auth/oauth/providers');
    const data = await response.json();
    return data.providers;
}

// Step 2: Initiate OAuth login
function loginWithGoogle() {
    const redirectUrl = encodeURIComponent(window.location.href);
    window.location.href = `/auth/oauth/google?redirect_url=${redirectUrl}`;
}

// Step 3: Handle OAuth callback (on page load)
function handleOAuthCallback() {
    const urlParams = new URLSearchParams(window.location.search);
    const oauth = urlParams.get('oauth');
    
    if (oauth === 'success') {
        const data = urlParams.get('data');
        if (data) {
            try {
                const decodedData = atob(decodeURIComponent(data));
                const oauthResponse = JSON.parse(decodedData);
                
                // Store tokens
                localStorage.setItem('access_token', oauthResponse.access_token);
                localStorage.setItem('refresh_token', oauthResponse.refresh_token);
                
                // User is now authenticated
                console.log('User logged in:', oauthResponse.user);
                onLoginSuccess(oauthResponse);
                
            } catch (e) {
                console.error('Failed to parse OAuth response:', e);
            }
        }
    } else if (oauth === 'error') {
        const message = urlParams.get('message');
        console.error('OAuth error:', decodeURIComponent(message || 'Unknown error'));
        onLoginError(message);
    }
    
    // Clean URL after processing
    if (oauth) {
        const cleanUrl = window.location.pathname;
        window.history.replaceState({}, document.title, cleanUrl);
    }
}

// Step 4: Make authenticated API calls
async function callAuthenticatedAPI(endpoint) {
    const token = localStorage.getItem('access_token');
    const response = await fetch(endpoint, {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    });
    return response.json();
}

// Initialize on page load
window.addEventListener('load', handleOAuthCallback);
```

### Complete Google OAuth Example

Here's a minimal, complete example for implementing Google OAuth login:

```html
<!DOCTYPE html>
<html>
<head>
    <title>Simple OAuth Example</title>
</head>
<body>
    <div id="loginSection">
        <h2>Login Required</h2>
        <button id="googleLogin" style="display:none;">Login with Google</button>
        <p id="status">Loading...</p>
    </div>
    
    <div id="userSection" style="display:none;">
        <h2>Welcome!</h2>
        <p>User: <span id="userName"></span></p>
        <p>Email: <span id="userEmail"></span></p>
        <button onclick="logout()">Logout</button>
        <button onclick="testProfile()">Test Profile API</button>
        <div id="apiResult"></div>
    </div>

    <script>
        // Check if user is already logged in
        function checkAuth() {
            const token = localStorage.getItem('access_token');
            if (token) {
                showUserSection();
            } else {
                showLoginSection();
            }
        }

        // Load OAuth providers and show login options
        async function showLoginSection() {
            document.getElementById('loginSection').style.display = 'block';
            document.getElementById('userSection').style.display = 'none';
            
            try {
                const response = await fetch('/auth/oauth/providers');
                const data = await response.json();
                
                if (data.success && data.data.providers) {
                    const googleProvider = data.data.providers.find(p => p.provider === 'google');
                    if (googleProvider && googleProvider.enabled) {
                        document.getElementById('googleLogin').style.display = 'block';
                        document.getElementById('status').textContent = 'Click to login with Google';
                    } else {
                        document.getElementById('status').textContent = 'Google OAuth not available';
                    }
                } else {
                    document.getElementById('status').textContent = 'OAuth providers not available';
                }
            } catch (error) {
                document.getElementById('status').textContent = 'Error loading OAuth providers';
                console.error('Error:', error);
            }
        }

        // Show user section after successful login
        function showUserSection() {
            document.getElementById('loginSection').style.display = 'none';
            document.getElementById('userSection').style.display = 'block';
            
            // Try to get user info from stored data
            const userStr = localStorage.getItem('user_data');
            if (userStr) {
                const user = JSON.parse(userStr);
                document.getElementById('userName').textContent = user.display_name || user.name;
                document.getElementById('userEmail').textContent = user.email;
            }
        }

        // Initiate Google OAuth login
        function loginWithGoogle() {
            const redirectUrl = encodeURIComponent(window.location.href);
            window.location.href = `/auth/oauth/google?redirect_url=${redirectUrl}`;
        }

        // Handle OAuth callback
        function handleOAuthCallback() {
            const urlParams = new URLSearchParams(window.location.search);
            const oauth = urlParams.get('oauth');
            
            if (oauth === 'success') {
                const data = urlParams.get('data');
                if (data) {
                    try {
                        const decodedData = atob(decodeURIComponent(data));
                        const oauthResponse = JSON.parse(decodedData);
                        
                        // Store authentication data
                        localStorage.setItem('access_token', oauthResponse.access_token);
                        localStorage.setItem('refresh_token', oauthResponse.refresh_token);
                        localStorage.setItem('user_data', JSON.stringify(oauthResponse.user));
                        
                        console.log('OAuth login successful:', oauthResponse.user);
                        showUserSection();
                        
                    } catch (e) {
                        console.error('Failed to parse OAuth response:', e);
                        alert('Login successful but failed to parse response');
                    }
                }
            } else if (oauth === 'error') {
                const message = urlParams.get('message');
                console.error('OAuth error:', decodeURIComponent(message || 'Unknown error'));
                alert('Login failed: ' + decodeURIComponent(message || 'Unknown error'));
            }
            
            // Clean URL after processing
            if (oauth) {
                const cleanUrl = window.location.pathname;
                window.history.replaceState({}, document.title, cleanUrl);
            }
        }

        // Test authenticated API call
        async function testProfile() {
            const token = localStorage.getItem('access_token');
            if (!token) {
                alert('No access token found');
                return;
            }

            try {
                const response = await fetch('/auth/profile', {
                    headers: {
                        'Authorization': `Bearer ${token}`
                    }
                });

                const result = await response.json();
                document.getElementById('apiResult').innerHTML = 
                    '<h3>Profile API Result:</h3><pre>' + 
                    JSON.stringify(result, null, 2) + '</pre>';
            } catch (error) {
                console.error('Profile API error:', error);
                alert('Failed to fetch profile');
            }
        }

        // Logout function
        function logout() {
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
            localStorage.removeItem('user_data');
            showLoginSection();
        }

        // Event listeners
        document.getElementById('googleLogin').addEventListener('click', loginWithGoogle);
        
        // Initialize on page load
        window.addEventListener('load', () => {
            handleOAuthCallback();
            checkAuth();
        });
    </script>
</body>
</html>
```

### Key Implementation Notes

1. **Mandatory redirect_url**: Always include `redirect_url` parameter when calling OAuth endpoints
2. **State Parameter**: The redirect URL is encoded in the OAuth state parameter for security
3. **Response Handling**: OAuth responses are base64-encoded JSON in the `data` URL parameter  
4. **Token Storage**: Store access/refresh tokens securely (consider using secure HTTP-only cookies in production)
5. **Error Handling**: Always handle both success and error OAuth callback scenarios
6. **URL Cleanup**: Clean the URL after processing OAuth parameters to improve UX

### Production Considerations

- Use secure storage for tokens (HTTP-only cookies recommended)
- Implement token refresh logic using the refresh_token
- Add proper error handling and user feedback
- Use HTTPS for all OAuth flows
- Consider implementing CSRF protection with state parameter validation