import { useState, type FormEvent } from 'react';
import { AuthGuard } from '../components/AuthGuard';
import { useAuth } from '../lib/auth';
import { changePassword, deleteAccount } from '../api/auth';
import { ApiError } from '../api/client';
import { useNavigate, Link } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { getUsage } from '../api/license';

export function SettingsPage() {
  return (
    <AuthGuard>
      <SettingsContent />
    </AuthGuard>
  );
}

function SettingsContent() {
  const { email, logout } = useAuth();
  const navigate = useNavigate();

  return (
    <div className="max-w-3xl mx-auto px-4 py-10">
      <h1 className="text-2xl font-bold text-text-primary">Settings</h1>

      <div className="mt-8 space-y-6">
        {/* Profile */}
        <div className="rounded-[2px] border border-border bg-surface-alt p-5">
          <h2 className="text-base font-semibold text-text-primary">Profile</h2>
          <div className="mt-4">
            <label className="block text-sm font-medium text-text-secondary">Email</label>
            <p className="mt-1 text-sm text-text-secondary">{email}</p>
          </div>
        </div>

        {/* Change Password */}
        <ChangePasswordSection />

        {/* Danger Zone */}
        <DangerZoneSection onDeleted={() => { logout(); navigate({ to: '/' }); }} />
      </div>
    </div>
  );
}

function ChangePasswordSection() {
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess(false);

    if (newPassword.length < 8) {
      setError('New password must be at least 8 characters');
      return;
    }

    if (newPassword !== confirmPassword) {
      setError('New passwords do not match');
      return;
    }

    setLoading(true);

    try {
      await changePassword(currentPassword, newPassword);
      setSuccess(true);
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError('An unexpected error occurred');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="rounded-[2px] border border-border bg-surface-alt p-5">
      <h2 className="text-base font-semibold text-text-primary">Change Password</h2>

      <form onSubmit={handleSubmit} className="mt-4 space-y-4">
        {error && (
          <div className="rounded-[2px] bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
            {error}
          </div>
        )}

        {success && (
          <div className="rounded-[2px] bg-emerald-500/10 border border-emerald-500/20 p-3 text-sm text-emerald-400">
            Password changed successfully
          </div>
        )}

        <div>
          <label htmlFor="currentPassword" className="block text-sm font-medium text-text-secondary mb-1">
            Current Password
          </label>
          <input
            id="currentPassword"
            type="password"
            required
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            className="w-full rounded-[2px] border border-border-hover bg-surface px-3 py-2 text-sm text-text-primary placeholder-text-tertiary focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent"
            placeholder="Enter your current password"
          />
        </div>

        <div>
          <label htmlFor="newPassword" className="block text-sm font-medium text-text-secondary mb-1">
            New Password
          </label>
          <input
            id="newPassword"
            type="password"
            required
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            className="w-full rounded-[2px] border border-border-hover bg-surface px-3 py-2 text-sm text-text-primary placeholder-text-tertiary focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent"
            placeholder="At least 8 characters"
          />
        </div>

        <div>
          <label htmlFor="confirmNewPassword" className="block text-sm font-medium text-text-secondary mb-1">
            Confirm New Password
          </label>
          <input
            id="confirmNewPassword"
            type="password"
            required
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            className="w-full rounded-[2px] border border-border-hover bg-surface px-3 py-2 text-sm text-text-primary placeholder-text-tertiary focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent"
            placeholder="Repeat your new password"
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="rounded-[2px] bg-brand-accent px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? 'Changing...' : 'Change Password'}
        </button>
      </form>
    </div>
  );
}

function DangerZoneSection({ onDeleted }: { onDeleted: () => void }) {
  const usageQuery = useQuery({
    queryKey: ['usage'],
    queryFn: getUsage,
  });

  const hasActiveSubscription = !!usageQuery.data;

  const [showConfirm, setShowConfirm] = useState(false);
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleDelete = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await deleteAccount(password);
      onDeleted();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError('An unexpected error occurred');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = () => {
    setShowConfirm(false);
    setPassword('');
    setError('');
  };

  return (
    <div className="rounded-[2px] border border-red-500/20 bg-red-500/5 p-5">
      <h2 className="text-base font-semibold text-text-primary">Danger Zone</h2>
      <p className="mt-2 text-sm text-text-secondary">
        Permanently delete your account and all associated data. This action cannot be undone.
      </p>

      {hasActiveSubscription && (
        <div className="mt-3 rounded-[2px] bg-amber-500/10 border border-amber-500/20 p-3 text-sm text-amber-400">
          Your active subscription will be cancelled immediately. No refund will be issued
          for the remaining billing period. See our{' '}
          <Link to="/terms" className="underline hover:text-amber-300">
            Terms of Service
          </Link>{' '}
          for details.
        </div>
      )}

      {!showConfirm ? (
        <button
          type="button"
          onClick={() => setShowConfirm(true)}
          className="mt-4 rounded-[2px] bg-red-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-red-500 transition-colors"
        >
          Delete Account
        </button>
      ) : (
        <form onSubmit={handleDelete} className="mt-4 space-y-4">
          {error && (
            <div className="rounded-[2px] bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
              {error}
            </div>
          )}

          <div>
            <label htmlFor="deletePassword" className="block text-sm font-medium text-text-secondary mb-1">
              Enter your password to confirm
            </label>
            <input
              id="deletePassword"
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-[2px] border border-border-hover bg-surface px-3 py-2 text-sm text-text-primary placeholder-text-tertiary focus:border-brand-accent focus:outline-none focus:ring-1 focus:ring-brand-accent"
              placeholder="Enter your password"
            />
          </div>

          <div className="flex gap-3">
            <button
              type="button"
              onClick={handleCancel}
              className="rounded-[2px] border border-border px-4 py-2.5 text-sm font-medium text-text-secondary hover:bg-surface-alt transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="rounded-[2px] bg-red-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-red-500 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? 'Deleting...' : 'Delete My Account'}
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
