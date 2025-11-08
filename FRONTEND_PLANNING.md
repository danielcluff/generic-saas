# Frontend Planning - Astro + Solid.js

## Project Structure

### Directory Layout
```
frontend/
├── src/
│   ├── components/
│   │   ├── ui/
│   │   │   ├── Button.tsx
│   │   │   ├── Input.tsx
│   │   │   ├── Modal.tsx
│   │   │   └── LoadingSpinner.tsx
│   │   ├── forms/
│   │   │   ├── LoginForm.tsx
│   │   │   ├── RegisterForm.tsx
│   │   │   └── PaymentForm.tsx
│   │   ├── layout/
│   │   │   ├── Header.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   ├── Footer.tsx
│   │   │   └── Layout.astro
│   │   └── features/
│   │       ├── auth/
│   │       ├── dashboard/
│   │       ├── billing/
│   │       └── settings/
│   ├── pages/
│   │   ├── index.astro
│   │   ├── login.astro
│   │   ├── register.astro
│   │   ├── dashboard/
│   │   │   ├── index.astro
│   │   │   ├── billing.astro
│   │   │   └── settings.astro
│   │   └── api/
│   │       └── auth/
│   ├── stores/
│   │   ├── authStore.ts
│   │   ├── userStore.ts
│   │   └── subscriptionStore.ts
│   ├── utils/
│   │   ├── api.ts
│   │   ├── auth.ts
│   │   └── validation.ts
│   ├── styles/
│   │   ├── global.css
│   │   └── components.css
│   └── types/
│       ├── api.ts
│       ├── user.ts
│       └── subscription.ts
├── public/
│   ├── favicon.ico
│   └── images/
├── astro.config.mjs
├── tailwind.config.mjs
├── package.json
└── tsconfig.json
```

## Configuration Files

### astro.config.mjs
```javascript
import { defineConfig } from 'astro/config';
import solidJs from '@astrojs/solid-js';
import tailwind from '@astrojs/tailwind';

export default defineConfig({
  integrations: [
    solidJs(),
    tailwind()
  ],
  output: 'hybrid',
  server: {
    port: 3000
  }
});
```

### package.json Dependencies
```json
{
  "dependencies": {
    "astro": "^4.0.0",
    "@astrojs/solid-js": "^4.0.0",
    "@astrojs/tailwind": "^5.0.0",
    "solid-js": "^1.8.0",
    "@solidjs/router": "^0.10.0",
    "axios": "^1.6.0",
    "@stripe/stripe-js": "^2.0.0",
    "zod": "^3.22.0"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "typescript": "^5.0.0",
    "tailwindcss": "^3.4.0"
  }
}
```

## Page Structure

### Landing Page (index.astro)
```astro
---
import Layout from '../components/layout/Layout.astro';
import Hero from '../components/Hero.tsx';
import Features from '../components/Features.tsx';
import Pricing from '../components/Pricing.tsx';
---

<Layout title="Generic SaaS Platform">
  <Hero client:load />
  <Features />
  <Pricing client:load />
</Layout>
```

### Authentication Pages
```astro
<!-- login.astro -->
---
import Layout from '../components/layout/Layout.astro';
import LoginForm from '../components/forms/LoginForm.tsx';
---

<Layout title="Login">
  <div class="min-h-screen flex items-center justify-center">
    <LoginForm client:load />
  </div>
</Layout>

<!-- register.astro -->
---
import Layout from '../components/layout/Layout.astro';
import RegisterForm from '../components/forms/RegisterForm.tsx';
---

<Layout title="Register">
  <div class="min-h-screen flex items-center justify-center">
    <RegisterForm client:load />
  </div>
</Layout>
```

### Dashboard Pages
```astro
<!-- dashboard/index.astro -->
---
import DashboardLayout from '../../components/layout/DashboardLayout.astro';
import DashboardOverview from '../../components/features/dashboard/Overview.tsx';
---

<DashboardLayout title="Dashboard">
  <DashboardOverview client:load />
</DashboardLayout>
```

## Solid.js Components

### Authentication Store
```typescript
// stores/authStore.ts
import { createSignal, createEffect } from 'solid-js';
import { createStore } from 'solid-js/store';

export interface User {
  id: string;
  email: string;
  createdAt: string;
}

export interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  token: string | null;
  loading: boolean;
}

const [authState, setAuthState] = createStore<AuthState>({
  isAuthenticated: false,
  user: null,
  token: null,
  loading: true
});

export const useAuth = () => {
  const login = async (email: string, password: string) => {
    setAuthState('loading', true);
    try {
      const response = await api.post('/auth/login', { email, password });
      const { token, user } = response.data.data;

      localStorage.setItem('token', token);
      setAuthState({
        isAuthenticated: true,
        user,
        token,
        loading: false
      });

      return { success: true };
    } catch (error) {
      setAuthState('loading', false);
      return { success: false, error: error.response?.data?.error?.message };
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    setAuthState({
      isAuthenticated: false,
      user: null,
      token: null,
      loading: false
    });
  };

  const checkAuth = async () => {
    const token = localStorage.getItem('token');
    if (!token) {
      setAuthState('loading', false);
      return;
    }

    try {
      const response = await api.get('/user/profile');
      setAuthState({
        isAuthenticated: true,
        user: response.data.data,
        token,
        loading: false
      });
    } catch (error) {
      logout();
    }
  };

  return {
    authState,
    login,
    logout,
    checkAuth
  };
};
```

