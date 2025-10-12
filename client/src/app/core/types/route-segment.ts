/**
 * Route segment domain models
 */

export interface RouteSegment {
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

export interface MapPoint {
  lat: number;
  lng: number;
  elevation: number;
  total_distance: number;
}

export interface RouteSegmentMatch {
  workout_id: number;
  workout_name: string;
  user_id: number;
  user_name: string;
  distance: number;
  duration: number;
  average_speed: number;
}

export interface RouteSegmentDetail extends RouteSegment {
  points: MapPoint[];
  matches: RouteSegmentMatch[];
  center: {
    lat: number;
    lng: number;
  };
  address_string: string;
}
