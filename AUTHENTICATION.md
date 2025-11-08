# Authentication System Planning

## Overview
Comprehensive authentication system supporting email/password and Google OAuth, with JWT-based session management and secure password handling.

## Authentication Methods

### 1. Email/Password Authentication

#### Password Requirements
- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number
- At least one special character
- Maximum 128 characters

#### Password Security
```go
// Password hashing with bcrypt
func HashPassword(password string) (string, error) {
    cost := 12 // Adjust based on security requirements vs performance
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
    return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

### 2. Google OAuth Integration

#### OAuth Flow
1. Frontend redirects to Google OAuth consent screen
2. User authorizes application
3. Google redirects back with authorization code
4. Backend exchanges code for access token
5. Backend retrieves user profile from Google
6. Backend creates/updates user record
7. Backend returns JWT tokens

#### Backend Implementation
```go
type GoogleConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
}

type GoogleUserInfo struct {
    ID      string `json:"id"`
    Email   string `json:"email"`
    Name    string `json:"name"`
    Picture string `json:"picture"`
}

func (s *AuthService) HandleGoogleCallback(code string) (*AuthResponse, error) {
    // Exchange code for token
    token, err := s.googleConfig.Exchange(context.Background(), code)
    if err != nil {
        return nil, err
    }

    // Get user info from Google
    userInfo, err := s.getGoogleUserInfo(token.AccessToken)
    if err != nil {
        return nil, err
    }

    // Create or update user
    user, err := s.findOrCreateGoogleUser(userInfo)
    if err != nil {
        return nil, err
    }

    // Generate JWT tokens
    return s.generateTokens(user)
}
```

#### Frontend Implementation
```typescript
// Google OAuth in Solid.js
import { createSignal } from 'solid-js';

export function GoogleAuthButton() {
  const [loading, setLoading] = createSignal(false);

  const handleGoogleAuth = () => {
    setLoading(true);

    const googleAuthURL = new URL('https://accounts.google.com/o/oauth2/v2/auth');
    googleAuthURL.searchParams.append('client_id', import.meta.env.PUBLIC_GOOGLE_CLIENT_ID);
    googleAuthURL.searchParams.append('redirect_uri', `${window.location.origin}/auth/google/callback`);
    googleAuthURL.searchParams.append('response_type', 'code');
    googleAuthURL.searchParams.append('scope', 'email profile');
    googleAuthURL.searchParams.append('access_type', 'offline');

    window.location.href = googleAuthURL.toString();
  };

  return (
    <button
      onClick={handleGoogleAuth}
      disabled={loading()}
      class="w-full flex justify-center items-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
    >
      <svg class="w-5 h-5 mr-2" viewBox="0 0 24 24">
        {/* Google icon SVG */}
      </svg>
      Continue with Google
    </button>
  );
}
```

## JWT Token Management

### Token Structure
```go
type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}

type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
}
```

### Token Generation
```go
func (s *AuthService) GenerateTokens(user *User) (*TokenPair, error) {
    // Access token (15 minutes)
    accessClaims := &Claims{
        UserID: user.ID,
        Email:  user.Email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "generic-saas",
        },
    }

    accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    accessTokenString, err := accessToken.SignedString([]byte(s.jwtSecret))
    if err != nil {
        return nil, err
    }

    // Refresh token (7 days)
    refreshClaims := &Claims{
        UserID: user.ID,
        Email:  user.Email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "generic-saas",
        },
    }

    refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    refreshTokenString, err := refreshToken.SignedString([]byte(s.jwtSecret))
    if err != nil {
        return nil, err
    }

    return &TokenPair{
        AccessToken:  accessTokenString,
        RefreshToken: refreshTokenString,
    }, nil
}
```

### Token Validation Middleware
```go
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(s.jwtSecret), nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }

    return nil, fmt.Errorf("invalid token")
}

func AuthMiddleware(authService *AuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := authService.ValidateToken(tokenString)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        c.Set("user_id", claims.UserID)
        c.Set("user_email", claims.Email)
        c.Next()
    }
}
```

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- NULL for OAuth-only users
    google_id VARCHAR(255) UNIQUE, -- NULL for email/password users
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_google_id ON users(google_id);
```

### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
```

### Email Verification Table
```sql
CREATE TABLE email_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_email_verifications_token ON email_verifications(token);
```

## Frontend Authentication Flow

### Auth Store Implementation
```typescript
// stores/authStore.ts
import { createStore } from 'solid-js/store';
import { createSignal, createEffect } from 'solid-js';
import api from '../utils/api';

