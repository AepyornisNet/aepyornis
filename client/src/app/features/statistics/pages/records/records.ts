import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { WorkoutRecord } from '../../../../core/types/workout';
import { UserPreferredUnits } from '../../../../core/types/user';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { StatisticsNav } from '../../components/statistics-nav/statistics-nav';
import { TranslatePipe } from '@ngx-translate/core';
import { getSportLabel } from '../../../../core/i18n/sport-labels';

@Component({
  selector: 'app-statistics-records',
  standalone: true,
  imports: [RouterLink, AppIcon, StatisticsNav, TranslatePipe],
  templateUrl: './records.html',
  styleUrl: './records.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class StatisticsRecords implements OnInit {
  private api = inject(Api);

  private readonly metersToMiles = 0.000621371;
  private readonly metersToFeet = 3.28084;

  public readonly records = signal<WorkoutRecord[]>([]);
  public readonly preferredUnits = signal<UserPreferredUnits | null>(null);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly sportLabel = getSportLabel;

  public async ngOnInit(): Promise<void> {
    await this.loadData();
  }

  private async loadData(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const [profile, records] = await Promise.all([
        firstValueFrom(this.api.getProfile()),
        firstValueFrom(this.api.getRecords()),
      ]);

      if (profile?.results?.profile?.preferred_units) {
        this.preferredUnits.set(profile.results.profile.preferred_units);
      }

      if (records?.results) {
        this.records.set(records.results);
      }
    } catch (err) {
      console.error('Failed to load statistics records:', err);
      this.error.set('Failed to load records. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  public formatDistance(meters: number | undefined): string {
    if (meters === undefined || meters === null) {
      return '—';
    }

    const units = this.preferredUnits();
    if (!units || units.distance === 'km') {
      return `${(meters / 1000).toFixed(2)} km`;
    }

    return `${(meters * this.metersToMiles).toFixed(2)} mi`;
  }

  public formatElevation(meters: number | undefined): string {
    if (meters === undefined || meters === null) {
      return '—';
    }

    const units = this.preferredUnits();
    if (!units || units.elevation === 'm') {
      return `${meters.toFixed(0)} m`;
    }

    return `${(meters * this.metersToFeet).toFixed(0)} ft`;
  }

  public formatSpeed(metersPerSecond: number | undefined): string {
    if (!metersPerSecond && metersPerSecond !== 0) {
      return '—';
    }

    const units = this.preferredUnits();
    if (!units || units.speed === 'km/h') {
      return `${(metersPerSecond * 3.6).toFixed(2)} km/h`;
    }

    return `${(metersPerSecond * 2.23694).toFixed(2)} mph`;
  }

  public formatDuration(seconds: number | undefined): string {
    if (seconds === undefined || seconds === null) {
      return '—';
    }

    const rounded = Math.round(seconds);
    const hours = Math.floor(rounded / 3600);
    const minutes = Math.floor((rounded % 3600) / 60);
    const secs = rounded % 60;

    const parts: string[] = [];
    if (hours > 0) {
      parts.push(`${hours}h`);
    }
    if (minutes > 0) {
      parts.push(`${minutes}m`);
    }
    if (hours === 0 && minutes === 0) {
      parts.push(`${secs}s`);
    }

    return parts.join(' ');
  }

  public formatDate(date: string | undefined): string {
    if (!date) {
      return '';
    }
    return new Date(date).toLocaleDateString();
  }

  public hasDistanceRecords(record: WorkoutRecord): boolean {
    return Boolean(record.distance_records && record.distance_records.length > 0);
  }
}
