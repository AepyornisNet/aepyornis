import { _ } from '@ngx-translate/core';
import { getWorkoutTypeConfig } from '../types/workout-types';

export const getAverageSpeedLabel = (value?: string | null): string => {
  if (!value) {
    return '';
  }
  return getWorkoutTypeConfig(value)?.pace
    ? _('Average pace (no pause)')
    : _('Average speed (no pause)');
};
