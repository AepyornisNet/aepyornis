/**
 * Workout domain models
 */

import { UserProfile } from './user';

export interface Workout {
  id: number;
  date: string;
  name: string;
  notes: string;
  type: string;
  custom_type?: string;
  user_id: number;
  user?: UserProfile;
  public_uuid?: string;
  locked: boolean;
  created_at: string;
  updated_at: string;
  has_file: boolean;
  has_tracks: boolean;

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
}

export type WorkoutDetail = {
  equipment?: Equipment[];
  map_data?: MapData;
  climbs?: ClimbSegment[];
  route_segment_matches?: RouteSegmentMatch[];
} & Workout;

export interface MapData {
  creator: string;
  center: MapCenter;
  extra_metrics?: string[];
  details?: MapDataDetails;
}

export interface MapCenter {
  tz: string;
  lat: number;
  lng: number;
}

export interface MapDataDetails {
  position: number[][]; // [[lat, lng], ...]
  time: string[];
  distance: number[]; // in km
  duration: number[]; // in seconds
  speed: number[]; // in m/s
  slope: number[];
  elevation: number[];

  extra_metrics?: Record<string, (number | null)[]>;
}

export interface ClimbSegment {
  index: number;
  type: string;
  start_distance: number;
  length: number;
  elevation: number;
  avg_slope: number;
  category: string;
}

export interface RouteSegmentMatch {
  route_segment_id: number;
  workout_id: number;
  route_segment: RouteSegmentInfo;
}

export interface RouteSegmentInfo {
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
}

export interface Equipment {
  id: number;
  name: string;
  description?: string;
  notes?: string;
  active: boolean;
  default_for?: string[];
  user_id: number;
  created_at: string;
  updated_at: string;
}

export interface Totals {
  workouts: number;
  distance: number;
  duration: number; // in seconds
  up: number;
  down: number;
}

export interface RecordEntry {
  value: number;
  workout_id: number;
  date: string;
}

export interface WorkoutRecord {
  workout_type: string;
  active: boolean;
  distance?: RecordEntry;
  average_speed?: RecordEntry;
  average_speed_no_pause?: RecordEntry;
  max_speed?: RecordEntry;
  duration?: RecordEntry;
  total_up?: RecordEntry;
}

export interface CalendarEvent {
  title: string;
  start: string;
  end: string;
  url: string;
}
