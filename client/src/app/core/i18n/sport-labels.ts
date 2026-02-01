import { _ } from '@ngx-translate/core';

const SPORT_LABELS: Record<string, string> = {
  auto: _('auto-detect'),
  unknown: _('unknown'),
  cycling: _('cycling'),
  'e-cycling': _('e-cycling'),
  'horse-riding': _('horse riding'),
  'inline-skating': _('inline skating'),
  golfing: _('golfing'),
  hiking: _('hiking'),
  'push-ups': _('push-ups'),
  running: _('running'),
  skiing: _('skiing'),
  snowboarding: _('snowboarding'),
  swimming: _('swimming'),
  walking: _('walking'),
  'weight-lifting': _('weight lifting'),
  kayaking: _('kayaking'),
  rowing: _('rowing'),
  other: _('other'),
};

const SPORT_SUBTYPE_LABELS: Record<string, string> = {
  virtual_activity: _('virtual activity'),
};

export const getSportLabel = (value?: string | null): string => {
  if (!value) {
    return '';
  }
  return SPORT_LABELS[value] ?? value;
};

export const getSportSubtypeLabel = (value?: string | null): string => {
  if (!value) {
    return '';
  }
  return SPORT_SUBTYPE_LABELS[value] ?? value;
};
