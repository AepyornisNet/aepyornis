import { Component, OnDestroy, AfterViewInit, signal, inject, ChangeDetectionStrategy, ViewChild, ElementRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Calendar } from '@fullcalendar/core';
import dayGridPlugin from '@fullcalendar/daygrid';
import interactionPlugin from '@fullcalendar/interaction';
import { Api } from '../../../../core/services/api';
import { Router } from '@angular/router';

@Component({
  selector: 'app-workout-calendar',
  imports: [CommonModule],
  templateUrl: './workout-calendar.html',
  styleUrl: './workout-calendar.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkoutCalendar implements AfterViewInit, OnDestroy {
  @ViewChild('calendarContainer', { static: false }) calendarContainer!: ElementRef;
  
  private api = inject(Api);
  private router = inject(Router);
  
  loading = signal(true);
  error = signal<string | null>(null);
  
  private calendar: Calendar | null = null;

  ngAfterViewInit() {
    this.initializeCalendar();
  }

  ngOnDestroy() {
    if (this.calendar) {
      this.calendar.destroy();
    }
  }

  private initializeCalendar() {
    if (!this.calendarContainer) {
      return;
    }

    const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    
    this.calendar = new Calendar(this.calendarContainer.nativeElement, {
      plugins: [dayGridPlugin, interactionPlugin],
      initialView: 'dayGridMonth',
      timeZone: timeZone,
      locale: Intl.DateTimeFormat().resolvedOptions().locale,
      firstDay: 1,
      height: 'auto',
      headerToolbar: {
        left: 'prev,next today',
        center: 'title',
        right: ''
      },
      eventClick: (info) => {
        info.jsEvent.preventDefault();
        if (info.event.url) {
          // Extract workout ID from URL and navigate
          const match = info.event.url.match(/\/workouts\/(\d+)/);
          if (match && match[1]) {
            this.router.navigate(['/workouts', match[1]]);
          }
        }
      },
      events: (fetchInfo, successCallback, failureCallback) => {
        this.loading.set(true);
        this.error.set(null);
        
        const params = {
          start: fetchInfo.startStr,
          end: fetchInfo.endStr,
          timeZone: timeZone
        };
        
        this.api.getCalendarEvents(params).subscribe({
          next: (response) => {
            this.loading.set(false);
            if (response.results) {
              successCallback(response.results);
            } else {
              successCallback([]);
            }
          },
          error: (err) => {
            this.loading.set(false);
            this.error.set('Failed to load calendar events');
            console.error('Failed to load calendar events:', err);
            failureCallback(err);
          }
        });
      }
    });
    
    this.calendar.render();
  }
}
