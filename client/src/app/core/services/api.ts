import { inject, Injectable } from '@angular/core';
import { HttpResponse } from '@angular/common/http';
import { Observable } from 'rxjs';
import { APIResponse, PaginatedAPIResponse, PaginationParams } from '../../core/types/api-response';
import {
  ActivityPubActor,
  ActivityPubProfileSummary,
  AppConfig,
  AppInfo,
  FollowRequest,
  FullUserProfile,
  HammerheadConnectionStatus,
  ProfileChangePasswordRequest,
  ProfileUpdateRequest,
  UserProfile,
  UserUpdateRequest,
  UserWebpushSubscription,
} from '../../core/types/user';
import {
  CalendarEvent,
  ClimbRecordEntry,
  DistanceRecordEntry,
  Totals,
  Workout,
  WorkoutBreakdown,
  WorkoutDetail,
  WorkoutListParams,
  WorkoutRangeStats,
  WorkoutRecord,
  WorkoutReply,
} from '../../core/types/workout';
import { Measurement } from '../../core/types/measurement';
import { Equipment } from '../../core/types/equipment';
import {
  RouteSegment,
  RouteSegmentDetail,
  RouteSegmentDifficulty,
  RouteSegmentMatch,
} from '../../core/types/route-segment';
import {
  GeoJsonFeatureCollection,
  HeatmapCoordinateList,
  Statistics,
  StatisticsParams,
} from '../../core/types/statistics';
import { RegisterRequest, SignInRequest } from '../../core/types/auth';
import {
  Notification,
  UserNotificationSettings,
  UserNotificationSettingsUpdate,
} from '../types/notification';
import { Like } from '../types/like';

import { AuthApiService } from '../../features/auth/services/auth-api.service';
import { NotificationApiService } from './notification-api.service';
import { WorkoutApiService } from '../../features/workouts/services/workout-api.service';
import { MeasurementApiService } from '../../features/measurements/services/measurement-api.service';
import { EquipmentApiService } from '../../features/equipment/services/equipment-api.service';
import { RouteSegmentApiService } from '../../features/route-segments/services/route-segment-api.service';
import { ProfileApiService } from '../../features/profile/services/profile-api.service';

/**
 * Api serves as the primary gateway and backward-compatible facade
 * delegating to feature-scoped API services.
 */
@Injectable({
  providedIn: 'root',
})
export class Api {
  public authApi = inject(AuthApiService);
  public notificationApi = inject(NotificationApiService);
  public workoutApi = inject(WorkoutApiService);
  public measurementApi = inject(MeasurementApiService);
  public equipmentApi = inject(EquipmentApiService);
  public routeSegmentApi = inject(RouteSegmentApiService);
  public profileApi = inject(ProfileApiService);

  // Auth endpoints
  public signIn(payload: SignInRequest): Observable<APIResponse<UserProfile>> {
    return this.authApi.signIn(payload);
  }

  public register(payload: RegisterRequest): Observable<APIResponse<{ message: string }>> {
    return this.authApi.register(payload);
  }

  public signOut(): Observable<APIResponse<{ message: string }>> {
    return this.authApi.signOut();
  }

  public whoami(): Observable<APIResponse<UserProfile>> {
    return this.authApi.whoami();
  }

  // Notification endpoints
  public getNotifications(
    params?: PaginationParams,
  ): Observable<PaginatedAPIResponse<Notification>> {
    return this.notificationApi.getNotifications(params);
  }

  public getNotificationSettings(): Observable<APIResponse<UserNotificationSettings[]>> {
    return this.notificationApi.getNotificationSettings();
  }

  public updateNotificationSettings(
    method: string,
    payload: UserNotificationSettingsUpdate,
  ): Observable<APIResponse<UserNotificationSettings>> {
    return this.notificationApi.updateNotificationSettings(method, payload);
  }

  public getWebpushSubscriptions(): Observable<APIResponse<UserWebpushSubscription[]>> {
    return this.notificationApi.getWebpushSubscriptions();
  }

  public subscribeWebpush(payload: {
    endpoint: string;
    keys: { auth: string; p256dh: string };
    user_agent?: string;
  }): Observable<APIResponse<UserWebpushSubscription>> {
    return this.notificationApi.subscribeWebpush(payload);
  }

  public unsubscribeWebpush(endpoint?: string): Observable<APIResponse<{ success: boolean }>> {
    return this.notificationApi.unsubscribeWebpush(endpoint);
  }

  public markNotificationsAsRead(ids?: number[]): Observable<APIResponse<{ success: boolean }>> {
    return this.notificationApi.markNotificationsAsRead(ids);
  }

  public getAppInfo(): Observable<APIResponse<AppInfo>> {
    return this.profileApi.getAppInfo();
  }

