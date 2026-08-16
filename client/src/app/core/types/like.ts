import { UserSummary } from './user';

export type Like = {
  id: number;
  user_id?: number;
  user?: UserSummary;
  actor_iri?: string;
  actor_name?: string;
  avatar_url?: string;
  created_at: string;
};
