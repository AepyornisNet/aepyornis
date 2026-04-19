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
  usage?: EquipmentUsageStats;
  profile_id: number;
  created_at: string;
  updated_at: string;
};

export type EquipmentUsageStats = {
  workouts: number;
  distance: number;
  duration_seconds: number;
  repetitions: number;
  last_used_at?: string;
};