export interface User {
  id: string;
  email: string;
  emailVerified: boolean;
  createdAt: string;
}

interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  loading: boolean;
}

const [authState, setAuthState] = createStore<AuthState>({
  isAuthenticated: false,
  user: null,
  loading: true
});

// Check authentication on app load
createEffect(() => {
  checkAuth();
});

const checkAuth = async () => {
  const token = localStorage.getItem('access_token');
  if (!token) {
    setAuthState({ loading: false });
    return;
  }

  try {
    const response = await api.get('/user/profile');
    setAuthState({
      isAuthenticated: true,
      user: response.data.data,
      loading: false
    });
  } catch (error) {
    logout();
  }
};

const login = async (email: string, password: string) => {
  try {
    const response = await api.post('/auth/login', { email, password });
    const { access_token, refresh_token, user } = response.data.data;

    localStorage.setItem('access_token', access_token);
    localStorage.setItem('refresh_token', refresh_token);

    setAuthState({
      isAuthenticated: true,
      user,
      loading: false
    });

    return { success: true };
  } catch (error) {
    return {
      success: false,
      error: error.response?.data?.error?.message || 'Login failed'
    };
  }
};

const logout = async () => {
  const refreshToken = localStorage.getItem('refresh_token');

  if (refreshToken) {
    try {
      await api.post('/auth/logout', { refresh_token: refreshToken });
    } catch (error) {
      console.error('Logout error:', error);
    }
  }

  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');

  setAuthState({
    isAuthenticated: false,
    user: null,
    loading: false
  });
};

export { authState, login, logout, checkAuth };
```

### Token Refresh Implementation
```typescript
// utils/api.ts
import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.PUBLIC_API_URL,
});

// Request interceptor
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor for token refresh
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      const refreshToken = localStorage.getItem('refresh_token');
      if (!refreshToken) {
        // Redirect to login
        window.location.href = '/login';
        return Promise.reject(error);
      }

      try {
        const response = await axios.post(`${import.meta.env.PUBLIC_API_URL}/auth/refresh`, {
          refresh_token: refreshToken
        });

        const { access_token } = response.data.data;
        localStorage.setItem('access_token', access_token);

        // Retry original request
        originalRequest.headers.Authorization = `Bearer ${access_token}`;
        return api(originalRequest);
      } catch (refreshError) {
        // Refresh failed, redirect to login
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        window.location.href = '/login';
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);

export default api;
```

## Security Considerations

### Password Security
- Use bcrypt with cost factor 12+
- Implement password strength validation
- Rate limit login attempts
- Account lockout after failed attempts

### JWT Security
- Short-lived access tokens (15 minutes)
- Secure refresh token storage
- Token rotation on refresh
- Proper token validation

### Session Management
- Secure HTTP-only cookies for refresh tokens
- CSRF protection
- Secure flag on cookies
- SameSite cookie attribute

### OWASP Compliance
- Input validation and sanitization
- SQL injection prevention
- XSS protection
- CSRF protection
- Rate limiting
- Secure headers

## Email Verification

### Backend Implementation
```go
func (s *AuthService) SendVerificationEmail(userID string) error {
    token := generateSecureToken(32)
    expiresAt := time.Now().Add(24 * time.Hour)

    // Store verification token
    err := s.db.CreateEmailVerification(userID, token, expiresAt)
    if err != nil {
        return err
    }

    // Send email
    verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.frontendURL, token)
    return s.emailService.SendVerificationEmail(user.Email, verificationURL)
}

func (s *AuthService) VerifyEmail(token string) error {
    verification, err := s.db.GetEmailVerification(token)
    if err != nil {
        return err
    }

    if time.Now().After(verification.ExpiresAt) {
        return errors.New("verification token expired")
    }

    // Mark email as verified
    err = s.db.MarkEmailVerified(verification.UserID)
    if err != nil {
        return err
    }

    // Mark verification as used
    return s.db.MarkVerificationUsed(verification.ID)
}
```

## Testing Strategy

### Unit Tests
- Password hashing/verification
- JWT token generation/validation
- OAuth flow components
- Email verification logic

### Integration Tests
- Complete authentication flows
- API endpoint testing
- Database interactions
- External service integrations

### Security Tests
- SQL injection attempts
- XSS prevention
- CSRF protection
- Rate limiting effectiveness