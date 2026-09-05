/**
 * Route segment domain models
 */

export type RouteSegmentDifficulty = 'easy' | 'moderate' | 'difficult' | '';

export type RouteSegmentMatch = {
  id?: number;
  workout_id: number;
  workout_name: string;
  workout_date?: string;
  profile_id?: number;
  profile_name?: string;
  user_id?: number;
  user_name?: string;
  distance: number;
  duration: number;
  average_speed: number;
};

export type CourseRecordInfo = {
  workout_id: number;
  workout_name: string;
  profile_id: number;
  profile_name: string;
  duration: number;
  speed: number;
};

export type RouteSegmentStats = {
  total_efforts: number;
  unique_athletes: number;
  avg_duration: number;
  avg_speed: number;
  course_record?: CourseRecordInfo;
};

export type RouteSegment = {
  id: number;
  profile_id: number;
  profile_name?: string;
  name: string;
  notes?: string;
  category?: string;
  visibility: 'public' | 'followers' | '' | 'private';
  description?: string;
  difficulty?: RouteSegmentDifficulty;
  filename: string;
  total_distance: number;
  min_elevation: number;
  max_elevation: number;
  total_up: number;
  total_down: number;
  bidirectional: boolean;
  circular: boolean;
  match_count: number;
  like_count?: number;
  has_liked?: boolean;
  can_edit?: boolean;
  can_delete?: boolean;
  created_at: string;
  updated_at: string;
};

export type MapPoint = {
  lat: number;
  lng: number;
  elevation?: number;
  total_distance?: number;
};

export type RouteSegmentDetail = {
  points: MapPoint[];
  matches: RouteSegmentMatch[];
  stats?: RouteSegmentStats;
  center: {
    lat: number;
    lng: number;
  };
  address_string?: string | null;
} & RouteSegment;
