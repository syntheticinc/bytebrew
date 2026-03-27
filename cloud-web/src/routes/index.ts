import {
  createRootRoute,
  createRoute,
} from '@tanstack/react-router';
import { RootLayout } from '../components/Layout';
import { LandingPage } from '../pages/Landing';
import { LoginPage } from '../pages/Login';
import { RegisterPage } from '../pages/Register';
import { DashboardPage } from '../pages/Dashboard';
import { BillingPage } from '../pages/Billing';
import { BillingSuccessPage } from '../pages/BillingSuccess';
import { BillingCancelPage } from '../pages/BillingCancel';
import { SettingsPage } from '../pages/Settings';
import { TeamPage } from '../pages/Team';
import { ForgotPasswordPage } from '../pages/ForgotPassword';
import { ResetPasswordPage } from '../pages/ResetPassword';
import { TermsPage } from '../pages/Terms';
import { PricingPage } from '../pages/Pricing';
import { DownloadPage } from '../pages/Download';
import { PrivacyPage } from '../pages/Privacy';
import { ExamplesPage } from '../pages/Examples';
import { ExampleDemoPage } from '../pages/ExampleDemo';
import { VerifyEmailPage } from '../pages/VerifyEmail';

const rootRoute = createRootRoute({
  component: RootLayout,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: LandingPage,
});

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginPage,
});

const registerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/register',
  component: RegisterPage,
});

const dashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/dashboard',
  component: DashboardPage,
});

const billingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/billing',
  component: BillingPage,
});

const billingSuccessRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/billing/success',
  component: BillingSuccessPage,
});

const billingCancelRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/billing/cancel',
  component: BillingCancelPage,
});

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  component: SettingsPage,
});

const teamRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/team',
  component: TeamPage,
});

const forgotPasswordRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/forgot-password',
  component: ForgotPasswordPage,
});

const resetPasswordRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/reset-password',
  component: ResetPasswordPage,
});

const termsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/terms',
  component: TermsPage,
});

const pricingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/pricing',
  component: PricingPage,
});

const downloadRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/download',
  component: DownloadPage,
});

const privacyRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/privacy',
  component: PrivacyPage,
});

/* /docs is served by Starlight (docs-site).
   In dev: Vite proxy → localhost:4321/docs
   In prod: Caddy serves /var/www/bytebrew-docs */

const examplesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/examples',
  component: ExamplesPage,
});

const exampleDemoRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/examples/$slug',
  component: ExampleDemoPage,
});

const verifyEmailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/verify-email',
  component: VerifyEmailPage,
});

export const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  registerRoute,
  dashboardRoute,
  billingRoute,
  billingSuccessRoute,
  billingCancelRoute,
  teamRoute,
  settingsRoute,
  forgotPasswordRoute,
  resetPasswordRoute,
  termsRoute,
  pricingRoute,
  downloadRoute,
  privacyRoute,

  examplesRoute,
  exampleDemoRoute,
  verifyEmailRoute,
]);
