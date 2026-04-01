import {
  ChangeDetectionStrategy,
  Component,
  inject,
  input,
  signal,
  ViewEncapsulation,
} from '@angular/core';

import {
  CalendarEvent,
  CalendarMonthViewComponent,
  DateAdapter,
  provideCalendar,
} from 'angular-calendar';
import { adapterFactory } from 'angular-calendar/date-adapters/date-fns';
import { Api } from '../../../../core/services/api';
import { Router } from '@angular/router';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-workout-calendar',
  imports: [TranslatePipe, CalendarMonthViewComponent],
  providers: [
    provideCalendar({
      provide: DateAdapter,
      useFactory: adapterFactory,
    }),
  ],
  templateUrl: './workout-calendar.html',
  styleUrls: ['./workout-calendar.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  encapsulation: ViewEncapsulation.None,
})
export class WorkoutCalendar {
  public readonly handle = input<string | null>(null);
  public readonly locale = Intl.DateTimeFormat().resolvedOptions().locale;

  private readonly api = inject(Api);
  private readonly router = inject(Router);
  private readonly translate = inject(TranslateService);

  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly viewDate = signal(new Date());
  public readonly events = signal<CalendarEvent[]>([]);
  public readonly monthLabel = signal(this.formatMonthLabel(this.viewDate()));

  public constructor() {
    this.loadMonthEvents();
  }

  public previousMonth(): void {
    const d = this.viewDate();
    this.viewDate.set(new Date(d.getFullYear(), d.getMonth() - 1, 1));
    this.monthLabel.set(this.formatMonthLabel(this.viewDate()));
    this.loadMonthEvents();
  }

  public nextMonth(): void {
    const d = this.viewDate();
    this.viewDate.set(new Date(d.getFullYear(), d.getMonth() + 1, 1));
    this.monthLabel.set(this.formatMonthLabel(this.viewDate()));
    this.loadMonthEvents();
  }

  public goToToday(): void {
    this.viewDate.set(new Date());
    this.monthLabel.set(this.formatMonthLabel(this.viewDate()));
    this.loadMonthEvents();
  }

  public onEventClicked(event: CalendarEvent): void {
    const eventLink = (event.meta as { url?: string } | undefined)?.url;
    if (!eventLink) {
      return;
    }

    const match = eventLink.match(/\/workouts\/(\d+)/);
    if (!match?.[1]) {
      return;
    }

    this.router.navigate(['/workouts', match[1]]);
  }

  private loadMonthEvents(): void {
    const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    const monthDate = this.viewDate();
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
        handle: this.handle() || undefined,
        start: this.formatDateTime(gridStart),
        end: this.formatDateTime(gridEnd),
        timeZone,
      })
      .subscribe({
        next: (response) => {
          const events = (response.results || []).map((event) => ({
            title: event.title,
            start: new Date(event.start),
            end: event.end ? new Date(event.end) : undefined,
            meta: {
              url: event.url,
            },
          }));

          this.events.set(events);
          this.loading.set(false);
        },
        error: (err) => {
          this.loading.set(false);
          this.error.set(
            this.translate.instant('Failed to load {{page}} data. Please try again.', {
              page: this.translate.instant('calendar event'),
            }),
          );
          console.error('Failed to load calendar events:', err);
        },
      });
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

  private formatMonthLabel(date: Date): string {
    return date.toLocaleDateString(this.locale, {
      month: 'long',
      year: 'numeric',
    });
  }
}
