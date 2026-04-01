import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';

import { TranslatePipe, TranslateService } from '@ngx-translate/core';
import { Api } from '../../../../core/services/api';

@Component({
  selector: 'app-feed-calendar',
  imports: [TranslatePipe, DatePipe],
  templateUrl: './feed-calendar.html',
  styleUrl: './feed-calendar.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FeedCalendar {
  private readonly api = inject(Api);
  private readonly translate = inject(TranslateService);

  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly currentMonthDate = signal(new Date());
  public readonly activeDays = signal<Set<string>>(new Set<string>());
  public readonly monthLabel = computed(() =>
    this.currentMonthDate().toLocaleDateString(Intl.DateTimeFormat().resolvedOptions().locale, {
      month: 'long',
      year: 'numeric',
    }),
  );
  public readonly weekdayLabels = computed(() => this.buildWeekdayLabels());
  public readonly monthDays = computed(() => this.buildMonthDays());

  public constructor() {
    this.loadCurrentMonthEvents();
  }

  public hasActivity(dayKey: string): boolean {
    return this.activeDays().has(dayKey);
  }

  private loadCurrentMonthEvents(): void {
    const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    const monthDate = this.currentMonthDate();
    const year = monthDate.getFullYear();
    const month = monthDate.getMonth();
    const firstOfMonth = new Date(year, month, 1);
    const mondayIndex = (firstOfMonth.getDay() + 6) % 7;
    const gridStart = new Date(year, month, 1 - mondayIndex, 0, 0, 0);
    const gridEnd = new Date(gridStart);
    gridEnd.setDate(gridStart.getDate() + 41);
    gridEnd.setHours(23, 59, 59, 0);

    this.loading.set(true);
    this.error.set(null);

    this.api
      .getCalendarEvents({
        start: this.formatDateTime(gridStart),
        end: this.formatDateTime(gridEnd),
        timeZone,
      })
      .subscribe({
        next: (response) => {
          const uniqueDays = new Set<string>();
          for (const event of response.results || []) {
            uniqueDays.add(event.start.slice(0, 10));
          }

          this.activeDays.set(uniqueDays);
          this.loading.set(false);
        },
        error: (err) => {
          this.loading.set(false);
          this.error.set(
            this.translate.instant('Failed to load {{page}} data. Please try again.', {
              page: this.translate.instant('calendar event'),
            }),
          );
          console.error('Failed to load feed calendar events:', err);
        },
      });
  }

  private buildWeekdayLabels(): string[] {
    const formatter = new Intl.DateTimeFormat(Intl.DateTimeFormat().resolvedOptions().locale, {
      weekday: 'short',
    });
    const baseMonday = new Date(Date.UTC(2025, 0, 6));
    const labels: string[] = [];
    for (let i = 0; i < 7; i += 1) {
      labels.push(formatter.format(new Date(baseMonday.getTime() + i * 24 * 60 * 60 * 1000)));
    }

    return labels;
  }

  private buildMonthDays(): { date: Date; key: string; inCurrentMonth: boolean }[] {
    const monthDate = this.currentMonthDate();
    const year = monthDate.getFullYear();
    const month = monthDate.getMonth();
    const firstOfMonth = new Date(year, month, 1);
    const mondayIndex = (firstOfMonth.getDay() + 6) % 7;
    const gridStart = new Date(year, month, 1 - mondayIndex);

    const days: { date: Date; key: string; inCurrentMonth: boolean }[] = [];
    for (let i = 0; i < 42; i += 1) {
      const day = new Date(gridStart);
      day.setDate(gridStart.getDate() + i);
      days.push({
        date: day,
        key: this.toLocalDateKey(day),
        inCurrentMonth: day.getMonth() === month,
      });
      if (days.length >= 35 && day.getMonth() !== month && day.getDay() === 0) {
        break;
      }
    }

    return days;
  }

  private formatDateTime(d: Date): string {
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const hours = String(d.getHours()).padStart(2, '0');
    const minutes = String(d.getMinutes()).padStart(2, '0');
    const seconds = String(d.getSeconds()).padStart(2, '0');
    return `${year}-${month}-${day}T${hours}:${minutes}:${seconds}`;
  }

  private toLocalDateKey(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }
}
