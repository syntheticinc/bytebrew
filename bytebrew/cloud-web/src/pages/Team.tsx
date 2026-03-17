import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { AuthGuard } from '../components/AuthGuard';
import { getUsage } from '../api/license';
import {
  getTeamMembers,
  createTeam,
  inviteMember,
  removeMember,
  type TeamMember,
  type TeamInvite,
} from '../api/teams';
import { ApiError } from '../api/client';
import { useAuth } from '../lib/auth';
import { useState } from 'react';
import { Link } from '@tanstack/react-router';

export function TeamPage() {
  return (
    <AuthGuard>
      <TeamContent />
    </AuthGuard>
  );
}

function TeamContent() {
  const usageQuery = useQuery({
    queryKey: ['usage'],
    queryFn: getUsage,
  });

  if (usageQuery.isLoading) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-10">
        <h1 className="text-2xl font-bold text-white">Team</h1>
        <div className="mt-8 text-gray-400">Loading...</div>
      </div>
    );
  }

  if (usageQuery.error || usageQuery.data?.tier !== 'teams') {
    return <UpgradePrompt />;
  }

  return <TeamManager />;
}

function UpgradePrompt() {
  return (
    <div className="max-w-4xl mx-auto px-4 py-10">
      <h1 className="text-2xl font-bold text-white">Team</h1>
      <div className="mt-8 rounded-xl border border-gray-800 bg-gray-900/50 p-6 text-center">
        <h2 className="text-lg font-semibold text-white">
          Upgrade to Teams
        </h2>
        <p className="mt-2 text-sm text-gray-400">
          Team management is available on the Teams plan. Upgrade to invite
          members and manage seats.
        </p>
        <Link
          to="/billing"
          className="mt-4 inline-block rounded-lg bg-indigo-600 px-6 py-2 text-sm font-medium text-white hover:bg-indigo-500 transition-colors"
        >
          View Plans
        </Link>
      </div>
    </div>
  );
}

function TeamManager() {
  const teamQuery = useQuery({
    queryKey: ['team-members'],
    queryFn: getTeamMembers,
    retry: (failureCount, error) => {
      if (error instanceof ApiError && error.status === 404) {
        return false;
      }
      return failureCount < 3;
    },
  });

  if (teamQuery.isLoading) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-10">
        <h1 className="text-2xl font-bold text-white">Team</h1>
        <div className="mt-8 text-gray-400">Loading team...</div>
      </div>
    );
  }

  const isNoTeam =
    teamQuery.error instanceof ApiError && teamQuery.error.status === 404;

  if (isNoTeam) {
    return <CreateTeamView />;
  }

  if (teamQuery.error) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-10">
        <h1 className="text-2xl font-bold text-white">Team</h1>
        <div className="mt-4 rounded-lg bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
          Failed to load team data
        </div>
      </div>
    );
  }

  if (!teamQuery.data) {
    return null;
  }

  return <TeamView team={teamQuery.data} />;
}

