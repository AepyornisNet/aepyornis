/**
 * User domain models
 */

export type UserProfile = {
  id: number;
  username: string;
  name: string;
  birthdate?: string;
  activity_pub: boolean;
  active: boolean;
  admin: boolean;
  last_version: string;
  created_at: string;
  updated_at: string;
  preferred_units: UserPreferredUnits;
  language: string;
  theme: string;
  timezone: string;
  default_workout_visibility: '' | 'followers' | 'public';
  prefer_full_date: boolean;
};

export type AppInfo = {
  version: string;
  version_sha: string;
  registration_disabled: boolean;
  socials_disabled: boolean;
  auto_import_enabled: boolean;
};

export type UserPreferredUnits = {
  speed: string;
  distance: string;
  elevation: string;
  weight: string;
  height: string;
};

export type ProfileSettings = {
  preferred_units: UserPreferredUnits;
  language: string;
  theme: string;
  totals_show: string;
  timezone: string;
  auto_import_directory: string;
  default_workout_visibility: '' | 'followers' | 'public';
  api_active: boolean;
  api_key?: string;
  prefer_full_date: boolean;
};

export type FullUserProfile = {
  id: number;
  username: string;
  name: string;
  birthdate?: string;
  activity_pub: boolean;
  active: boolean;
  admin: boolean;
  last_version: string;
  created_at: string;
  updated_at: string;
  profile: ProfileSettings;
};

export type UserUpdateRequest = {
  name: string;
  username: string;
  admin: boolean;
  active: boolean;
  password?: string;
};

export type ProfileUpdateRequest = {
  birthdate?: string;
  preferred_units: UserPreferredUnits;
  language: string;
  theme: string;
  totals_show: string;
  timezone: string;
  auto_import_directory: string;
  api_active: boolean;
  default_workout_visibility: '' | 'followers' | 'public';
  prefer_full_date: boolean;
};

export type AppConfig = {
  registration_disabled: boolean;
  socials_disabled: boolean;
};

export type FollowRequest = {
  id: number;
  actor_iri: string;
  created_at: string;
};

export type ActivityPubProfileSummary = {
  id: number;
  username: string;
  name: string;
  handle: string;
  actor_url: string;
  icon_url: string;
  is_external: boolean;
  is_own: boolean;
  is_following: boolean;
  posts_count: number;
  followers_count: number;
  following_count: number;
  member_since: string;
};

export type ActivityPubActor = {
  id: string;
  name?: string;
  preferredUsername?: string;
  published?: string;
  icon?: string | { url?: string };
};