  // Workouts endpoints
  public getWorkouts(params?: WorkoutListParams): Observable<PaginatedAPIResponse<Workout>> {
    return this.workoutApi.getWorkouts(params);
  }

  public getWorkout(id: number): Observable<APIResponse<WorkoutDetail>> {
    return this.workoutApi.getWorkout(id);
  }

  public getWorkoutBreakdown(
    id: number,
    params?: { count?: number; mode?: 'auto' | 'laps' | 'unit' | string },
  ): Observable<APIResponse<WorkoutBreakdown>> {
    return this.workoutApi.getWorkoutBreakdown(id, params);
  }

  public getWorkoutRangeStats(
    id: number,
    params?: { start_index?: number; end_index?: number },
  ): Observable<APIResponse<WorkoutRangeStats>> {
    return this.workoutApi.getWorkoutRangeStats(id, params);
  }

  public getRecentWorkouts(
    limit?: number,
    offset?: number,
    scope?: 'following' | 'global',
    handle?: string,
  ): Observable<APIResponse<Workout[]>> {
    return this.workoutApi.getRecentWorkouts(limit, offset, scope, handle);
  }

  public getWorkoutFilterOptions(): Observable<
    APIResponse<{ types: string[]; sub_types_by_type: Record<string, string[]> }>
  > {
    return this.workoutApi.getWorkoutFilterOptions();
  }

  public createWorkoutFromFile(formData: FormData): Observable<APIResponse<Workout[]>> {
    return this.workoutApi.createWorkoutFromFile(formData);
  }

  public createWorkoutManual(workout: {
    name: string;
    date: string;
    timezone: string;
    location?: string;
    duration_hours?: number;
    duration_minutes?: number;
    duration_seconds?: number;
    distance?: number;
    repetitions?: number;
    weight?: number;
    notes?: string;
    type: string;
    visibility?: '' | 'followers' | 'public';
    custom_type?: string;
    equipment_ids?: number[];
  }): Observable<APIResponse<Workout>> {
    return this.workoutApi.createWorkoutManual(workout);
  }

  public updateWorkout(
    id: number,
    workout: {
      name?: string;
      date?: string;
      timezone?: string;
      location?: string;
      duration_hours?: number;
      duration_minutes?: number;
      duration_seconds?: number;
      distance?: number;
      repetitions?: number;
      weight?: number;
      notes?: string;
      type?: string;
      visibility?: '' | 'followers' | 'public';
      custom_type?: string;
      equipment_ids?: number[];
    },
  ): Observable<APIResponse<Workout>> {
    return this.workoutApi.updateWorkout(id, workout);
  }

  public deleteWorkout(id: number): Observable<APIResponse<{ message: string }>> {
    return this.workoutApi.deleteWorkout(id);
  }

  public downloadWorkoutsZip(ids: number[]): Observable<Blob> {
    return this.workoutApi.downloadWorkoutsZip(ids);
  }

  public addEquipmentToWorkouts(
    workoutIds: number[],
    equipmentIds: number[],
  ): Observable<APIResponse<{ message: string }>> {
    return this.workoutApi.addEquipmentToWorkouts(workoutIds, equipmentIds);
  }

  public toggleWorkoutLock(id: number): Observable<APIResponse<Workout>> {
    return this.workoutApi.toggleWorkoutLock(id);
  }

  public refreshWorkout(id: number): Observable<APIResponse<{ message: string }>> {
    return this.workoutApi.refreshWorkout(id);
  }

  public likeWorkout(
    id: number,
  ): Observable<APIResponse<{ workout_id: number; likes_count: number; liked: boolean }>> {
    return this.workoutApi.likeWorkout(id);
  }

  public getWorkoutLikes(workoutId: number): Observable<APIResponse<Like[]>> {
    return this.workoutApi.getWorkoutLikes(workoutId);
  }

  public createReply(workoutId: number, content: string): Observable<APIResponse<WorkoutReply>> {
    return this.workoutApi.createReply(workoutId, content);
  }

  public getWorkoutReplies(
    workoutId: number,
    params?: PaginationParams,
  ): Observable<PaginatedAPIResponse<WorkoutReply>> {
    return this.workoutApi.getWorkoutReplies(workoutId, params);
  }

  public downloadWorkout(id: number): Observable<HttpResponse<Blob>> {
    return this.workoutApi.downloadWorkout(id);
  }

  // Measurements endpoints
  public getMeasurements(params?: PaginationParams): Observable<PaginatedAPIResponse<Measurement>> {
    return this.measurementApi.getMeasurements(params);
  }

  public createOrUpdateMeasurement(measurement: {
    date: string;
    weight?: number;
    height?: number;
    steps?: number;
    ftp?: number;
    resting_heart_rate?: number;
    max_heart_rate?: number;
  }): Observable<APIResponse<Measurement>> {
    return this.measurementApi.createOrUpdateMeasurement(measurement);
  }

