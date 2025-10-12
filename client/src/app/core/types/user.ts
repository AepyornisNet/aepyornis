/**
 * User domain models
 */

export interface UserProfile {
  id: number;
  username: string;
  name: string;
  active: boolean;
  admin: boolean;
  last_version: string;
  created_at: string;
  updated_at: string;
  preferred_units: UserPreferredUnits;
  language: string;
  theme: string;
  timezone: string;
  socials_disabled: boolean;
  prefer_full_date: boolean;
}

export interface AppInfo {
  version: string;
  version_sha: string;
  registration_disabled: boolean;
  socials_disabled: boolean;
}

export interface UserPreferredUnits {
  speed: string;
  distance: string;
  elevation: string;
  weight: string;
  height: string;
}

export interface ProfileSettings {
  preferred_units: UserPreferredUnits;
  language: string;
  theme: string;
  totals_show: string;
  timezone: string;
  auto_import_directory: string;
  api_active: boolean;
  api_key?: string;
  socials_disabled: boolean;
  prefer_full_date: boolean;
}

export interface FullUserProfile {
  id: number;
  username: string;
  name: string;
  active: boolean;
  admin: boolean;
  last_version: string;
  created_at: string;
  updated_at: string;
  profile: ProfileSettings;
}

export interface UserUpdateRequest {
  name: string;
  username: string;
  admin: boolean;
  active: boolean;
  password?: string;
}

export interface ProfileUpdateRequest {
  preferred_units: UserPreferredUnits;
  language: string;
  theme: string;
  totals_show: string;
  timezone: string;
  auto_import_directory: string;
  api_active: boolean;
  socials_disabled: boolean;
  prefer_full_date: boolean;
}

export interface AppConfig {
  registration_disabled: boolean;
  socials_disabled: boolean;
}
