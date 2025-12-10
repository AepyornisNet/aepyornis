/**
 * Measurement domain models
 */

export type Measurement = {
  id: number;
  date: string;
  weight?: number;
  height?: number;
  steps?: number;
  ftp?: number;
  resting_heart_rate?: number;
  max_heart_rate?: number;
  user_id: number;
  created_at: string;
  updated_at: string;
};
