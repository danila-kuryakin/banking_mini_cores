import { Navigate, Route, Routes } from 'react-router-dom';
import { homeFor, useAuth } from './auth/AuthContext';
import { GuestOnly, RoleGuard } from './auth/RoleGuard';
import { FullScreenLoader } from './components/Spinner';
import { AccountPage } from './features/accounts/AccountPage';
import { AccountsPage } from './features/accounts/AccountsPage';
import { AntifraudPage } from './features/antifraud/AntifraudPage';
import { CashPage } from './features/cash/CashPage';
import { LoginPage } from './features/auth/LoginPage';
import { RegisterPage } from './features/auth/RegisterPage';
import { CustomerPage } from './features/customers/CustomerPage';
import { CustomersPage } from './features/customers/CustomersPage';
import { DashboardPage } from './features/dashboard/DashboardPage';
import { ApplicationPage } from './features/kyc-queue/ApplicationPage';
import { QueuePage } from './features/kyc-queue/QueuePage';
import { NotificationsPage } from './features/notifications/NotificationsPage';
import { OfficersPage } from './features/officers/OfficersPage';
import { OnboardingPage } from './features/onboarding/OnboardingPage';
import { ProfilePage } from './features/profile/ProfilePage';
import { SanctionsPage } from './features/sanctions/SanctionsPage';
import { StaffAccountPage } from './features/staff-accounts/StaffAccountPage';
import { TransferPage } from './features/transfers/TransferPage';
import { AuthLayout } from './layouts/AuthLayout';
import { ClientLayout } from './layouts/ClientLayout';
import { StaffLayout } from './layouts/StaffLayout';

function Home() {
  const { user, ready } = useAuth();
  if (!ready) return <FullScreenLoader />;
  return <Navigate to={user ? homeFor(user.role) : '/login'} replace />;
}

export function AppRoutes() {
  return (
    <Routes>
      <Route
        element={
          <GuestOnly>
            <AuthLayout />
          </GuestOnly>
        }
      >
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
      </Route>

      <Route
        path="/app"
        element={
          <RoleGuard roles={['client']}>
            <ClientLayout />
          </RoleGuard>
        }
      >
        <Route index element={<DashboardPage />} />
        <Route path="onboarding" element={<OnboardingPage />} />
        <Route path="accounts" element={<AccountsPage />} />
        <Route path="accounts/:accountId" element={<AccountPage />} />
        <Route path="transfer" element={<TransferPage />} />
        <Route path="profile" element={<ProfilePage />} />
        <Route path="notifications" element={<NotificationsPage />} />
        <Route path="*" element={<Navigate to="/app" replace />} />
      </Route>

      <Route
        path="/staff"
        element={
          <RoleGuard roles={['officer', 'admin']}>
            <StaffLayout />
          </RoleGuard>
        }
      >
        <Route index element={<Navigate to="kyc" replace />} />
        <Route path="kyc" element={<QueuePage />} />
        <Route path="kyc/:applicationId" element={<ApplicationPage />} />
        <Route path="customers" element={<CustomersPage />} />
        <Route path="customers/:userId" element={<CustomerPage />} />
        <Route path="accounts/:accountId" element={<StaffAccountPage />} />
        <Route path="cash" element={<CashPage />} />
        <Route path="sanctions" element={<SanctionsPage />} />
        <Route path="antifraud" element={<AntifraudPage />} />
        <Route
          path="officers"
          element={
            <RoleGuard roles={['admin']}>
              <OfficersPage />
            </RoleGuard>
          }
        />
        <Route path="*" element={<Navigate to="kyc" replace />} />
      </Route>

      <Route path="*" element={<Home />} />
    </Routes>
  );
}
