import { Component, Input, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MapDataDetails } from '../../../../core/types/workout';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { WorkoutDetailIntervalService, IntervalData } from '../../services/workout-detail-interval.service';

@Component({
  selector: 'app-workout-breakdown',
  imports: [CommonModule],
  templateUrl: './workout-breakdown.html',
  styleUrl: './workout-breakdown.scss'
})
export class WorkoutBreakdownComponent implements OnInit {
  @Input() mapData?: MapDataDetails;
  @Input() extraMetrics: string[] = [];
  
  private coordinatorService = inject(WorkoutDetailCoordinatorService);
  private intervalService = inject(WorkoutDetailIntervalService);

  intervalDistance = 1; // km
  availableIntervals: number[] = [];
  intervals: IntervalData[] = [];
  selectedIntervalIndex: number | null = null;

  ngOnInit() {
    if (this.mapData) {
      this.availableIntervals = this.intervalService.calculateAvailableIntervals(this.mapData);
      this.intervals = this.intervalService.calculateIntervals(
        this.mapData,
        this.intervalDistance,
        this.extraMetrics
      );
    }
  }

  setIntervalDistance(distance: number) {
    this.intervalDistance = distance;
    this.selectedIntervalIndex = null;
    
    if (this.mapData) {
      this.intervals = this.intervalService.calculateIntervals(
        this.mapData,
        this.intervalDistance,
        this.extraMetrics
      );
    }
    
    this.coordinatorService.clearSelection();
  }

  selectInterval(index: number) {
    if (this.selectedIntervalIndex === index) {
      // Deselect
      this.selectedIntervalIndex = null;
      this.coordinatorService.clearSelection();
    } else {
      // Select new interval
      this.selectedIntervalIndex = index;
      const interval = this.intervals[index];
      this.coordinatorService.selectInterval(interval.startIndex, interval.endIndex);
    }
  }

  formatDuration(milliseconds: number): string {
    const totalSeconds = Math.floor(milliseconds / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  }

  formatSpeed(speedMps: number): string {
    return (speedMps * 3.6).toFixed(2); // Convert m/s to km/h
  }

  hasExtraMetric(metric: string): boolean {
    return this.extraMetrics.includes(metric);
  }
}