  public deleteMeasurement(date: string): Observable<void> {
    return this.measurementApi.deleteMeasurement(date);
  }

  // Equipment endpoints
  public getEquipment(params?: PaginationParams): Observable<PaginatedAPIResponse<Equipment>> {
    return this.equipmentApi.getEquipment(params);
  }

  public getEquipmentById(id: number): Observable<APIResponse<Equipment>> {
    return this.equipmentApi.getEquipmentById(id);
  }

  public createEquipment(equipment: Partial<Equipment>): Observable<APIResponse<Equipment>> {
    return this.equipmentApi.createEquipment(equipment);
  }

  public updateEquipment(
    id: number,
    equipment: Partial<Equipment>,
  ): Observable<APIResponse<Equipment>> {
    return this.equipmentApi.updateEquipment(id, equipment);
  }

  public deleteEquipment(id: number): Observable<void> {
    return this.equipmentApi.deleteEquipment(id);
  }

  // Route segments endpoints
  public getRouteSegments(
    params?: PaginationParams,
  ): Observable<PaginatedAPIResponse<RouteSegment>> {
    return this.routeSegmentApi.getRouteSegments(params);
  }

  public getRouteSegment(id: number): Observable<APIResponse<RouteSegmentDetail>> {
    return this.routeSegmentApi.getRouteSegment(id);
  }

  public createRouteSegment(formData: FormData): Observable<APIResponse<RouteSegment[]>> {
    return this.routeSegmentApi.createRouteSegment(formData);
  }

  public createRouteSegmentFromWorkout(
    workoutId: number,
    params: {
      name: string;
      start: number;
      end: number;
      category?: string;
      visibility?: 'public' | 'followers' | '' | 'private';
      description?: string;
      difficulty?: RouteSegmentDifficulty;
      notes?: string;
      bidirectional?: boolean;
      circular?: boolean;
    },
  ): Observable<APIResponse<RouteSegmentDetail>> {
    return this.routeSegmentApi.createRouteSegmentFromWorkout(workoutId, params);
  }

  public updateRouteSegment(
    id: number,
    params: {
      name: string;
      notes?: string;
      category?: string;
      visibility?: 'public' | 'followers' | '' | 'private';
      description?: string;
      difficulty?: RouteSegmentDifficulty;
      bidirectional?: boolean;
      circular?: boolean;
    },
  ): Observable<APIResponse<RouteSegmentDetail>> {
    return this.routeSegmentApi.updateRouteSegment(id, params);
  }

  public deleteRouteSegment(id: number): Observable<APIResponse<{ message: string }>> {
    return this.routeSegmentApi.deleteRouteSegment(id);
  }

  public refreshRouteSegment(id: number): Observable<APIResponse<{ message: string }>> {
    return this.routeSegmentApi.refreshRouteSegment(id);
  }

  public findRouteSegmentMatches(id: number): Observable<APIResponse<{ message: string }>> {
    return this.routeSegmentApi.findRouteSegmentMatches(id);
  }

  public downloadRouteSegment(id: number): Observable<Blob> {
    return this.routeSegmentApi.downloadRouteSegment(id);
  }

  public getRouteSegmentMatches(
    id: number,
    params?: { page?: number; per_page?: number; sort?: string },
  ): Observable<PaginatedAPIResponse<RouteSegmentMatch>> {
    return this.routeSegmentApi.getRouteSegmentMatches(id, params);
  }

  public likeRouteSegment(
    id: number,
  ): Observable<APIResponse<{ liked: boolean; like_count: number }>> {
    return this.routeSegmentApi.likeRouteSegment(id);
  }

  public unlikeRouteSegment(
    id: number,
  ): Observable<APIResponse<{ liked: boolean; like_count: number }>> {
    return this.routeSegmentApi.unlikeRouteSegment(id);
  }

  public getRouteSegmentLikes(id: number): Observable<APIResponse<Like[]>> {
    return this.routeSegmentApi.getRouteSegmentLikes(id);
  }

  // Dashboard endpoints
  public getTotals(handle?: string): Observable<APIResponse<Totals[]>> {
    return this.profileApi.getTotals(handle);
  }

  public getRecords(handle?: string): Observable<APIResponse<WorkoutRecord[]>> {
    return this.profileApi.getRecords(handle);
  }

  public getDistanceRecordRanking(params: {
    workout_type: string;
    label: string;
    handle?: string;
    start?: string;
    end?: string;
    page?: number;
    per_page?: number;
  }): Observable<PaginatedAPIResponse<DistanceRecordEntry>> {
    return this.profileApi.getDistanceRecordRanking(params);
  }

