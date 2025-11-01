/**
 * Statistics domain models
 */

export type Statistics = {
  user_id: number;
  bucket_format: string;
  buckets: Record<string, StatisticBuckets>;
};

export type StatisticBuckets = {
  workout_type: string;
  local_workout_type: string;
  buckets: Record<string, StatisticData>;
};

export type StatisticData = {
  bucket: string;
  workouts: number;
  duration_seconds: number;
  distance: number;
  average_speed: number;
  average_speed_no_pause: number;
  max_speed: number;
  duration: number;
};

export type StatisticsParams = {
  since?: string;
  per?: string;
};

export type GeoJsonFeature = {
  type: string;
  geometry: {
    type: string;
    coordinates: number[];
  };
  properties?: Record<string, unknown>;
};

export type GeoJsonFeatureCollection = {
  type: string;
  features: GeoJsonFeature[];
};

export type WorkoutPopupData = {
  id: number;
  name: string;
  date: string;
  type: string;
  custom_type?: string;
  locked: boolean;

  // Type-specific fields
  total_distance?: number;
  total_duration?: number;
  total_repetitions?: number;
  repetition_frequency_per_min?: number;
  total_weight?: number;
  average_speed?: number;
};
