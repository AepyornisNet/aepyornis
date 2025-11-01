import { Injectable } from '@angular/core';
import { MapDataDetails } from '../../../core/types/workout';

/**
 * Data structure for a calculated interval.
 */
export type IntervalData = {
  intervalNumber: number;
  startDistance: number;
  endDistance: number;
  duration: number; // milliseconds
  avgSpeed: number; // m/s
  pace: string; // min/km format
  avgElevation: number;
  elevationGain: number;
  elevationLoss: number;
  avgHeartRate?: number;
  avgCadence?: number;
  avgTemperature?: number;
  startIndex: number;
  endIndex: number;
  isFastest?: boolean;
  isSlowest?: boolean;
};

/**
 * Service responsible for calculating workout intervals and related statistics.
 */
@Injectable({
  providedIn: 'root',
})
export class WorkoutDetailIntervalService {
  /**
   * Calculate available interval distances based on total workout distance.
   */
  public calculateAvailableIntervals(mapData: MapDataDetails): number[] {
    if (!mapData || mapData.distance.length === 0) {
      return [1];
    }

    const totalDistanceKm = mapData.distance[mapData.distance.length - 1];
    const intervals = [1, 2, 5, 10, 25].filter((d) => d < totalDistanceKm);

    return intervals.length > 0 ? intervals : [1];
  }

  /**
   * Calculate intervals for a given distance.
   */
  public calculateIntervals(
    mapData: MapDataDetails,
    intervalDistanceKm: number,
    extraMetrics: string[],
  ): IntervalData[] {
    const intervals: IntervalData[] = [];

    if (!mapData || mapData.distance.length === 0) {
      return intervals;
    }

    let currentIntervalStart = 0;
    let intervalNumber = 1;

    for (let i = 0; i < mapData.distance.length; i++) {
      const distance = mapData.distance[i];

      // Check if we've reached the next interval boundary or end
      if (distance >= intervalDistanceKm * intervalNumber || i === mapData.distance.length - 1) {
        const interval = this.calculateIntervalData(
          mapData,
          intervalNumber,
          currentIntervalStart,
          i,
          extraMetrics,
        );
        intervals.push(interval);

        currentIntervalStart = i;
        intervalNumber++;
      }
    }

    // Mark fastest and slowest
    this.markFastestSlowest(intervals);

    return intervals;
  }

  /**
   * Calculate statistics for a single interval.
   */
  private calculateIntervalData(
    mapData: MapDataDetails,
    intervalNumber: number,
    startIndex: number,
    endIndex: number,
    extraMetrics: string[],
  ): IntervalData {
    const startDistance = mapData.distance[startIndex];
    const endDistance = mapData.distance[endIndex];

    const startTime = mapData.duration[startIndex];
    const endTime = mapData.duration[endIndex];
    const duration = (endTime - startTime) * 1000; // Convert to milliseconds

    // Calculate average speed from the data points
    const speedValues = mapData.speed.slice(startIndex, endIndex + 1).filter((s) => s > 0);
    const avgSpeed =
      speedValues.length > 0 ? speedValues.reduce((sum, s) => sum + s, 0) / speedValues.length : 0;
    const pace = this.calculatePace(avgSpeed);

    // Calculate elevation changes
    let elevationGain = 0;
    let elevationLoss = 0;
    for (let i = startIndex + 1; i <= endIndex; i++) {
      const elevDiff = mapData.elevation[i] - mapData.elevation[i - 1];
      if (elevDiff > 0) {
        elevationGain += elevDiff;
      } else {
        elevationLoss += Math.abs(elevDiff);
      }
    }

    const elevationValues = mapData.elevation.slice(startIndex, endIndex + 1);
    const avgElevation = elevationValues.reduce((sum, e) => sum + e, 0) / elevationValues.length;

    // Calculate extra metrics averages
    let avgHeartRate: number | undefined;
    let avgCadence: number | undefined;
    let avgTemperature: number | undefined;

    if (mapData.extra_metrics) {
      if (extraMetrics.includes('heart-rate') && mapData.extra_metrics['heart-rate']) {
        const hrValues = mapData.extra_metrics['heart-rate']
          .slice(startIndex, endIndex + 1)
          .filter((v): v is number => v !== null && v > 0);
        avgHeartRate =
          hrValues.length > 0
            ? hrValues.reduce((sum, v) => sum + v, 0) / hrValues.length
            : undefined;
      }

      if (extraMetrics.includes('cadence') && mapData.extra_metrics['cadence']) {
        const cadenceValues = mapData.extra_metrics['cadence']
          .slice(startIndex, endIndex + 1)
          .filter((v): v is number => v !== null && v > 0);
        avgCadence =
          cadenceValues.length > 0
            ? cadenceValues.reduce((sum: number, v: number) => sum + v, 0) / cadenceValues.length
            : undefined;
      }

      if (extraMetrics.includes('temperature') && mapData.extra_metrics['temperature']) {
        const tempValues = mapData.extra_metrics['temperature']
          .slice(startIndex, endIndex + 1)
          .filter((v): v is number => v !== null);
        avgTemperature =
          tempValues.length > 0
            ? tempValues.reduce((sum: number, v: number) => sum + v, 0) / tempValues.length
            : undefined;
      }
    }

    return {
      intervalNumber,
      startDistance,
      endDistance,
      duration,
      avgSpeed,
      pace,
      avgElevation,
      elevationGain,
      elevationLoss,
      avgHeartRate,
      avgCadence,
      avgTemperature,
      startIndex,
      endIndex,
    };
  }

  /**
   * Calculate pace from speed in m/s to min/km format.
   */
  private calculatePace(speedMps: number): string {
    if (speedMps === 0) {
      return '-';
    }

    const paceSecondsPerKm = 1000 / speedMps;
    const minutes = Math.floor(paceSecondsPerKm / 60);
    const seconds = Math.round(paceSecondsPerKm % 60);

    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  }

  /**
   * Mark the fastest and slowest intervals.
   */
  private markFastestSlowest(intervals: IntervalData[]): void {
    if (intervals.length === 0) {
      return;
    }

    let fastestIndex = 0;
    let slowestIndex = 0;
    let fastestSpeed = intervals[0].avgSpeed;
    let slowestSpeed = intervals[0].avgSpeed;

    for (let i = 1; i < intervals.length; i++) {
      const speed = intervals[i].avgSpeed;
      if (speed > fastestSpeed) {
        fastestSpeed = speed;
        fastestIndex = i;
      }
      if (speed < slowestSpeed && speed > 0) {
        slowestSpeed = speed;
        slowestIndex = i;
      }
    }

    intervals[fastestIndex].isFastest = true;
    intervals[slowestIndex].isSlowest = true;
  }
}
