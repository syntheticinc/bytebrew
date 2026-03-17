import { api } from './client';

export interface TeamMember {
  id: string;
  user_id: string;
  email: string;
  role: 'admin' | 'member';
  joined_at: string;
}

export interface TeamInvite {
  id: string;
  email: string;
  status: 'pending' | 'accepted' | 'revoked';
  created_at: string;
  expires_at: string;
}

export interface TeamInfo {
  team_id: string;
  team_name: string;
  max_seats: number;
  members: TeamMember[];
  invites: TeamInvite[];
}

export interface CreateTeamResult {
  id: string;
  name: string;
  max_seats: number;
}

export async function getTeamMembers(): Promise<TeamInfo> {
  return api.request<TeamInfo>('GET', '/api/v1/teams/members');
}

export async function createTeam(name: string): Promise<CreateTeamResult> {
  return api.request<CreateTeamResult>('POST', '/api/v1/teams', { name });
}

export async function inviteMember(
  email: string,
): Promise<{ invite_id: string; token: string }> {
  return api.request<{ invite_id: string; token: string }>(
    'POST',
    '/api/v1/teams/invite',
    { email },
  );
}

export async function removeMember(userId: string): Promise<void> {
  await api.request<void>('DELETE', `/api/v1/teams/members/${userId}`);
}