### Login Form Component
```tsx
// components/forms/LoginForm.tsx
import { createSignal } from 'solid-js';
import { useAuth } from '../../stores/authStore';
import Button from '../ui/Button';
import Input from '../ui/Input';

export default function LoginForm() {
  const [email, setEmail] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [error, setError] = createSignal('');
  const { login, authState } = useAuth();

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setError('');

    const result = await login(email(), password());
    if (!result.success) {
      setError(result.error || 'Login failed');
    } else {
      window.location.href = '/dashboard';
    }
  };

  return (
    <div class="max-w-md w-full space-y-8">
      <div>
        <h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">
          Sign in to your account
        </h2>
      </div>
      <form class="mt-8 space-y-6" onSubmit={handleSubmit}>
        <div class="space-y-4">
          <Input
            type="email"
            placeholder="Email address"
            value={email()}
            onInput={(e) => setEmail(e.currentTarget.value)}
            required
          />
          <Input
            type="password"
            placeholder="Password"
            value={password()}
            onInput={(e) => setPassword(e.currentTarget.value)}
            required
          />
        </div>

        {error() && (
          <div class="text-red-600 text-sm text-center">{error()}</div>
        )}

        <Button
          type="submit"
          class="w-full"
          loading={authState.loading}
          disabled={authState.loading}
        >
          Sign in
        </Button>
      </form>

      <div class="text-center">
        <a href="/register" class="text-blue-600 hover:text-blue-500">
          Don't have an account? Sign up
        </a>
      </div>
    </div>
  );
}
```

### Payment Form Component
```tsx
// components/forms/PaymentForm.tsx
import { createSignal, onMount } from 'solid-js';
import { loadStripe } from '@stripe/stripe-js';

interface PaymentFormProps {
  planId: string;
  amount: number;
  currency: string;
}

export default function PaymentForm(props: PaymentFormProps) {
  const [stripe, setStripe] = createSignal(null);
  const [elements, setElements] = createSignal(null);
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal('');

  onMount(async () => {
    const stripeInstance = await loadStripe(import.meta.env.PUBLIC_STRIPE_KEY);
    setStripe(stripeInstance);

    // Create payment intent
    const response = await api.post('/payments/stripe/create-intent', {
      amount: props.amount,
      currency: props.currency,
      planId: props.planId
    });

    const elementsInstance = stripeInstance.elements({
      clientSecret: response.data.data.clientSecret
    });
    setElements(elementsInstance);
  });

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    if (!stripe() || !elements()) return;

    setLoading(true);
    setError('');

    const { error: stripeError } = await stripe().confirmPayment({
      elements: elements(),
      confirmParams: {
        return_url: `${window.location.origin}/dashboard/billing/success`
      }
    });

    if (stripeError) {
      setError(stripeError.message);
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} class="space-y-6">
      <div id="payment-element">
        {/* Stripe Elements will mount here */}
      </div>

      {error() && (
        <div class="text-red-600 text-sm">{error()}</div>
      )}

      <Button
        type="submit"
        loading={loading()}
        disabled={loading() || !stripe()}
        class="w-full"
      >
        Pay ${props.amount / 100}
      </Button>
    </form>
  );
}
```

## UI Component Library

### Button Component
```tsx
// components/ui/Button.tsx
import { JSX, mergeProps, splitProps } from 'solid-js';

interface ButtonProps extends JSX.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
}

export default function Button(props: ButtonProps) {
  const merged = mergeProps({ variant: 'primary', size: 'md' }, props);
  const [local, others] = splitProps(merged, ['variant', 'size', 'loading', 'children', 'class']);

  const baseClass = 'inline-flex items-center justify-center font-medium rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed';

  const variantClasses = {
    primary: 'bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500',
    secondary: 'bg-gray-200 text-gray-900 hover:bg-gray-300 focus:ring-gray-500',
    danger: 'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500'
  };

  const sizeClasses = {
    sm: 'px-3 py-2 text-sm',
    md: 'px-4 py-2 text-base',
    lg: 'px-6 py-3 text-lg'
  };

  const classes = `${baseClass} ${variantClasses[local.variant]} ${sizeClasses[local.size]} ${local.class || ''}`;

  return (
    <button class={classes} disabled={local.loading} {...others}>
      {local.loading && (
        <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      )}
      {local.children}
    </button>
  );
}
```

## API Client

### HTTP Client Setup
```typescript
// utils/api.ts
import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.PUBLIC_API_URL || 'http://localhost:8080/api/v1',
  timeout: 10000
});

// Request interceptor to add auth token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default api;
```

## Routing Strategy

### Client-Side Navigation
```typescript
// utils/router.ts
import { createSignal } from 'solid-js';

export const [currentPath, setCurrentPath] = createSignal(window.location.pathname);

export const navigate = (path: string) => {
  window.history.pushState({}, '', path);
  setCurrentPath(path);
};

// Listen for browser back/forward buttons
window.addEventListener('popstate', () => {
  setCurrentPath(window.location.pathname);
});
```

## Styling with Tailwind CSS

### Global Styles
```css
/* styles/global.css */
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  html {
    font-family: 'Inter', system-ui, sans-serif;
  }
}

@layer components {
  .btn {
    @apply inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2;
  }

  .input {
    @apply block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500;
  }
}
```

## Performance Optimizations

### Lazy Loading Components
```tsx
import { lazy } from 'solid-js';

const DashboardAnalytics = lazy(() => import('./DashboardAnalytics'));

// Use with Suspense
<Suspense fallback={<LoadingSpinner />}>
  <DashboardAnalytics />
</Suspense>
```

### Bundle Splitting
```javascript
// astro.config.mjs
export default defineConfig({
  vite: {
    build: {
      rollupOptions: {
        output: {
          manualChunks: {
            vendor: ['solid-js'],
            stripe: ['@stripe/stripe-js']
          }
        }
      }
    }
  }
});
```