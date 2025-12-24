import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { DistanceRecordEntry } from '../../../../core/types/workout';
import { UserPreferredUnits } from '../../../../core/types/user';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-records-ranking',
  standalone: true,
  imports: [RouterLink, AppIcon, TranslatePipe],
  templateUrl: './records-ranking.html',
  styleUrl: './records-ranking.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RecordsRankingPage implements OnInit {
  private api = inject(Api);
  private route = inject(ActivatedRoute);

  private readonly metersToMiles = 0.000621371;
  private readonly metersToFeet = 3.28084;

  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly records = signal<DistanceRecordEntry[]>([]);
  public readonly preferredUnits = signal<UserPreferredUnits | null>(null);
  public readonly page = signal(1);
  public readonly totalPages = signal(1);
  public readonly perPage = 20;

  public workoutType = '';
  public label = '';

  public async ngOnInit(): Promise<void> {
    const workoutType = this.route.snapshot.paramMap.get('workoutType') || '';
    const label = this.route.snapshot.paramMap.get('label') || '';

    this.workoutType = workoutType;
    this.label = decodeURIComponent(label);

    await this.loadData();
  }

  private async loadData(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const [profile, ranking] = await Promise.all([
        firstValueFrom(this.api.getProfile()),
        firstValueFrom(
          this.api.getDistanceRecordRanking({
            workout_type: this.workoutType,
            label: this.label,
            page: this.page(),
            per_page: this.perPage,
          }),
        ),
      ]);

      if (profile?.results?.profile?.preferred_units) {
        this.preferredUnits.set(profile.results.profile.preferred_units);
      }

      this.records.set(ranking.results ?? []);
      this.totalPages.set(ranking.total_pages ?? 1);
    } catch (err) {
      console.error('Failed to load distance ranking', err);
      this.error.set('Failed to load ranking. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  public async changePage(delta: number): Promise<void> {
    const next = this.page() + delta;
    if (next < 1 || next > this.totalPages()) {
      return;
    }
    this.page.set(next);
    await this.loadData();
  }

  public rankNumber(index: number): number {
    return (this.page() - 1) * this.perPage + index + 1;
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
}
