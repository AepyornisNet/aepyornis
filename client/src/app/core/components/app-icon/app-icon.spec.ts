import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideZonelessChangeDetection } from '@angular/core';
import { provideIcons } from '@ng-icons/core';
import { faSolidDumbbell, faSolidQuestion } from '@ng-icons/font-awesome/solid';

import { AppIcon } from './app-icon';

describe('AppIcon', () => {
  let component: AppIcon;
  let fixture: ComponentFixture<AppIcon>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AppIcon],
      providers: [
        provideZonelessChangeDetection(),
        provideIcons({ faSolidDumbbell, faSolidQuestion }),
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(AppIcon);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should resolve icon name from icon map', () => {
    fixture.componentRef.setInput('name', 'workout');
    fixture.detectChanges();

    expect(component.iconName()).toBe('faSolidDumbbell');
  });

  it('should return default icon for unknown icon key', () => {
    fixture.componentRef.setInput('name', 'unknown-icon-key');
    fixture.detectChanges();

    expect(component.iconName()).toBe('faSolidQuestion');
  });
});
