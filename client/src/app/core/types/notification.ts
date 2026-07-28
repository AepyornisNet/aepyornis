export type NotificationMeta = {
  url?: string;
  workout_id?: number;
  [key: string]: unknown;
};

export type Notification = {
  id: number;
  type: string;
  meta?: NotificationMeta | null;
  read_at?: string | null;
  created_at?: string;
  subject: string;
  msg: string;
};

export type UserNotificationSettings = {
  id: number;
  method: string;
  method_settings?: unknown;
  follow_request: boolean;
  workout_like: boolean;
  workout_reply: boolean;
};

export type UserNotificationSettingsUpdate = {
  method_settings: string;
  follow_request: boolean;
  workout_like: boolean;
  workout_reply: boolean;
};
