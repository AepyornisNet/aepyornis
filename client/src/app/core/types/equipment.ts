/**
 * Equipment domain models
 */

export type Equipment = {
  id: number;
  name: string;
  description?: string;
  notes?: string;
  active: boolean;
  default_for?: string[];
  user_id: number;
  created_at: string;
  updated_at: string;
};
