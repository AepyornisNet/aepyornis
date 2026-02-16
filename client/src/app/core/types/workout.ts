/**
 * Workout domain models
 */

import { PaginationParams } from './api-response';
import { UserProfile } from './user';

export type Workout = {
  id: number;
  date: string;
  dirty: boolean;
  name: string;
  notes: string;
  type: string;
  sub_type?: string;
  custom_type?: string;
  user_id: number;
  user?: UserProfile;
  public_uuid?: string;
  locked: boolean;
  created_at: string;
  updated_at: string;
  has_file: boolean;
  has_tracks: boolean;
  activity_pub_published: boolean;

  // Optional map data
  address_string?: string;
  total_distance?: number;
  total_duration?: number;
  total_weight?: number;
  total_repetitions?: number;
  total_up?: number;
  total_down?: number;
  average_speed?: number;
  average_speed_no_pause?: number;
  max_speed?: number;
  min_elevation?: number;
  max_elevation?: number;
  pause_duration?: number;
  average_cadence?: number;
  max_cadence?: number;
  average_heart_rate?: number;
  max_heart_rate?: number;
  average_power?: number;
  max_power?: number;
};

export type WorkoutDetail = {
  equipment?: Equipment[];
  map_data?: MapData;
  climbs?: ClimbSegment[];
  route_segment_matches?: RouteSegmentMatch[];
  records?: WorkoutIntervalRecord[];
  laps?: WorkoutLap[];
} & Workout;

export type MapData = {
  creator: string;
  center: MapCenter;
  extra_metrics?: string[];
  details?: MapDataDetails;
};

export type MapCenter = {
  tz: string;
  lat: number;
  lng: number;
};

export type MapDataDetails = {
  position: [number, number][]; // [[lat, lng], ...]
  time: string[];
  distance: number[]; // in km
  duration: number[]; // in seconds
  speed: number[]; // in m/s
  slope: number[];
  elevation: number[];

  extra_metrics?: Record<string, (number | null)[]>;
  zone_ranges?: ZoneRangeMap;
};

export type ZoneRangeDefinition = {
  zone: number;
  min: number | null;
  max?: number | null;
};

export type ZoneRangeMap = {
  'heart-rate'?: ZoneRangeDefinition[];
  power?: ZoneRangeDefinition[];
};

export type WorkoutLap = {
  start: string;
  stop: string;
  total_distance: number;
  total_duration: number;
  pause_duration: number;
  min_elevation: number;
  max_elevation: number;
  total_up: number;
  total_down: number;
  average_speed: number;
  average_speed_no_pause: number;
  max_speed: number;
  average_pace?: number;
  average_cadence: number;
  max_cadence: number;
  average_heart_rate: number;
  max_heart_rate: number;
  average_power: number;
  max_power: number;
};

export type WorkoutBreakdownItem = {
  start_index: number;
  end_index: number;
  distance: number;
  duration: number;
  min_elevation: number;
  max_elevation: number;
  total_up: number;
  total_down: number;
  average_speed: number;
  average_speed_no_pause: number;
  average_pace?: number;
  max_speed: number;
  average_cadence: number;
  max_cadence: number;
  average_heart_rate: number;
  max_heart_rate: number;
  average_power: number;
  max_power: number;
  is_best?: boolean;
  is_worst?: boolean;
};

export type WorkoutBreakdown = {
  mode: 'laps' | 'unit';
  items?: WorkoutBreakdownItem[];
};

export type WorkoutRangeStatsUnits = {
  distance: string;
  speed: string;
  elevation: string;
  temperature: string;
};

export type WorkoutRangeStats = {
  start_index: number;
  end_index: number;

  distance: number;
  duration: number;
  moving_duration: number;
  pause_duration: number;

  min_elevation: number;
  max_elevation: number;
  total_up: number;
  total_down: number;

  average_slope: number;
  min_slope: number;
  max_slope: number;

  average_speed: number;
  average_speed_no_pause: number;
  min_speed: number;
  max_speed: number;

  average_cadence?: number;
  min_cadence?: number;
  max_cadence?: number;

  average_heart_rate?: number;
  min_heart_rate?: number;
  max_heart_rate?: number;

  average_respiration_rate?: number;
  min_respiration_rate?: number;
  max_respiration_rate?: number;

  average_power?: number;
  min_power?: number;
  max_power?: number;

  average_temperature?: number;
  min_temperature?: number;
  max_temperature?: number;

  units: WorkoutRangeStatsUnits;
};

export type ClimbSegment = {
  index: number;
  type: string;
  start_distance: number;
  length: number;
  elevation: number;
  avg_slope: number;
  category: string;
  start_index: number;
  end_index: number;
  duration: number;
};

export type RouteSegmentMatch = {
  route_segment_id: number;
  workout_id: number;
  distance: number;
  duration: number;
  start_index: number;
  end_index: number;
  route_segment: RouteSegmentInfo;
};

export type RouteSegmentInfo = {
  id: number;
  name: string;
  notes?: string;
  filename: string;
  total_distance: number;
  min_elevation: number;
  max_elevation: number;
  total_up: number;
  total_down: number;
  bidirectional: boolean;
  circular: boolean;
  match_count: number;
  created_at: string;
  updated_at: string;
};

export type WorkoutIntervalRecord = {
  label: string;
  target_distance: number;
  distance: number;
  duration_seconds: number;
  average_speed: number;
  start_index?: number;
  end_index?: number;
  rank: number;
};

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

export type Totals = {
  workouts: number;
  distance: number;
  duration: number; // in seconds
  up: number;
  down: number;
};

export type RecordEntry = {
  value: number;
  workout_id: number;
  date: string;
};

export type DistanceRecordEntry = {
  label: string;
  target_distance: number;
  distance: number;
  duration_seconds: number;
  average_speed: number;
  workout_id: number;
  date: string;
  start_index?: number;
  end_index?: number;
};

export type ClimbRecordEntry = {
  elevation_gain: number;
  distance: number;
  average_slope: number;
  workout_id: number;
  date: string;
  start_index?: number;
  end_index?: number;
};

export type WorkoutRecord = {
  workout_type: string;
  active: boolean;
  distance?: RecordEntry;
  average_speed?: RecordEntry;
  average_speed_no_pause?: RecordEntry;
  max_speed?: RecordEntry;
  duration?: RecordEntry;
  total_up?: RecordEntry;
  distance_records?: DistanceRecordEntry[];
  biggest_climb?: ClimbRecordEntry;
};

export type CalendarEvent = {
  title: string;
  start: string;
  end: string;
  url: string;
};

export type WorkoutListParams = PaginationParams & {
  type?: string;
  active?: boolean;
  since?: string;
  order_by?: string;
  order_dir?: string;
};