  public getClimbRanking(params: {
    workout_type: string;
    handle?: string;
    start?: string;
    end?: string;
    page?: number;
    per_page?: number;
  }): Observable<PaginatedAPIResponse<ClimbRecordEntry>> {
    return this.profileApi.getClimbRanking(params);
  }

  // Profile endpoints
  public getProfile(): Observable<APIResponse<FullUserProfile>> {
    return this.profileApi.getProfile();
  }

  public updateProfile(profile: ProfileUpdateRequest): Observable<APIResponse<FullUserProfile>> {
    return this.profileApi.updateProfile(profile);
  }

  public changePassword(
    payload: ProfileChangePasswordRequest,
  ): Observable<APIResponse<{ message: string }>> {
    return this.profileApi.changePassword(payload);
  }

  public resetAPIKey(): Observable<APIResponse<{ api_key: string; message: string }>> {
    return this.profileApi.resetAPIKey();
  }

  public enableActivityPub(): Observable<APIResponse<{ activity_pub: boolean; message: string }>> {
    return this.profileApi.enableActivityPub();
  }

  public getFollowRequests(): Observable<APIResponse<FollowRequest[]>> {
    return this.profileApi.getFollowRequests();
  }

  public getUserProfileSummary(
    handle?: string,
  ): Observable<APIResponse<ActivityPubProfileSummary>> {
    return this.profileApi.getUserProfileSummary(handle);
  }

  public searchProfiles(query: string): Observable<APIResponse<ActivityPubProfileSummary[]>> {
    return this.profileApi.searchProfiles(query);
  }

  public getLocalActivityPubActor(username: string): Observable<ActivityPubActor> {
    return this.profileApi.getLocalActivityPubActor(username);
  }

  public followUserByHandle(handle: string): Observable<APIResponse<ActivityPubProfileSummary>> {
    return this.profileApi.followUserByHandle(handle);
  }

  public unfollowUserByHandle(handle: string): Observable<APIResponse<ActivityPubProfileSummary>> {
    return this.profileApi.unfollowUserByHandle(handle);
  }

  public acceptFollowRequest(id: number): Observable<APIResponse<FollowRequest>> {
    return this.profileApi.acceptFollowRequest(id);
  }

  public refreshWorkouts(): Observable<APIResponse<{ message: string }>> {
    return this.profileApi.refreshWorkouts();
  }

  public getHammerheadConnection(): Observable<APIResponse<HammerheadConnectionStatus>> {
    return this.profileApi.getHammerheadConnection();
  }

  public connectHammerhead(): Observable<APIResponse<{ authorize_url: string }>> {
    return this.profileApi.connectHammerhead();
  }

  public disconnectHammerhead(): Observable<APIResponse<{ message: string }>> {
    return this.profileApi.disconnectHammerhead();
  }

  // Admin endpoints
  public getUsers(): Observable<APIResponse<UserProfile[]>> {
    return this.profileApi.getUsers();
  }

  public getUser(id: number): Observable<APIResponse<UserProfile>> {
    return this.profileApi.getUser(id);
  }

  public updateUser(id: number, user: UserUpdateRequest): Observable<APIResponse<UserProfile>> {
    return this.profileApi.updateUser(id, user);
  }

  public deleteUser(id: number): Observable<APIResponse<{ message: string }>> {
    return this.profileApi.deleteUser(id);
  }

  public updateAppConfig(config: AppConfig): Observable<APIResponse<AppInfo>> {
    return this.profileApi.updateAppConfig(config);
  }

  // Statistics endpoints
  public getStatistics(params?: StatisticsParams): Observable<APIResponse<Statistics>> {
    return this.profileApi.getStatistics(params);
  }

  public getWorkoutsCoordinates(params?: {
    cellSize?: number;
    minLat?: number;
    minLng?: number;
    maxLat?: number;
    maxLng?: number;
    type?: string;
    sub_type?: string;
    since?: string;
  }): Observable<APIResponse<HeatmapCoordinateList>> {
    return this.profileApi.getWorkoutsCoordinates(params);
  }

  public getWorkoutsCenters(params?: {
    type?: string;
    sub_type?: string;
    since?: string;
  }): Observable<APIResponse<GeoJsonFeatureCollection>> {
    return this.profileApi.getWorkoutsCenters(params);
  }

  // Calendar endpoints
  public getCalendarEvents(params?: {
    handle?: string;
    start?: string;
    end?: string;
    timeZone?: string;
  }): Observable<APIResponse<CalendarEvent[]>> {
    return this.profileApi.getCalendarEvents(params);
  }

  // Legal endpoints
  public getLegalNotice(lang: string): Observable<string> {
    return this.profileApi.getLegalNotice(lang);
  }

  public getPrivacyPolicy(lang: string): Observable<string> {
    return this.profileApi.getPrivacyPolicy(lang);
  }
}
