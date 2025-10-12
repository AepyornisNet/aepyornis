/**
 * Measurement domain models
 */

export interface Measurement {
  id: number;
  date: string;
  weight?: number;
  height?: number;
  steps?: number;
  user_id: number;
  created_at: string;
  updated_at: string;
}
