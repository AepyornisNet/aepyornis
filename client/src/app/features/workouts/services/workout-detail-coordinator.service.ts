import { Injectable, signal } from '@angular/core';

/**
 * Interval selection for coordinating between breakdown, chart, and map components.
 */
export interface IntervalSelection {
  startIndex: number;
  endIndex: number;
}

/**
 * Service responsible for coordinating communication between workout detail components.
 * This service enables components to share state like interval selections across
 * the breakdown, chart, and map visualizations.
 */
@Injectable({
  providedIn: 'root',
})
export class WorkoutDetailCoordinatorService {
  // Signal to track the currently selected interval across all components
  readonly selectedInterval = signal<IntervalSelection | null>(null);

  /**
   * Select an interval range. This will update all subscribed components.
   * @param startIndex - The starting index of the interval
   * @param endIndex - The ending index of the interval
   */
  selectInterval(startIndex: number, endIndex: number): void {
    if (startIndex === -1 || endIndex === -1) {
      this.clearSelection();
    } else {
      this.selectedInterval.set({ startIndex, endIndex });
    }
  }

  /**
   * Clear the current interval selection.
   */
  clearSelection(): void {
    this.selectedInterval.set(null);
  }

  /**
   * Check if a specific interval is currently selected.
   * @param startIndex - The starting index to check
   * @param endIndex - The ending index to check
   * @returns true if the interval matches the current selection
   */
  isIntervalSelected(startIndex: number, endIndex: number): boolean {
    const current = this.selectedInterval();
    return current !== null && current.startIndex === startIndex && current.endIndex === endIndex;
  }
}
