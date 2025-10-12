import { Component, input, computed, ChangeDetectionStrategy, inject } from '@angular/core';
import { WorkoutPopupData } from '../../../../core/types/statistics';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { RouterLink } from '@angular/router';
import { User } from '../../../../core/services/user';

/**
 * WorkoutPopup Component
 *
 * Renders a popup card for displaying workout information on the heatmap.
 * Uses structured data from the API to render workout details with proper icons.
 */
@Component({
  selector: 'app-workout-popup',
  imports: [AppIcon, RouterLink],
  templateUrl: './workout-popup.html',
  styles: [`
    :host {
      display: block;
    }

    app-icon {
      display: inline-flex;
      width: 16px;
      height: 16px;
    }
  `],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkoutPopup {
  private userService = inject(User);
  
  /**
   * Workout data to display in the popup
   */
  data = input.required<WorkoutPopupData>();

  /**
   * Format workout type for display (convert snake_case to Title Case)
   */
  formatWorkoutType = computed(() => {
    const type = this.data().type;
    return type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
  });

  /**
   * Format duration in seconds to human-readable format (Xh Ym Zs)
   */
  formatDuration = computed(() => {
    const seconds = this.data().total_duration;
    if (seconds === undefined || seconds === null) return '';

    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);

    if (hours > 0) {
      return `${hours}h ${minutes}m ${secs}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${secs}s`;
    } else {
      return `${secs}s`;
    }
  });

  /**
   * Format distance in meters to km or mi
   */
  formatDistance = computed(() => {
    const meters = this.data().total_distance;
    if (meters === undefined || meters === null) return '';

    const unit = this.distanceUnit();
    if (unit === 'mi') {
      // Convert meters to miles
      const miles = meters * 0.000621371;
      return miles.toFixed(2);
    } else {
      // Convert meters to kilometers
      const km = meters / 1000;
      return km.toFixed(2);
    }
  });

  /**
   * Format weight in kg to kg or lbs
   */
  formatWeight = computed(() => {
    const kg = this.data().total_weight;
    if (kg === undefined || kg === null) return '';

    const unit = this.weightUnit();
    if (unit === 'lbs') {
      // Convert kg to lbs
      const lbs = kg * 2.20462;
      return lbs.toFixed(1);
    } else {
      return kg.toFixed(1);
    }
  });

  /**
   * Format speed in m/s to km/h or mph
   */
  formatSpeed = computed(() => {
    const mps = this.data().average_speed;
    if (mps === undefined || mps === null) return '';

    const unit = this.speedUnit();
    if (unit === 'mph') {
      // Convert m/s to mph
      const mph = mps * 2.23694;
      return mph.toFixed(1);
    } else {
      // Convert m/s to km/h
      const kmh = mps * 3.6;
      return kmh.toFixed(1);
    }
  });

  /**
   * Get distance unit from user profile
   */
  distanceUnit = computed(() => {
    const userInfo = this.userService.getUserInfo()();
    return userInfo?.profile?.preferred_units?.distance || 'km';
  });

  /**
   * Get weight unit from user profile
   */
  weightUnit = computed(() => {
    const userInfo = this.userService.getUserInfo()();
    return userInfo?.profile?.preferred_units?.weight || 'kg';
  });

  /**
   * Get speed unit from user profile
   */
  speedUnit = computed(() => {
    const userInfo = this.userService.getUserInfo()();
    return userInfo?.profile?.preferred_units?.speed || 'km/h';
  });
}
