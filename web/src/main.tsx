import * as RadixTooltip from '@radix-ui/react-tooltip';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { ApiError } from './api/http';
import { AuthProvider } from './auth/AuthContext';
import { ToastProvider } from './components/Toast';
import './index.css';
import { AppRoutes } from './routes';

// 4xx не повторяем: это ответ по существу, а не сбой сети.
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 15_000,
      refetchOnWindowFocus: false,
      retry: (count, err) => !(err instanceof ApiError && err.status >= 400 && err.status < 500) && count < 2,
    },
    mutations: { retry: false },
  },
});

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <ToastProvider>
          <RadixTooltip.Provider>
            <AuthProvider>
              <AppRoutes />
            </AuthProvider>
          </RadixTooltip.Provider>
        </ToastProvider>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
);
