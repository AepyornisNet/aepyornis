export type MetricDef = {
  key: string;
  icon: string;
  color: string; // HEX color for charts
  colorClass: string; // Custom metric color class name (e.g. 'metric-distance')
};

export const METRIC_DEFS: Record<string, MetricDef> = {
  distance: {
    key: 'distance',
    icon: 'distance',
    color: '#3b82f6', // blue
    colorClass: 'metric-distance',
  },
  'distance-summary': {
    key: 'distance-summary',
    icon: 'distance',
    color: '#3b82f6', // blue
    colorClass: 'metric-distance',
  },
  duration: {
    key: 'duration',
    icon: 'duration',
    color: '#10b981', // green
    colorClass: 'metric-duration',
  },
  speed: {
    key: 'speed',
    icon: 'speed',
    color: '#06b6d4', // cyan
    colorClass: 'metric-speed',
  },
  'average-speed': {
    key: 'average-speed',
    icon: 'speed',
    color: '#06b6d4',
    colorClass: 'metric-speed',
  },
  'max-speed': {
    key: 'max-speed',
    icon: 'max-speed',
    color: '#06b6d4',
    colorClass: 'metric-speed',
  },
  elevation: {
    key: 'elevation',
    icon: 'elevation',
    color: '#10b981', // green
    colorClass: 'metric-elevation',
  },
  'elevation-summary': {
    key: 'elevation-summary',
    icon: 'elevation',
    color: '#10b981', // green
    colorClass: 'metric-elevation',
  },
  slope: {
    key: 'slope',
    icon: 'slope',
    color: '#10b981', // green
    colorClass: 'metric-slope',
  },
  'heart-rate': {
    key: 'heart-rate',
    icon: 'heart-rate',
    color: '#ef4444', // red
    colorClass: 'metric-heart-rate',
  },
  'respiration-rate': {
    key: 'respiration-rate',
    icon: 'respiration-rate',
    color: '#f97316', // orange
    colorClass: 'metric-respiration-rate',
  },
  cadence: {
    key: 'cadence',
    icon: 'cadence',
    color: '#f59e0b', // amber
    colorClass: 'metric-cadence',
  },
  power: {
    key: 'power',
    icon: 'power',
    color: '#3b82f6', // blue
    colorClass: 'metric-power',
  },
  temperature: {
    key: 'temperature',
    icon: 'temperature',
    color: '#06b6d4', // cyan
    colorClass: 'metric-temperature',
  },
  repetitions: {
    key: 'repetitions',
    icon: 'repetitions',
    color: '#f59e0b', // amber
    colorClass: 'metric-repetitions',
  },
  weight: {
    key: 'weight',
    icon: 'weight',
    color: '#ef4444', // red
    colorClass: 'metric-weight',
  },
};

const FALLBACK_PALETTE = [
  '#f59e0b',
  '#10b981',
  '#3b82f6',
  '#8b5cf6',
  '#ec4899',
  '#f97316',
  '#a855f7',
];

export function getMetricDef(key: string, fallbackIdx?: number): MetricDef {
  if (METRIC_DEFS[key]) {
    return METRIC_DEFS[key];
  }

  const color = FALLBACK_PALETTE[(fallbackIdx ?? 0) % FALLBACK_PALETTE.length];
  return {
    key,
    icon: key,
    color,
    colorClass: 'metric-fallback',
  };
}