function CreateTeamView() {
  const queryClient = useQueryClient();
  const [name, setName] = useState('');
  const [error, setError] = useState('');

  const mutation = useMutation({
    mutationFn: (teamName: string) => createTeam(teamName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['team-members'] });
    },
    onError: (err) => {
      setError(
        err instanceof ApiError ? err.message : 'Failed to create team',
      );
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) {
      return;
    }
    setError('');
    mutation.mutate(trimmed);
  };

  return (
    <div className="max-w-4xl mx-auto px-4 py-10">
      <h1 className="text-2xl font-bold text-white">Team</h1>

      <div className="mt-8 rounded-xl border border-gray-800 bg-gray-900/50 p-6">
        <h2 className="text-base font-semibold text-white">
          Create Your Team
        </h2>
        <p className="mt-2 text-sm text-gray-400">
          Set up your team to invite members and manage seats.
        </p>

        {error && (
          <div className="mt-4 rounded-lg bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="mt-4 flex gap-3">
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Team name"
            className="flex-1 rounded-lg border border-gray-700 bg-gray-800 px-4 py-2 text-sm text-white placeholder-gray-500 focus:border-indigo-500 focus:outline-none"
          />
          <button
            type="submit"
            disabled={mutation.isPending || !name.trim()}
            className="rounded-lg bg-indigo-600 px-6 py-2 text-sm font-medium text-white hover:bg-indigo-500 transition-colors disabled:opacity-50"
          >
            {mutation.isPending ? 'Creating...' : 'Create Team'}
          </button>
        </form>
      </div>
    </div>
  );
}

function TeamView({
  team,
}: {
  team: { team_id: string; team_name: string; max_seats: number; members: TeamMember[]; invites: TeamInvite[] };
}) {
  const usedSeats = team.members.length;
  const pendingInvites = team.invites.filter((i) => i.status === 'pending');

  return (
    <div className="max-w-4xl mx-auto px-4 py-10">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-white">{team.team_name}</h1>
        <span className="text-sm text-gray-400">
          {usedSeats} active {usedSeats === 1 ? 'seat' : 'seats'} · $30/seat/month
        </span>
      </div>

      {/* Members */}
      <div className="mt-8 rounded-xl border border-gray-800 bg-gray-900/50 p-6">
        <h2 className="text-base font-semibold text-white">Members</h2>
        <MembersTable members={team.members} />
      </div>

      {/* Pending Invites */}
      {pendingInvites.length > 0 && (
        <div className="mt-6 rounded-xl border border-gray-800 bg-gray-900/50 p-6">
          <h2 className="text-base font-semibold text-white">
            Pending Invites
          </h2>
          <PendingInvitesList invites={pendingInvites} />
        </div>
      )}

      {/* Invite Form */}
      <div className="mt-6 rounded-xl border border-gray-800 bg-gray-900/50 p-6">
        <h2 className="text-base font-semibold text-white">Invite Member</h2>
        <InviteForm />
      </div>
    </div>
  );
}

function MembersTable({ members }: { members: TeamMember[] }) {
  const { email: currentUserEmail } = useAuth();
  const queryClient = useQueryClient();
  const [error, setError] = useState('');

  const currentMember = members.find((m) => m.email === currentUserEmail);
  const isAdmin = currentMember?.role === 'admin';

  const mutation = useMutation({
    mutationFn: (userId: string) => removeMember(userId),
    onSuccess: () => {
      setError('');
      queryClient.invalidateQueries({ queryKey: ['team-members'] });
    },
    onError: (err) => {
      setError(
        err instanceof ApiError ? err.message : 'Failed to remove member',
      );
    },
  });

  return (
    <div className="mt-4">
      {error && (
        <div className="mb-4 rounded-lg bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
          {error}
        </div>
      )}

      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-gray-800">
              <th className="pb-3 font-medium text-gray-400">Email</th>
              <th className="pb-3 font-medium text-gray-400">Role</th>
              <th className="pb-3 font-medium text-gray-400">Joined</th>
              {isAdmin && (
                <th className="pb-3 font-medium text-gray-400 text-right">
                  Actions
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {members.map((member) => {
              const isSelf = member.email === currentUserEmail;
              const canRemove =
                isAdmin && !isSelf && member.role !== 'admin';

              return (
                <tr
                  key={member.id}
                  className="border-b border-gray-800/50 last:border-0"
                >
                  <td className="py-3 text-gray-200">
                    {member.email}
                    {isSelf && (
                      <span className="ml-2 text-xs text-gray-500">
                        (you)
                      </span>
                    )}
                  </td>
                  <td className="py-3">
                    <span
                      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                        member.role === 'admin'
                          ? 'bg-purple-600/20 text-purple-400'
                          : 'bg-gray-700/50 text-gray-300'
                      }`}
                    >
                      {member.role}
                    </span>
                  </td>
                  <td className="py-3 text-gray-400">
                    {new Date(member.joined_at).toLocaleDateString()}
                  </td>
                  {isAdmin && (
                    <td className="py-3 text-right">
                      {canRemove && (
                        <button
                          onClick={() => mutation.mutate(member.user_id)}
                          disabled={mutation.isPending}
                          className="text-sm text-red-400 hover:text-red-300 transition-colors disabled:opacity-50"
                        >
                          Remove
                        </button>
                      )}
                    </td>
                  )}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function PendingInvitesList({ invites }: { invites: TeamInvite[] }) {
  return (
    <div className="mt-4 space-y-2">
      {invites.map((invite) => (
        <div
          key={invite.id}
          className="flex items-center justify-between rounded-lg bg-gray-800/50 border border-gray-700 p-3"
        >
          <div>
            <p className="text-sm text-gray-200">{invite.email}</p>
            <p className="text-xs text-gray-500">
              Expires{' '}
              {new Date(invite.expires_at).toLocaleDateString()}
            </p>
          </div>
          <span className="inline-flex items-center rounded-full bg-yellow-500/10 px-2 py-0.5 text-xs font-medium text-yellow-400">
            pending
          </span>
        </div>
      ))}
    </div>
  );
}

function InviteForm() {
  const queryClient = useQueryClient();
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const mutation = useMutation({
    mutationFn: (inviteEmail: string) => inviteMember(inviteEmail),
    onSuccess: () => {
      setError('');
      setSuccess(`Invitation sent to ${email}`);
      setEmail('');
      queryClient.invalidateQueries({ queryKey: ['team-members'] });
    },
    onError: (err) => {
      setSuccess('');
      setError(
        err instanceof ApiError ? err.message : 'Failed to send invite',
      );
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = email.trim();
    if (!trimmed) {
      return;
    }
    setError('');
    setSuccess('');
    mutation.mutate(trimmed);
  };

  return (
    <div className="mt-4">
      {error && (
        <div className="mb-3 rounded-lg bg-red-500/10 border border-red-500/20 p-3 text-sm text-red-400">
          {error}
        </div>
      )}

      {success && (
        <div className="mb-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 p-3 text-sm text-emerald-400">
          {success}
        </div>
      )}

      <form onSubmit={handleSubmit} className="flex gap-3">
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="colleague@example.com"
          className="flex-1 rounded-lg border border-gray-700 bg-gray-800 px-4 py-2 text-sm text-white placeholder-gray-500 focus:border-indigo-500 focus:outline-none"
        />
        <button
          type="submit"
          disabled={mutation.isPending || !email.trim()}
          className="rounded-lg bg-indigo-600 px-6 py-2 text-sm font-medium text-white hover:bg-indigo-500 transition-colors disabled:opacity-50"
        >
          {mutation.isPending ? 'Sending...' : 'Send Invite'}
        </button>
      </form>
      <p className="mt-2 text-xs text-gray-500">
        Adding a member will add $30/month to your bill (prorated)
      </p>
    </div>
  );
}
